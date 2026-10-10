#!/bin/bash
# Go through Annie OS's first start like a person would, in a virtual machine:
#   1. the Welcome window asks to create a username and password;
#   2. press Create, type a username, type a password twice;
#   3. the login screen appears;
#   4. type the password there, and the desktop opens as the new user.
#
#   os/tests/account-flow.sh qemu ISO OUTDIR
#   os/tests/account-flow.sh vbox ISO OUTDIR
#
# qemu boots the ISO's kernel directly (with the console on a serial log);
# vbox boots the ISO in VirtualBox through its own UEFI boot menu, as a real
# computer would. Each step is checked by reading the screen (tesseract OCR).
# Leaves screenshots (step-*.png), ocr-*.txt, summary.txt and the logs in
# OUTDIR, and exits non-zero if any step failed.
set -uo pipefail

mode="${1:?usage: account-flow.sh qemu|boxes|vbox ISO OUTDIR}"
iso="$(realpath "${2:?}")"
out="$(realpath -m "${3:?}")"
readonly USERNAME=tester PASSWORD=annie123
mkdir -p "$out"
rm -f "$out"/step-*.png "$out"/ocr-*.txt "$out/summary.txt"

results=()
pass() { results+=("ok    $1"); echo "ok    $1"; }
fail() { results+=("FAIL  $1"); echo "FAIL  $1"; }

# --- The virtual machine -------------------------------------------------------
vm=annie-test
start_vm() {
    case "$mode" in
    qemu | boxes)
        local boot="$out/bootfiles" entry options accel=tcg cpu=max
        # qemu: a modern PC (q35) with virtio graphics, like the tests always
        # used. boxes: what GNOME Boxes gives an "unknown" system, an older
        # PC (i440fx) with QXL graphics and a 30 GB disk, with everything the
        # system logs copied to the serial log to see why it fails.
        local machine=q35 gfx=(-device virtio-vga) extra=() append=""
        if [[ "$mode" == boxes ]]; then
            machine=pc gfx=(-vga qxl)
            qemu-img create -f qcow2 "$out/disk.qcow2" 30G >/dev/null
            extra=(-drive "file=$out/disk.qcow2,if=virtio")
            # copytoram=y: what archiso does on its own on computers with
            # plenty of RAM (it then lets go of the boot drive), like the
            # Steam Machine.
            append="systemd.journald.forward_to_console=1 copytoram=y"
        fi
        rm -rf "$boot"
        mkdir -p "$boot"
        xorriso -osirrox on -indev "$iso" \
            -extract /arch/boot/x86_64/vmlinuz-linux "$boot/vmlinuz-linux" \
            -extract /arch/boot/x86_64/initramfs-linux.img "$boot/initramfs-linux.img" \
            -extract /loader/entries "$boot/entries" >/dev/null 2>&1
        entry="$(ls "$boot"/entries/*.conf | head -n1)"
        options="$(awk '$1 == "options" {$1 = ""; print substr($0, 2)}' "$entry")"
        if [[ -w /dev/kvm ]]; then accel=kvm cpu=host; fi
        mon="$out/monitor.sock"
        rm -f "$mon"
        qemu-system-x86_64 \
            -machine $machine,accel=$accel -cpu $cpu -smp 2 -m 4096 \
            -kernel "$boot/vmlinuz-linux" -initrd "$boot/initramfs-linux.img" \
            -append "$options console=tty0 console=ttyS0,115200 $append" \
            -cdrom "$iso" "${gfx[@]}" "${extra[@]}" -display none \
            -serial file:"$out/serial.log" \
            -monitor unix:"$mon",server,nowait \
            -nic user,model=virtio-net-pci &
        qemu=$!
        ;;
    vbox)
        sudo VBoxManage unregistervm "$vm" --delete >/dev/null 2>&1 || true
        sudo VBoxManage createvm --name "$vm" --ostype ArchLinux_64 --register >/dev/null
        sudo VBoxManage modifyvm "$vm" --memory 4096 --cpus 2 --firmware efi \
            --graphicscontroller vmsvga --vram 128 --nic1 nat --audio-enabled off \
            --uart1 0x3F8 4 --uartmode1 file "$out/vbox-serial.log" >/dev/null
        sudo VBoxManage storagectl "$vm" --name SATA --add sata --portcount 2 >/dev/null
        sudo VBoxManage storageattach "$vm" --storagectl SATA --port 0 --type dvddrive --medium "$iso" >/dev/null
        sudo VBoxManage startvm "$vm" --type headless
        ;;
    esac
}
stop_vm() {
    case "$mode" in
    qemu | boxes) kill "$qemu" 2>/dev/null || true ;;
    vbox)
        timeout 60 sudo VBoxManage controlvm "$vm" poweroff >/dev/null 2>&1 || true
        sleep 3
        cp "$(sudo VBoxManage showvminfo "$vm" --machinereadable | sed -n 's/^LogFldr="\(.*\)"$/\1/p')/VBox.log" "$out/VBox.log" 2>/dev/null ||
            sudo find /root/VirtualBox\ VMs/"$vm" -name VBox.log -exec cp {} "$out/VBox.log" \; 2>/dev/null || true
        sudo chmod a+r "$out"/* 2>/dev/null || true
        ;;
    esac
}
trap stop_vm EXIT

screenshot() { # screenshot FILE.png
    case "$mode" in
    qemu | boxes)
        local ppm="${1%.png}.ppm"
        printf 'screendump %s\n' "$ppm" | socat - UNIX-CONNECT:"$mon" >/dev/null 2>&1
        sleep 1
        [[ -f "$ppm" ]] && convert "$ppm" "$1" && rm -f "$ppm"
        ;;
    vbox)
        timeout 60 sudo VBoxManage controlvm "$vm" screenshotpng "$1" >/dev/null 2>&1
        sudo chmod a+r "$1" 2>/dev/null
        ;;
    esac
    [[ -s "$1" ]]
}

# Keys. In QEMU through its monitor; in VirtualBox as keyboard scancodes.
press_enter() {
    case "$mode" in
    qemu | boxes) printf 'sendkey ret\n' | socat - UNIX-CONNECT:"$mon" >/dev/null 2>&1 ;;
    vbox) timeout 60 sudo VBoxManage controlvm "$vm" keyboardputscancode 1c 9c ;;
    esac
}
type_text() { # type_text TEXT (small letters and digits)
    local i c
    case "$mode" in
    qemu | boxes)
        for ((i = 0; i < ${#1}; i++)); do
            c="${1:i:1}"
            printf 'sendkey %s\n' "$c" | socat - UNIX-CONNECT:"$mon" >/dev/null 2>&1
            sleep 0.15
        done
        ;;
    vbox) timeout 60 sudo VBoxManage controlvm "$vm" keyboardputstring "$1" ;;
    esac
}

# Read the screen; wait up to SECONDS for TEXT (a regular expression) to show.
echo 0 >"$out/.count"
read_screen() { # read_screen NAME -> prints the text on the screen
    local png n
    n="$(cat "$out/.count")"
    echo $((n + 1)) >"$out/.count"
    png="$out/step-$(printf %03d "$n")-$1.png"
    screenshot "$png" || return 1
    # A larger, grey picture reads better.
    convert "$png" -colorspace Gray -resize 200% "$out/ocr-in.png" 2>/dev/null
    tesseract "$out/ocr-in.png" - 2>/dev/null | tee "$out/ocr-$(basename "${png%.png}").txt"
}
wait_for() { # wait_for NAME SECONDS REGEX
    local name="$1" secs="$2" re="$3" t
    for ((t = 0; t <= secs; t += 10)); do
        if read_screen "$name" | grep -qiE "$re"; then
            return 0
        fi
        sleep 10
    done
    return 1
}

# --- The steps -----------------------------------------------------------------
start_vm
echo "started the virtual machine ($mode)"

if wait_for welcome 420 'Welcome to Annie|Create your username'; then
    pass "first start shows the Welcome window asking to create a username and password"
else
    fail "first start shows the Welcome window asking to create a username and password"
fi

sleep 3
press_enter # Create
if wait_for username 40 'Username'; then pass "Create asks for a username"; else fail "Create asks for a username"; fi

type_text "$USERNAME"
sleep 1
press_enter
if wait_for password 40 "Password for|$USERNAME"; then pass "then asks for a password"; else fail "then asks for a password"; fi

type_text "$PASSWORD"
sleep 1
press_enter
if wait_for again 40 'same password|again'; then pass "then asks for the password again"; else fail "then asks for the password again"; fi

type_text "$PASSWORD"
sleep 1
press_enter
if wait_for allset 60 'All set|login screen'; then pass "says the account is made"; else fail "says the account is made"; fi

# Press OK on "All set!". The login screen is recognised by its own buttons
# (the "All set!" box itself says "Log in as ..."), and OK is pressed again
# if the first press came before the box could take it.
at_login=false
for _ in 1 2 3; do
    sleep 3
    press_enter
    if wait_for login 60 'Shut ?Down|Virtual Keyboard|Restart'; then
        at_login=true
        break
    fi
done
if $at_login; then
    pass "Annie OS goes to the login screen, showing $USERNAME"
else
    fail "Annie OS goes to the login screen, showing $USERNAME"
fi

sleep 5
type_text "$PASSWORD"
sleep 1
press_enter
# The desktop: the bar's search box or the wallpaper's name, and no
# password box any more.
if wait_for desktop 180 '(Search|Google|Annie ?OS)' && ! read_screen desktop-check | grep -qiE 'Password'; then
    pass "logging in with the new password opens the desktop"
else
    fail "logging in with the new password opens the desktop"
fi
# The setup account's desktop has the Install Annie OS icon; the new
# account's does not, which shows this desktop is really $USERNAME's.
if read_screen whose-desktop | grep -qiE 'Install Annie'; then
    fail "the desktop belongs to $USERNAME, not the setup account"
else
    pass "the desktop belongs to $USERNAME, not the setup account"
fi
# The Welcome window must not come back for the new account.
sleep 20
if read_screen after | grep -qiE 'Welcome to Annie|Create your username'; then
    fail "the Welcome window does not ask again"
else
    pass "the Welcome window does not ask again"
fi

stop_vm
trap - EXIT
{
    echo "== First start, in $mode"
    printf '%s\n' "${results[@]}"
    if [[ -f "$out/serial.log" ]]; then
        echo
        echo "== Annie OS services"
        sed 's/\x1b\[[0-9;]*m//g' "$out/serial.log" | grep -aE 'annie-|Display Manager|Session [0-9]+ of User' | sort -u | head -n 40 || true
        echo
        echo "== Graphics and login screen problems"
        sed 's/\x1b\[[0-9;]*m//g' "$out/serial.log" | tr -d '\r' |
            grep -aiE 'sddm|kwin|drm|qxl|virtio.?gpu|wayland|failed|error|segfault|core dump' |
            grep -viE 'audit:|ACPI' | head -n 120 || true
    fi
    if [[ -f "$out/VBox.log" ]]; then
        echo
        echo "== VirtualBox: problems in VBox.log"
        grep -aiE 'error|fail|guru|VERR_' "$out/VBox.log" | grep -viE 'VERR_NOT_SUPPORTED|VERR_FILE_NOT_FOUND|fUnrestricted' | head -n 40 || true
    fi
} >"$out/summary.txt"
cat "$out/summary.txt"
! printf '%s\n' "${results[@]}" | grep -q '^FAIL'

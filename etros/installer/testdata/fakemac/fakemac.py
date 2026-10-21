#!/usr/bin/env python3
# Stand-ins for macOS's diskutil, osascript, networksetup, sw_vers and
# sysctl, for testing Etros Mac Installer's real code on Linux
# (mac_tools_test.go). The drives live in the JSON file $FAKEMAC_STATE:
#   {"disks": [{"id": "disk0", "name": "...", "size": N, "internal": true,
#               "parts": [{"num": 1, "content": "EFI", "name": "EFI",
#                          "start": N, "size": N}, ...]}],
#    "boot_store": "disk0s2", "apfs_free": N, "need_admin": false,
#    "cancel_password": false, "volumes": "/dir/for/mount/points",
#    "wifi": "Home", "busy": ["disk0s3"]}
# A "busy" partition (open in Finder, indexed by Spotlight...) can't be
# erased until it's unmounted with force; erasing one deletes what was in
# its mount point. "stuck": {"disk0s3": N} makes the next N erases of it
# fail as being used even so (macOS still checking it); "procs" is what ps
# shows, "open": {"disk0s3": ["Finder"]} what lsof shows has it open.
# Every call is logged to $FAKEMAC_STATE.log.
import json, os, plistlib, re, sys

MIB = 1 << 20
state_path = os.environ["FAKEMAC_STATE"]
with open(state_path) as f:
    st = json.load(f)


def save():
    with open(state_path, "w") as f:
        json.dump(st, f, indent=1)


def log(*a):
    with open(state_path + ".log", "a") as f:
        f.write(" ".join(a) + "\n")


def die(msg, code=1):
    print(msg, file=sys.stderr)
    sys.exit(code)


def pid(d, p):
    return "%ss%d" % (d["id"], p["num"])


def find(ident):
    for d in st["disks"]:
        if d["id"] == ident:
            return d, None
        for p in d["parts"]:
            if pid(d, p) == ident:
                return d, p
    return None, None


def size_arg(s, avail=None):
    s = s.strip()
    if s in ("R", "0") and avail is not None:
        return avail
    m = re.fullmatch(r"([0-9.]+)([BSKMGT]?)", s, re.I)
    if not m:
        die("Error: -69734: invalid size %r" % s)
    mult = {"": 1, "B": 1, "S": 512, "K": 1 << 10, "M": MIB, "G": 1 << 30, "T": 1 << 40}[m.group(2).upper()]
    return int(float(m.group(1)) * mult)


def content_of(fmt):
    return {"MS-DOS FAT32": "Microsoft Basic Data", "ExFAT": "Microsoft Basic Data", "JHFS+": "Apple_HFS",
            "APFS": "Apple_APFS"}.get(fmt) or die("Error: -69882: unknown file system %r" % fmt)


def mount(d, p):
    if p["content"] == "Microsoft Basic Data" and p.get("name"):
        mp = os.path.join(st["volumes"], p["name"])
        n = 1
        while os.path.exists(mp) and not p.get("mount") == mp and any(
                q.get("mount") == mp for dd in st["disks"] for q in dd["parts"] if q is not p):
            n += 1
            mp = os.path.join(st["volumes"], "%s %d" % (p["name"], n))
        os.makedirs(mp, exist_ok=True)
        p["mount"] = mp


def new_num(d):
    used = {p["num"] for p in d["parts"]}
    n = 1
    while n in used:
        n += 1
    return n


def admin_only():
    if st.get("need_admin") and os.environ.get("FAKEMAC_ADMIN") != "1":
        die("Error: -69888: Couldn't unmount disk. You may need to run this as root")


def diskutil(args):
    log("diskutil", *args)
    if args[:2] == ["list", "-plist"]:
        out = {"AllDisksAndPartitions": [], "WholeDisks": [d["id"] for d in st["disks"]]}
        for d in st["disks"]:
            parts = []
            for p in sorted(d["parts"], key=lambda p: p["start"]):
                e = {"DeviceIdentifier": pid(d, p), "Content": p["content"], "Size": p["size"]}
                if p.get("name"):
                    e["VolumeName"] = p["name"]
                if p.get("mount"):
                    e["MountPoint"] = p["mount"]
                parts.append(e)
            out["AllDisksAndPartitions"].append({"DeviceIdentifier": d["id"], "Size": d["size"],
                                                 "Content": "GUID_partition_scheme", "Partitions": parts})
        sys.stdout.buffer.write(plistlib.dumps(out))
        return
    if args[0] == "list" and len(args) == 2 and args[1] != "-plist":
        d, p = find(args[1])
        if d is None or p is not None:
            die("Could not find disk: " + args[1])
        print("/dev/%s (%s, physical):" % (d["id"], "internal" if d["internal"] else "external"))
        print("   #:                       TYPE NAME                    SIZE       IDENTIFIER")
        print("   0:      GUID_partition_scheme                        *%.1f GB   %s" % (d["size"] / 1e9, d["id"]))
        for q in sorted(d["parts"], key=lambda q: q["start"]):
            print("%4d:%27s %-23s %.1f GB    %s" % (q["num"], q["content"][:27], q.get("name", ""), q["size"] / 1e9, pid(d, q)))
        return
    if args[:2] == ["info", "-plist"]:
        ident = args[2]
        if ident == "/":
            sys.stdout.buffer.write(plistlib.dumps({"DeviceIdentifier": "disk1s5", "MountPoint": "/",
                                                    "APFSPhysicalStores": [{"APFSPhysicalStore": st["boot_store"]}]}))
            return
        d, p = find(ident)
        if d is None:
            die("Could not find disk: " + ident)
        if p is None:
            info = {"DeviceIdentifier": d["id"], "Internal": d["internal"], "MediaName": d["name"], "Size": d["size"]}
        else:
            info = {"DeviceIdentifier": ident, "Size": p["size"], "MountPoint": p.get("mount", "")}
        sys.stdout.buffer.write(plistlib.dumps(info))
        return
    if args[:3] == ["apfs", "list", "-plist"]:
        sys.stdout.buffer.write(plistlib.dumps({"Containers": [{"ContainerReference": "disk1",
            "CapacityFree": st["apfs_free"], "PhysicalStores": [{"DeviceIdentifier": st["boot_store"]}]}]}))
        return
    if args[0] in ("unmount", "unmountDisk"):
        ids = [args[-1]]
        d, p = find(args[-1])
        if d is None:
            die("Could not find disk: " + args[-1])
        if p is None:
            ids = [pid(d, q) for q in d["parts"]]
        if "force" not in args and any(i in (st.get("busy") or []) for i in ids):
            die("Unmount of %s failed: at least one volume could not be unmounted" % args[-1])
        st["busy"] = [b for b in (st.get("busy") or []) if b not in ids]
        save()
        print("Volume(s) on %s unmounted" % args[-1])
        return
    admin_only()
    if args[0] == "eraseVolume" and len(args) == 4:
        fmt, name, ident = args[1:]
        d, p = find(ident)
        if p is None:
            die("Could not find disk: " + ident)
        if pid(d, p) == st["boot_store"]:
            die("Error: -69888: Couldn't unmount disk (it's the startup disk)")
        stuck = st.get("stuck") or {}
        if stuck.get(ident, 0) > 0:
            stuck[ident] -= 1
            st["stuck"] = stuck
            save()
            die("Error: -69877: The target disk is being used")
        if ident in (st.get("busy") or []):
            die("0:182: execution error: Error: -69877: The target disk is being used")
        if fmt.lower() in ("free", "free space", "%noformat%") and name.lower() == "none":
            if p.get("mount"):
                import shutil
                shutil.rmtree(p["mount"], ignore_errors=True)
            d["parts"].remove(p)
            print("Finished erase on " + ident)
        else:
            die("Error: unsupported erase " + " ".join(args[1:]))
        save()
        return
    if args[0] == "addPartition" and len(args) == 5:
        after, fmt, name, size = args[1:]
        d, p = find(after)
        if p is None:
            die("Could not find disk: " + after)
        start = (p["start"] + p["size"] + MIB - 1) // MIB * MIB
        later = [q["start"] for q in d["parts"] if q["start"] > p["start"]]
        end = min(later) if later else d["size"] - MIB
        sz = size_arg(size, end - start)
        if sz <= 0 or start + sz > end:
            die("Error: -69519: The target disk is too small for this operation (%d bytes asked, %d free)" % (sz, end - start))
        q = {"num": new_num(d), "content": content_of(fmt), "name": name, "start": start, "size": sz}
        d["parts"].append(q)
        mount(d, q)
        print("Finished partitioning on " + d["id"])
        save()
        return
    if args[0] == "partitionDisk":
        d, p = find(args[1])
        if d is None or p is not None:
            die("Could not find disk: " + args[1])
        if d["internal"]:
            die("Error: -69888: Couldn't unmount disk (it's the startup disk)")
        if any(pid(d, q) in (st.get("busy") or []) for q in d["parts"]):
            die("Error: -69877: The target disk is being used")
        rest = args[3:]
        d["parts"] = [{"num": 1, "content": "EFI", "name": "EFI", "start": 20480 * 512, "size": 200 * MIB}]
        start = 20480 * 512 + 200 * MIB
        for i in range(0, len(rest), 3):
            fmt, name, size = rest[i:i + 3]
            sz = size_arg(size, d["size"] - MIB - start)
            q = {"num": new_num(d), "content": content_of(fmt), "name": name, "start": start, "size": sz}
            d["parts"].append(q)
            mount(d, q)
            start += sz
        print("Finished partitioning on " + d["id"])
        save()
        return
    if args[:2] == ["apfs", "resizeContainer"] and len(args) == 4:
        d, p = find(args[2])
        if p is None or p["content"] != "Apple_APFS":
            die("Error: not an APFS physical store")
        new = size_arg(args[3])
        if p["size"] - new > st["apfs_free"]:
            die("Error: -69519: The target disk is too small for this operation")
        st["apfs_free"] -= p["size"] - new
        p["size"] = new
        print("Finished APFS operation")
        save()
        return
    if args[0] == "mount":
        d, p = find(args[1])
        if p:
            mount(d, p)
            save()
        return
    die("fake diskutil doesn't know: " + " ".join(args))


def osascript(args):
    log("osascript", *args)
    script = args[args.index("-e") + 1]
    m = re.match(r'do shell script "(.*)" with administrator privileges', script, re.S)
    if not m:
        die("fake osascript doesn't know: " + script)
    if st.get("cancel_password"):
        die("0:203: execution error: User canceled. (-128)")
    cmd = m.group(1).encode().decode("unicode_escape")
    env = dict(os.environ, FAKEMAC_ADMIN="1")
    os.execvpe("sh", ["sh", "-c", cmd], env)


tool = os.path.basename(sys.argv[0])
args = sys.argv[1:]
if tool == "diskutil":
    diskutil(args)
elif tool == "osascript":
    osascript(args)
elif tool == "networksetup":
    if args == ["-listallhardwareports"]:
        print("\nHardware Port: Wi-Fi\nDevice: en0\nEthernet Address: aa:bb\n")
    elif args[0] == "-getairportnetwork":
        print("Current Wi-Fi Network: " + st.get("wifi", "") if st.get("wifi") else "You are not associated with an AirPort network.")
elif tool == "ps":
    print("\n".join(["/sbin/launchd", "/usr/libexec/mds"] + (st.get("procs") or [])))
elif tool == "lsof":
    dev = args[-1].replace("/dev/", "")
    for i, name in enumerate((st.get("open") or {}).get(dev, [])):
        print("p%d\nc%s" % (100 + i, name))
elif tool == "sw_vers":
    print("11.7.11")
elif tool == "sysctl":
    print("Intel(R) Core(TM) M-5Y31 CPU @ 0.90GHz")
else:
    die("fakemac: unknown tool " + tool)

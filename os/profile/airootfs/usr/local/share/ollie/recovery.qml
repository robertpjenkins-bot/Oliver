// Recovery Mode (ollie-recovery): pick an ISO file and press Write to put it
// over this Ollie OS, for getting newer versions. Then: is it an Ollie OS
// version? If yes, keep your files, username and passwords, Wi-Fi and
// website logins (everything comes along to the new version except the OS
// itself), or start fresh. If not, it erases the whole drive and writes the
// ISO over it. Starting fresh and erasing need the parent PIN, if there is one.
// Says on the console: OLLIE-ACTION start keep|fresh|erase FILE, restart.
// Reads how it goes from the progress file (recovery-helper's lines).
// Arguments: progress=FILE, pin= (the parent PIN's check sum), home=DIR.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Dialogs

ApplicationWindow {
    id: win
    width: 760
    height: 560
    visible: true
    title: "Recovery Mode"
    color: "#0f172a"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }
    function md5(pin) { return Qt.md5("ollie-os:" + pin) }

    property string page: "pick"      // pick, ask, keep, erase, working, done, error, isollie, notollie
    property string iso: ""
    property string mode: ""
    property int percent: 0
    property string step: ""
    property string problem: ""
    readonly property string pinSum: arg("pin")
    readonly property string isoName: iso.substring(iso.lastIndexOf("/") + 1)

    function start(m) {
        if ((m === "fresh" || m === "erase") && pinSum !== "" && md5(pinField.text) !== pinSum) {
            pinWrong.visible = true; return;
        }
        mode = m; percent = 0; step = "Starting"; page = "working"; pinField.text = "";
        act("start " + m + " " + iso);
        poll.start();
    }
    Timer {
        id: poll
        interval: 500; repeat: true
        onTriggered: {
            const x = new XMLHttpRequest();
            x.open("GET", "file://" + win.arg("progress"), false);
            try { x.send(); } catch (e) { return; }
            for (const line of x.responseText.split(/[\r\n]+/)) {
                if (line.indexOf("P ") === 0) {
                    const m = line.match(/^P (\d+) (.*)$/);
                    if (m) { win.percent = parseInt(m[1]); win.step = m[2]; }
                } else if (line === "DONE") { win.page = "done"; poll.stop(); }
                else if (line === "ISOLLIE") { win.page = "isollie"; poll.stop(); }
                else if (line === "NOTOLLIE") { win.page = "notollie"; poll.stop(); }
                else if (line.indexOf("ERROR ") === 0) { win.problem = line.substring(6); win.page = "error"; poll.stop(); }
                else {
                    // dd's own progress while erasing: "1234567 bytes (...) copied"
                    const b = line.match(/^(\d+) bytes/);
                    if (b && win.mode === "erase" && win.isoSize > 0) win.percent = Math.min(99, 15 + Math.floor(parseInt(b[1]) * 85 / win.isoSize));
                }
            }
        }
    }
    property real isoSize: 0

    FileDialog {
        id: picker
        title: "Pick the ISO file"
        nameFilters: ["ISO files (*.iso)", "All files (*)"]
        currentFolder: "file://" + win.arg("home") + "/Downloads"
        onAccepted: {
            let f = selectedFile.toString();
            if (f.indexOf("file://") === 0) f = decodeURIComponent(f.substring(7));
            win.iso = f;
        }
    }

    component Big: Button {
        font.pixelSize: 19
        Layout.preferredHeight: 58
        Layout.fillWidth: true
    }
    component Words: Label {
        color: "#e2e8f0"; font.pixelSize: 17; wrapMode: Text.WordWrap
        Layout.fillWidth: true; horizontalAlignment: Text.AlignHCenter
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 30
        spacing: 16

        RowLayout {
            Layout.alignment: Qt.AlignHCenter
            spacing: 12
            Label { text: "🛟"; font.pixelSize: 44 }
            Label { text: "Recovery Mode"; color: "white"; font.pixelSize: 34; font.bold: true }
        }
        Item { Layout.fillHeight: true }

        // --- Pick the ISO ---
        ColumnLayout {
            visible: win.page === "pick"
            spacing: 14
            Words { text: "Put a newer Ollie OS over this one: pick its .iso file, then press Write." }
            Big { text: "📂 Choose an ISO file..."; onClicked: picker.open() }
            Words { visible: win.iso !== ""; text: "💿 " + win.isoName; font.bold: true; color: "#5eead4" }
            Big { text: "✍️ Write"; highlighted: true; enabled: win.iso !== ""; onClicked: win.page = "ask" }
        }

        // --- Is it Ollie OS? ---
        ColumnLayout {
            visible: win.page === "ask"
            spacing: 14
            Words { text: "Is " + win.isoName + " an Ollie OS version?"; font.pixelSize: 22; font.bold: true }
            RowLayout {
                spacing: 14
                Big { text: "✅ Yes, it's Ollie OS"; highlighted: true; onClicked: win.page = "keep" }
                Big { text: "❌ No"; onClicked: win.page = "erase" }
            }
            Button { text: "← Back"; Layout.alignment: Qt.AlignHCenter; onClicked: win.page = "pick" }
        }

        // --- Keep everything? ---
        ColumnLayout {
            visible: win.page === "keep"
            spacing: 14
            Words { text: "Keep your files, username and password, Wi-Fi passwords and website logins?"; font.pixelSize: 21; font.bold: true }
            Words { text: "Yes takes everything with you to the new version: only Ollie OS itself is new. Your games, pictures, settings and apps stay."; font.pixelSize: 15; color: "#94a3b8" }
            Big { text: "💾 Yes, keep everything"; highlighted: true; onClicked: win.start("keep") }
            Big { text: "🧹 No, start fresh (your files are erased)"; onClicked: win.start("fresh") }
            Button { text: "← Back"; Layout.alignment: Qt.AlignHCenter; onClicked: win.page = "ask" }
        }

        // --- Erase everything ---
        ColumnLayout {
            visible: win.page === "erase"
            spacing: 14
            Words { text: "⚠️ This ERASES the whole drive: Ollie OS and all your files!"; font.pixelSize: 22; font.bold: true; color: "#fca5a5" }
            Words { text: win.isoName + " is written over the drive, then the computer starts from it. Only ISOs that can start from a USB stick (Linux) work this way."; font.pixelSize: 15; color: "#94a3b8" }
            Big { text: "💥 Erase and write"; onClicked: win.start("erase") }
            Button { text: "← Back"; Layout.alignment: Qt.AlignHCenter; onClicked: win.page = "ask" }
        }

        // The parent PIN, for starting fresh and erasing.
        ColumnLayout {
            visible: win.pinSum !== "" && (win.page === "keep" || win.page === "erase")
            Layout.alignment: Qt.AlignHCenter
            spacing: 4
            TextField {
                id: pinField
                Layout.alignment: Qt.AlignHCenter
                Layout.preferredWidth: 280
                echoMode: TextInput.Password
                placeholderText: win.page === "erase" ? "Parent PIN (to erase)" : "Parent PIN (to start fresh)"
                inputMethodHints: Qt.ImhDigitsOnly
                onTextEdited: pinWrong.visible = false
            }
            Label { id: pinWrong; visible: false; text: "That PIN is not right."; color: "#fca5a5"; Layout.alignment: Qt.AlignHCenter }
        }

        // --- Working ---
        ColumnLayout {
            visible: win.page === "working"
            spacing: 14
            Words { text: win.step; font.pixelSize: 19 }
            ProgressBar { Layout.fillWidth: true; from: 0; to: 100; value: win.percent }
            Words { text: win.percent + "%   Don't turn the computer off!"; color: "#94a3b8"; font.pixelSize: 15 }
        }

        // --- Done ---
        ColumnLayout {
            visible: win.page === "done"
            spacing: 14
            Words { text: "🎉 The new Ollie OS is ready!"; font.pixelSize: 24; font.bold: true; color: "#5eead4" }
            Words {
                text: win.mode === "keep" ? "Restart to start it. Everything of yours comes along. (If something is wrong with it, pick \"Ollie OS (before the update)\" on the start-up menu in the first 10 minutes.)"
                    : "Restart to start it. It asks you to make a username and password, like the first time."
                font.pixelSize: 15; color: "#94a3b8"
            }
            Big { text: "🔄 Restart now"; highlighted: true; onClicked: win.act("restart") }
        }

        // --- Problems ---
        ColumnLayout {
            visible: win.page === "error" || win.page === "isollie" || win.page === "notollie"
            spacing: 14
            Words {
                text: win.page === "isollie" ? "😮 Wait: " + win.isoName + " IS Ollie OS! Pick Yes, and you can keep all your files."
                    : win.page === "notollie" ? "🤔 " + win.isoName + " is not Ollie OS, so your files can't be brought along."
                    : "😕 " + win.problem
                font.pixelSize: 19
            }
            Words { visible: win.page === "error"; text: "Nothing was changed: this Ollie OS is just as it was."; font.pixelSize: 15; color: "#94a3b8" }
            RowLayout {
                spacing: 14
                Big { visible: win.page === "isollie"; text: "✅ Yes, it's Ollie OS"; highlighted: true; onClicked: win.page = "keep" }
                Big { visible: win.page === "isollie"; text: "💥 Erase anyway"; onClicked: win.page = "erase" }
                Big { visible: win.page === "notollie"; text: "💥 Erase and write it"; onClicked: win.page = "erase" }
                Big { text: "← Start again"; onClicked: { win.page = "pick"; win.problem = ""; } }
            }
        }
        Item { Layout.fillHeight: true }
    }
}

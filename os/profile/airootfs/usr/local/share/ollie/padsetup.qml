// The first time a controller is ever plugged in (ollie-gamehub --setup,
// opened by ollie-padd): which button should open the Game Hub? Press it,
// then press it again to choose it (so it all works with just the
// controller); the mouse works too. Also: the controller as a mouse on the
// desktop, yes or no. Says on the console: OLLIE-ACTION set CODE on|off.
// Arguments: learn=FILE (ollie-padd writes the last button pressed:
// {"code", "name", "seq"}).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 760
    height: 560
    visible: true
    title: "A controller!"
    color: "#0f172a"
    flags: Qt.Window | Qt.WindowStaysOnTopHint

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    property int code: 0
    property string name: ""
    property int seq: 0
    property int presses: 0
    property bool chosen: false
    function choose() {
        if (code === 0) return;
        chosen = true;
        console.log("OLLIE-ACTION set " + code + " " + (mouse.checked ? "on" : "off"));
        done.start();
    }
    Timer { id: done; interval: 2200; onTriggered: Qt.quit() }
    Timer {
        interval: 100; running: !win.chosen; repeat: true
        onTriggered: {
            const x = new XMLHttpRequest();
            x.open("GET", "file://" + win.arg("learn"), false);
            let d;
            try { x.send(); d = JSON.parse(x.responseText); } catch (e) { return; }
            if (d.seq === win.seq) return;
            win.seq = d.seq;
            if (d.code === win.code) { win.presses++; if (win.presses >= 2) win.choose(); }
            else { win.code = d.code; win.name = d.name; win.presses = 1; }
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 30
        spacing: 16
        Label { text: "🎮"; font.pixelSize: 80; Layout.alignment: Qt.AlignHCenter }
        Label { text: win.chosen ? "All set! 🎉" : "A controller! Hi!"; color: "white"; font.pixelSize: 34; font.bold: true; Layout.alignment: Qt.AlignHCenter }
        Label {
            Layout.alignment: Qt.AlignHCenter
            Layout.maximumWidth: 640
            wrapMode: Text.WordWrap
            horizontalAlignment: Text.AlignHCenter
            color: "#cbd5e1"; font.pixelSize: 20
            text: win.chosen ? "Press " + win.name + " any time to open the Game Hub. In a game, it opens the Quick Menu."
                : win.code === 0 ? "Which button should open the Game Hub?\nPress it on your controller now.\n(The Home button, in the middle, is a good one.)"
                : "You pressed: " + win.name + "\nPress it again to choose it, or press a different one."
        }
        Rectangle {
            visible: win.code !== 0
            Layout.alignment: Qt.AlignHCenter
            width: label.width + 60; height: 70; radius: 35
            color: win.chosen ? "#16a34a" : "#facc15"
            Label { id: label; anchors.centerIn: parent; text: win.name; font.pixelSize: 26; font.bold: true; color: win.chosen ? "white" : "#422006" }
        }
        Item { Layout.fillHeight: true }
        CheckBox {
            id: mouse
            visible: !win.chosen
            Layout.alignment: Qt.AlignHCenter
            checked: true
            text: "Use the controller as a mouse on the desktop"
            palette.windowText: "white"
            font.pixelSize: 16
        }
        RowLayout {
            visible: !win.chosen
            Layout.alignment: Qt.AlignHCenter
            spacing: 12
            Button { text: "✅ Use " + (win.name || "this button"); enabled: win.code !== 0; highlighted: true; font.pixelSize: 16; onClicked: win.choose() }
            Button { text: "Later"; font.pixelSize: 16; onClicked: Qt.quit() }
        }
    }
}

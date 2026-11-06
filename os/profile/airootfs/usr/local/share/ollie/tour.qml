// Ollie's Tour (ollie-tour): the first time Ollie OS starts, Ollie shows you
// around: the screen goes dim except for one part at a time (the menu, the
// search box, your apps, the clock, the fun fact...), and Ollie says what it
// is. Next and Back (or the arrows, Enter, the controller), Skip (Escape).
// Over everything (Plasma's layer shell).
// Arguments: name= (yours), events=FILE (ollie-padd's controller buttons).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.layershell 1.0 as LayerShell

Window {
    id: win
    visible: true
    color: "transparent"
    title: "Ollie's Tour"
    LayerShell.Window.scope: "ollie-tour"
    LayerShell.Window.layer: LayerShell.Window.LayerOverlay
    LayerShell.Window.anchors: LayerShell.Window.AnchorTop | LayerShell.Window.AnchorBottom | LayerShell.Window.AnchorLeft | LayerShell.Window.AnchorRight
    LayerShell.Window.keyboardInteractivity: LayerShell.Window.KeyboardInteractivityExclusive
    LayerShell.Window.exclusionZone: -1
    width: Screen.width
    height: Screen.height

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    readonly property real bar: 52       // the bar along the bottom
    readonly property real sw: width
    readonly property real sh: height
    // [what Ollie says, where (x, y, size of the light; size 0: no light)]
    readonly property var steps: [
        ["Hi" + (arg("name") ? " " + arg("name") : "") + "! I'm Ollie! 👋 Let me show you around your new computer.", sw / 2, sh / 2, 0],
        ["This is the Ollie menu. Click the cube to find all your apps: games, drawing, music and lots more!", 26, sh - bar / 2, 46],
        ["Type in this box to search the internet with Google.", 250, sh - bar / 2, 200],
        ["These are your favourite apps: Firefox, Steam, your files and more. Click one to open it.", sw - 560, sh - bar / 2, 210],
        ["Here are the time, the Wi-Fi, the sound and the battery.", sw - 120, sh - bar / 2, 130],
        ["Every day there's a new fun fact up here. Press Another one for more.", sw - 240, 145, 220],
        ["Got a controller? Plug it in and press its Home button for the Game Hub. 🎮", sw / 2, sh / 2, 0],
        ["Play Ollie OS's games and apps to win trophies! 🏆 Find Trophies in the menu.", sw / 2, sh / 2, 0],
        ["That's it! Open Ollie OS Welcome from the menu any time you need help. Have fun! 🎉", sw / 2, sh / 2, 0]
    ]
    property int at: 0
    readonly property var cur: steps[at]
    function go(w) {
        if (w === "next" || w === "ok" || w === "right") {
            if (at === steps.length - 1) { Qt.quit(); return; }
            at++;
        } else if (w === "back" && at > 0) at--;
        else if (w === "left" && at > 0) at--;
        else if (w === "skip") Qt.quit();
    }
    onAtChanged: { canvas.requestPaint(); if (at === steps.length - 1) confetti.burst(); }

    // The screen, dim except for the light.
    Canvas {
        id: canvas
        anchors.fill: parent
        property real lx: win.cur[1]
        property real ly: win.cur[2]
        property real lr: win.cur[3]
        Behavior on lx { NumberAnimation { duration: 400; easing.type: Easing.InOutQuad } }
        Behavior on ly { NumberAnimation { duration: 400; easing.type: Easing.InOutQuad } }
        Behavior on lr { NumberAnimation { duration: 400 } }
        onLxChanged: requestPaint()
        onLyChanged: requestPaint()
        onLrChanged: requestPaint()
        onPaint: {
            const c = getContext("2d");
            c.reset();
            c.fillStyle = "rgba(2, 6, 23, 0.72)";
            c.fillRect(0, 0, width, height);
            if (lr > 2) {
                c.globalCompositeOperation = "destination-out";
                c.beginPath();
                c.ellipse(lx - lr, ly - lr * 0.55, lr * 2, lr * 1.1);
                c.fill();
                c.globalCompositeOperation = "source-over";
                c.strokeStyle = "#facc15"; c.lineWidth = 5;
                c.beginPath();
                c.ellipse(lx - lr, ly - lr * 0.55, lr * 2, lr * 1.1);
                c.stroke();
            }
        }
    }

    // Ollie and what Ollie says, near the light (or in the middle).
    Item {
        id: talk
        readonly property bool middle: win.cur[3] === 0
        width: 640; height: 260
        x: middle ? (win.sw - width) / 2 : Math.max(20, Math.min(win.sw - width - 20, win.cur[1] - width / 2))
        y: middle ? (win.sh - height) / 2 : (win.cur[2] > win.sh / 2 ? win.cur[2] - win.cur[3] * 0.55 - height - 30 : win.cur[2] + win.cur[3] * 0.55 + 30)
        Behavior on x { NumberAnimation { duration: 400; easing.type: Easing.InOutQuad } }
        Behavior on y { NumberAnimation { duration: 400; easing.type: Easing.InOutQuad } }
        PetBody {
            id: pet
            x: 0; y: 70
            mood: "happy"; stage: 2
            SequentialAnimation on hopY { loops: Animation.Infinite
                NumberAnimation { to: 14; duration: 300; easing.type: Easing.OutQuad }
                NumberAnimation { to: 0; duration: 300; easing.type: Easing.InQuad }
                PauseAnimation { duration: 1400 } }
        }
        Rectangle {
            x: 150; y: 0
            width: 490; height: 250
            radius: 26
            color: "white"
            border.color: "#0b4f4a"; border.width: 3
            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 22
                spacing: 10
                Label {
                    text: win.cur[0]
                    color: "#0b4f4a"; font.pixelSize: 22; font.bold: true
                    wrapMode: Text.WordWrap
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                }
                RowLayout {
                    Label { text: (win.at + 1) + " / " + win.steps.length; color: "#64748b"; font.pixelSize: 15 }
                    Item { Layout.fillWidth: true }
                    Button { text: "Skip"; flat: true; visible: win.at < win.steps.length - 1; onClicked: win.go("skip") }
                    Button { text: "Back"; visible: win.at > 0; onClicked: win.go("back") }
                    Button { text: win.at === win.steps.length - 1 ? "Let's go!" : "Next"; highlighted: true; font.bold: true; onClicked: win.go("next") }
                }
            }
        }
    }

    Item {
        anchors.fill: parent
        focus: true
        Keys.onPressed: (k) => {
            if (k.key === Qt.Key_Right || k.key === Qt.Key_Return || k.key === Qt.Key_Enter || k.key === Qt.Key_Space) win.go("next");
            else if (k.key === Qt.Key_Left || k.key === Qt.Key_Backspace) win.go("back");
            else if (k.key === Qt.Key_Escape) win.go("skip");
        }
    }
    property int lastSeq: -1
    Timer {
        interval: 80; running: true; repeat: true
        onTriggered: {
            const x = new XMLHttpRequest();
            x.open("GET", "file://" + win.arg("events"), false);
            let e;
            try { x.send(); e = JSON.parse(x.responseText).events; } catch (z) { return; }
            if (win.lastSeq < 0) { win.lastSeq = e.length ? e[e.length - 1][0] : 0; return; }
            for (const [seq, w] of e) if (seq > win.lastSeq) { win.lastSeq = seq; win.go(w === "back" ? "back" : w); }
        }
    }
    Confetti { id: confetti }
}

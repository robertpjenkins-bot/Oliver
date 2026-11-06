// Ollie Quick Menu (ollie-quickmenu): a panel down the right of the screen,
// over games too (Plasma's layer shell, the overlay layer). Volume,
// brightness, an FPS limit (MangoHud), the power mode, Do Not Disturb, a
// screenshot, closing the game, and the Game Hub. Up and down pick, left and
// right change, A does it, B closes; the keyboard works too.
// The controller's buttons come from ollie-padd ($XDG_RUNTIME_DIR/ollie-pad-events).
// Says on the console: OLLIE-ACTION volume N | bright N | fps N | power MODE |
// dnd on|off | screenshot | close-game | run PROGRAM | quit.
// Arguments: game= (what is being played) today= (its minutes today)
// vol= bright= (0-100; -1 when the screen has no brightness control)
// fps= (0 for no limit) power= (power-saver|balanced|performance|none)
// dnd=on|off events=FILE.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.layershell 1.0 as LayerShell

Window {
    id: win
    visible: true
    color: "transparent"
    title: "Ollie Quick Menu"
    width: 440
    height: Screen.height
    LayerShell.Window.scope: "ollie-quickmenu"
    LayerShell.Window.layer: LayerShell.Window.LayerOverlay
    LayerShell.Window.anchors: LayerShell.Window.AnchorRight | LayerShell.Window.AnchorTop | LayerShell.Window.AnchorBottom
    LayerShell.Window.keyboardInteractivity: LayerShell.Window.KeyboardInteractivityOnDemand
    LayerShell.Window.exclusionZone: -1

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }
    function num(n, d) { const v = parseInt(arg(n)); return isNaN(v) ? d : v }

    readonly property string game: arg("game")
    property int vol: num("vol", 50)
    property int bright: num("bright", -1)
    property int fps: num("fps", 0)
    property string power: arg("power") || "none"
    property bool dnd: arg("dnd") === "on"
    property bool sure: false          // "Close the game?" asked once
    readonly property var fpsSteps: [0, 30, 45, 60, 90, 120]
    readonly property var powers: [["power-saver", "Battery saver"], ["balanced", "Balanced"], ["performance", "Turbo"]]

    // The rows: [id, shown]
    readonly property var items: [["vol"], ["bright"], ["fps"], ["power"], ["dnd"], ["shot"], ["close"], ["hub"], ["playtime"]]
        .filter(i => (i[0] !== "bright" || bright >= 0) && (i[0] !== "power" || power !== "none") && (i[0] !== "close" || game !== ""))
    property int at: 0
    readonly property string cur: items[Math.min(at, items.length - 1)][0]

    function change(d) {
        if (cur === "vol") { vol = Math.max(0, Math.min(100, vol + d * 10)); act("volume " + vol); }
        else if (cur === "bright") { bright = Math.max(5, Math.min(100, bright + d * 10)); act("bright " + bright); }
        else if (cur === "fps") { const i = Math.max(0, Math.min(fpsSteps.length - 1, fpsSteps.indexOf(fps) + d)); fps = fpsSteps[i]; act("fps " + fps); }
        else if (cur === "power") { const i = Math.max(0, Math.min(2, powers.findIndex(p => p[0] === power) + d)); power = powers[i][0]; act("power " + power); }
        else if (cur === "dnd") { dnd = !dnd; act("dnd " + (dnd ? "on" : "off")); }
    }
    function go(w) {
        if (w === "up") { at = Math.max(0, at - 1); sure = false; }
        else if (w === "down") { at = Math.min(items.length - 1, at + 1); sure = false; }
        else if (w === "left") change(-1);
        else if (w === "right") change(1);
        else if (w === "ok") {
            if (cur === "dnd") change(1);
            else if (cur === "shot") act("screenshot");
            else if (cur === "close") { if (sure) act("close-game"); else sure = true; }
            else if (cur === "hub") act("run ollie-gamehub");
            else if (cur === "playtime") act("run ollie-playtime");
            else change(1);
        }
        else if (w === "back" || w === "start" || w === "select") act("quit");
    }
    property int lastSeq: -1
    Timer {
        interval: 60; running: true; repeat: true
        onTriggered: {
            const x = new XMLHttpRequest();
            x.open("GET", "file://" + win.arg("events"), false);
            let e;
            try { x.send(); e = JSON.parse(x.responseText).events; } catch (z) { return; }
            if (win.lastSeq < 0) { win.lastSeq = e.length ? e[e.length - 1][0] : 0; return; }
            for (const [seq, w] of e) if (seq > win.lastSeq) { win.lastSeq = seq; win.go(w); }
        }
    }

    Rectangle {
        id: panel
        anchors.fill: parent
        anchors.margins: 14
        radius: 8
        color: "#F217181C"
        border.color: "#2F3138"; border.width: 1
        focus: true
        Keys.onPressed: (k) => {
            const m = { [Qt.Key_Left]: "left", [Qt.Key_Right]: "right", [Qt.Key_Up]: "up", [Qt.Key_Down]: "down",
                        [Qt.Key_Return]: "ok", [Qt.Key_Enter]: "ok", [Qt.Key_Space]: "ok", [Qt.Key_Escape]: "back" };
            if (m[k.key]) { win.go(m[k.key]); k.accepted = true; }
        }
        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 22
            spacing: 12
            RowLayout {
                Label { text: "Quick Menu"; color: "white"; font.pixelSize: 28; font.bold: true }
                Item { Layout.fillWidth: true }
                Label { id: clock; color: "#C9CAD0"; font.pixelSize: 20
                    Timer { interval: 1000; running: true; repeat: true; triggeredOnStart: true; onTriggered: clock.text = Qt.formatTime(new Date(), "h:mm AP") } }
            }
            Rectangle {
                visible: win.game !== ""
                Layout.fillWidth: true
                Layout.preferredHeight: 70
                radius: 6
                color: "#25272D"
                ColumnLayout {
                    anchors.fill: parent; anchors.margins: 12; spacing: 0
                    Label { text: "" + win.game; color: "white"; font.pixelSize: 18; font.bold: true; elide: Text.ElideRight; Layout.fillWidth: true }
                    Label { text: "Played today: " + win.arg("today") + " min"; color: "#9A9CA3"; font.pixelSize: 14 }
                }
            }
            Repeater {
                model: win.items
                delegate: Rectangle {
                    id: row
                    required property var modelData
                    required property int index
                    readonly property bool on: index === win.at
                    readonly property string what: modelData[0]
                    Layout.fillWidth: true
                    Layout.preferredHeight: 62
                    radius: 6
                    color: on ? "#33251B" : "#25272D"
                    border.color: on ? "#E07A2F" : "transparent"; border.width: 2
                    MouseArea { anchors.fill: parent; onClicked: { win.at = row.index; win.go("ok"); } }
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 16; anchors.rightMargin: 16; spacing: 12
                        Label {
                            color: "white"; font.pixelSize: 18
                            text: ({ vol: "Volume", bright: "Brightness", fps: "FPS limit", power: "Power", dnd: "Do Not Disturb",
                                     shot: "Take a screenshot", close: win.sure && row.on ? "Press again to close it" : "Close the game",
                                     hub: "Game Hub", playtime: "Play Time" })[row.what]
                        }
                        Item { Layout.fillWidth: true }
                        Label {
                            color: "#F0A066"; font.pixelSize: 17; font.bold: true
                            text: row.what === "vol" ? "" + win.vol + "%" : row.what === "bright" ? "" + win.bright + "%"
                                : row.what === "fps" ? "◀ " + (win.fps ? win.fps : "No limit") + " ▶"
                                : row.what === "power" ? "◀ " + (win.powers.find(p => p[0] === win.power) || ["", "?"])[1] + " ▶"
                                : row.what === "dnd" ? (win.dnd ? "On" : "Off") : ""
                        }
                    }
                }
            }
            Item { Layout.fillHeight: true }
            Label { text: "A: choose   ←→: change   B: close"; color: "#9A9CA3"; font.pixelSize: 16; Layout.alignment: Qt.AlignHCenter }
        }
    }
}

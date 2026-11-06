// Photo Booth (ollie-photobooth): the webcam, with funny hats, glasses and
// stickers to drag onto the picture (bigger and smaller with the mouse
// wheel), colour effects, a mirror, and a 3-2-1 countdown. Photos go to
// Pictures/Photo Booth.
// Says on the console: OLLIE-ACTION saved FILE, trophy ...
// Argument: dir=DIR (where photos go).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Effects
import QtMultimedia

ApplicationWindow {
    id: win
    width: 1100
    height: 760
    visible: true
    title: "Photo Booth"
    color: "#18181b"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }

    readonly property var props: ["🎩", "👑", "🤠", "🎓", "🧢", "🪖", "🎀", "🕶️", "👓", "🥸", "👃", "👄", "🐶", "🐱", "🦄", "🐸",
                                  "⭐", "💖", "🌈", "🔥", "💥", "✨", "🎉", "🍕", "👻", "👽", "🤖", "💬"]
    readonly property var effects: [
        ["Normal", 0, 0, 0, "#000000", 0], ["Black and white", -1, 0, 0, "#000000", 0], ["Super colour", 0.9, 0.15, 0, "#000000", 0],
        ["Comic", 0.4, 0, 0.6, "#000000", 0], ["Dreamy", 0, 0.15, -0.2, "#000000", 0.5], ["Alien", 0, 0, 0.1, "#22c55e", 0],
        ["Ice", -0.3, 0.1, 0.1, "#38bdf8", 0], ["Spooky", -0.6, -0.25, 0.4, "#7c3aed", 0]
    ]
    property int effect: 0
    property bool mirror: true
    property var stuck: []        // [{e, x, y, s}]
    property int count: 0
    property var shots: []        // saved files, newest first

    MediaDevices { id: devices }
    readonly property bool hasCamera: devices.videoInputs.length > 0
    CaptureSession {
        camera: Camera { id: camera; active: win.hasCamera }
        videoOutput: video
    }

    function snap() {
        if (count > 0) return;
        count = 3; countdown.start();
    }
    Timer {
        id: countdown
        interval: 800; repeat: true
        onTriggered: {
            win.count--;
            if (win.count > 0) return;
            stop();
            flash.opacity = 1; fade.start();
            const f = win.arg("dir") + "/Photo " + Qt.formatDateTime(new Date(), "yyyy-MM-dd hh-mm-ss") + ".png";
            photo.grabToImage(r => {
                if (r.saveToFile(f)) { win.shots = [f].concat(win.shots).slice(0, 8); win.act("saved " + f); win.act("trophy count photo"); }
            });
        }
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 16

        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 12
            Label { text: "Photo Booth"; color: "white"; font.pixelSize: 30; font.bold: true }

            // The picture: the camera, the effect, and the stickers on top.
            Item {
                id: photo
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true
                Rectangle { anchors.fill: parent; color: "#27272a"; radius: 12 }
                VideoOutput {
                    id: video
                    anchors.fill: parent
                    fillMode: VideoOutput.PreserveAspectCrop
                    visible: false
                }
                MultiEffect {
                    source: video
                    anchors.fill: video
                    visible: win.hasCamera
                    transform: Scale { origin.x: video.width / 2; xScale: win.mirror ? -1 : 1 }
                    saturation: win.effects[win.effect][1]
                    brightness: win.effects[win.effect][2]
                    contrast: win.effects[win.effect][3]
                    colorization: win.effects[win.effect][4] === "#000000" ? 0 : 0.45
                    colorizationColor: win.effects[win.effect][4]
                    blurEnabled: win.effects[win.effect][5] > 0
                    blur: win.effects[win.effect][5]
                }
                ColumnLayout {
                    visible: !win.hasCamera
                    anchors.centerIn: parent
                    Image { source: "file:///usr/share/icons/hicolor/128x128/apps/ollie-photobooth.png"; sourceSize: Qt.size(96, 96); Layout.alignment: Qt.AlignHCenter }
                    Label { text: "No camera found.\nPlug in a webcam, then open Photo Booth again."; color: "#d4d4d8"; font.pixelSize: 20; horizontalAlignment: Text.AlignHCenter; Layout.alignment: Qt.AlignHCenter }
                }
                Repeater {
                    model: win.stuck
                    delegate: Text {
                        id: prop
                        required property var modelData
                        required property int index
                        x: modelData.x; y: modelData.y
                        text: modelData.e
                        font.pixelSize: 90 * modelData.s
                        MouseArea {
                            anchors.fill: parent
                            drag.target: prop
                            cursorShape: Qt.SizeAllCursor
                            onReleased: { const s = win.stuck.slice(); s[prop.index] = Object.assign({}, s[prop.index], { x: prop.x, y: prop.y }); win.stuck = s; }
                            onWheel: (w) => { const s = win.stuck.slice(); s[prop.index] = Object.assign({}, s[prop.index], { s: Math.max(0.3, Math.min(4, s[prop.index].s * (w.angleDelta.y > 0 ? 1.15 : 0.87))) }); win.stuck = s; }
                            onDoubleClicked: { const s = win.stuck.slice(); s.splice(prop.index, 1); win.stuck = s; }
                        }
                    }
                }
                Label {
                    visible: win.count > 0
                    anchors.centerIn: parent
                    text: win.count
                    color: "white"; font.pixelSize: 200; font.bold: true
                    style: Text.Outline; styleColor: "#000000"
                }
                Rectangle { id: flash; anchors.fill: parent; color: "white"; opacity: 0
                    NumberAnimation on opacity { id: fade; to: 0; duration: 500; running: false } }
            }
            RowLayout {
                spacing: 10
                Button { text: "Take a photo"; font.pixelSize: 20; font.bold: true; highlighted: true; enabled: win.hasCamera && win.count === 0; onClicked: win.snap() }
                Button { text: win.mirror ? "Mirror: on" : "Mirror: off"; onClicked: win.mirror = !win.mirror }
                Button { text: "Take the stickers off"; onClicked: win.stuck = [] }
                Label { text: "Drag stickers to move them · mouse wheel: bigger or smaller · double-click: take it off"; color: "#a1a1aa"; font.pixelSize: 13; wrapMode: Text.WordWrap; Layout.fillWidth: true }
            }
            // The last photos.
            Row {
                visible: win.shots.length > 0
                spacing: 8
                Repeater {
                    model: win.shots
                    delegate: Image { required property string modelData; source: "file://" + modelData; height: 70; width: 105; fillMode: Image.PreserveAspectCrop }
                }
            }
        }

        // Stickers and effects.
        ColumnLayout {
            Layout.preferredWidth: 290
            Layout.minimumWidth: 290
            Layout.maximumWidth: 290
            Layout.fillHeight: true
            spacing: 10
            Label { text: "Add a sticker"; color: "white"; font.pixelSize: 17; font.bold: true }
            GridLayout {
                columns: 5
                columnSpacing: 4; rowSpacing: 4
                Repeater {
                    model: win.props
                    delegate: Button {
                        required property string modelData
                        text: modelData; font.pixelSize: 26; flat: true
                        Layout.preferredWidth: 54; Layout.preferredHeight: 50
                        onClicked: win.stuck = win.stuck.concat([{ e: modelData, x: photo.width / 2 - 45 + (Math.random() - 0.5) * 120, y: photo.height / 3 + (Math.random() - 0.5) * 80, s: 1 }])
                    }
                }
            }
            Label { text: "Effects"; color: "white"; font.pixelSize: 17; font.bold: true; Layout.topMargin: 8 }
            GridLayout {
                columns: 2
                columnSpacing: 6; rowSpacing: 6
                Repeater {
                    model: win.effects
                    delegate: Button {
                        required property var modelData
                        required property int index
                        text: modelData[0]
                        highlighted: win.effect === index
                        Layout.fillWidth: true
                        onClicked: win.effect = index
                    }
                }
            }
            Item { Layout.fillHeight: true }
        }
    }
}

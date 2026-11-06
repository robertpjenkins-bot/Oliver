// Sound Board (ollie-soundboard): 12 big buttons that play sounds. Six come
// with fun sounds; the others (or any, after Change) play what you record
// with the microphone (up to 10 seconds). Each gets a name and an emoji.
// Says on the console: OLLIE-ACTION play FILE | record FILE | stop.
// Arguments: config=FILE, sounds=DIR (the made sounds), mine=DIR (recordings).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 980
    height: 720
    visible: true
    title: "Sound Board"
    color: "#1e1b4b"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }

    readonly property var builtIn: [
        { emoji: "", name: "Laser", file: "laser.ogg" }, { emoji: "", name: "Boing", file: "boing.ogg" },
        { emoji: "", name: "Drum", file: "drum.ogg" }, { emoji: "", name: "Coin", file: "coin.ogg" },
        { emoji: "", name: "Siren", file: "siren.ogg" }, { emoji: "", name: "Silly", file: "silly.ogg" }
    ]
    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property string pads: ""
    }
    property var pads: []
    Component.onCompleted: {
        try { pads = JSON.parse(saved.pads); } catch (e) { pads = []; }
        if (pads.length !== 12) {
            const p = builtIn.map(b => ({ emoji: b.emoji, name: b.name, file: arg("sounds") + "/" + b.file }));
            const colours = ["🎤", "🎵", "😂", "🐶", "🐱", "👻"];
            for (let i = 0; i < 6; i++) p.push({ emoji: colours[i], name: "Record me!", file: "" });
            pads = p;
        }
    }
    function savePads(p) { pads = p; saved.pads = JSON.stringify(p); }

    property int recording: -1
    property int recordTime: 0
    property int editing: -1
    property int playingPad: -1
    function startRecord(i) {
        const f = arg("mine") + "/sound-" + Date.now() + ".wav";
        recording = i; recordTime = 0;
        const p = pads.slice(); p[i] = Object.assign({}, p[i], { file: f, name: p[i].name === "Record me!" ? "My sound" : p[i].name }); savePads(p);
        act("record " + f); act("trophy count sound-record");
    }
    function stopRecord() { if (recording < 0) return; act("stop"); recording = -1; }
    Timer { interval: 1000; running: win.recording >= 0; repeat: true
        onTriggered: { win.recordTime++; if (win.recordTime >= 10) win.stopRecord(); } }
    Timer { id: glow; interval: 450; onTriggered: win.playingPad = -1 }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 14
        RowLayout {
            Label { text: "Sound Board"; color: "white"; font.pixelSize: 32; font.bold: true }
            Item { Layout.fillWidth: true }
            Label {
                text: win.recording >= 0 ? "Recording... " + (10 - win.recordTime) + " s left" : "Click a button to play it. changes a button."
                color: win.recording >= 0 ? "#fca5a5" : "#c7d2fe"; font.pixelSize: 16
            }
            Button { visible: win.recording >= 0; text: "Stop"; onClicked: win.stopRecord() }
        }
        GridLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            columns: 4
            rowSpacing: 14
            columnSpacing: 14
            Repeater {
                model: win.pads
                delegate: Rectangle {
                    required property var modelData
                    required property int index
                    readonly property bool empty: modelData.file === ""
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: 22
                    color: win.recording === index ? "#dc2626" : win.playingPad === index ? "#facc15"
                        : empty ? "#312e81" : ["#7c3aed", "#db2777", "#2563eb", "#059669", "#ea580c", "#0891b2"][index % 6]
                    scale: pad.pressed ? 0.94 : win.playingPad === index ? 1.05 : 1
                    Behavior on scale { NumberAnimation { duration: 90 } }
                    border.color: "white"; border.width: empty ? 2 : 0
                    ColumnLayout {
                        anchors.centerIn: parent
                        spacing: 4
                        Label { text: win.recording === index ? "" : modelData.emoji; font.pixelSize: 52; Layout.alignment: Qt.AlignHCenter }
                        Label { text: modelData.name; color: "white"; font.pixelSize: 17; font.bold: true; Layout.alignment: Qt.AlignHCenter }
                    }
                    MouseArea {
                        id: pad
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        onClicked: {
                            if (win.recording === index) { win.stopRecord(); return; }
                            if (parent.empty) { win.startRecord(index); return; }
                            win.playingPad = index; glow.restart();
                            win.act("play " + modelData.file);
                        }
                    }
                    Button {
                        anchors.right: parent.right; anchors.top: parent.top; anchors.margins: 6
                        text: ""; flat: true
                        onClicked: { win.editing = index; nameField.text = modelData.name; editBox.open(); }
                    }
                }
            }
        }
    }

    Dialog {
        id: editBox
        anchors.centerIn: parent
        title: "Change this button"
        modal: true
        standardButtons: Dialog.Close
        ColumnLayout {
            spacing: 10
            TextField { id: nameField; placeholderText: "Its name"; maximumLength: 14; Layout.preferredWidth: 300
                onTextEdited: if (win.editing >= 0) { const p = win.pads.slice(); p[win.editing] = Object.assign({}, p[win.editing], { name: text }); win.savePads(p); } }
            Flow {
                Layout.preferredWidth: 340
                spacing: 2
                Repeater {
                    model: ["🎤", "🎵", "😂", "🐶", "🐱", "👻", "🦖", "🐸", "🐔", "🦁", "💨", "🎺", "🥁", "🔔", "💥", "👏", "🤖", "👽"]
                    delegate: Button { required property string modelData; text: modelData; font.pixelSize: 20; flat: true
                        onClicked: if (win.editing >= 0) { const p = win.pads.slice(); p[win.editing] = Object.assign({}, p[win.editing], { emoji: modelData }); win.savePads(p); } }
                }
            }
            RowLayout {
                Button { text: "Record a new sound"; onClicked: { editBox.close(); win.startRecord(win.editing); } }
                Button {
                    text: "Its first sound"
                    visible: win.editing >= 0 && win.editing < 6
                    onClicked: { const b = win.builtIn[win.editing]; const p = win.pads.slice();
                        p[win.editing] = { emoji: b.emoji, name: b.name, file: win.arg("sounds") + "/" + b.file }; win.savePads(p); editBox.close(); }
                }
            }
        }
    }
}

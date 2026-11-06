// Music Maker (ollie-music): a piano of two octaves (click the keys, or play
// them on the keyboard: A S D F G H J K L ; ' and W E T Y U O P), three
// instruments, and a beat maker (4 drums x 16 steps, with a speed). Record
// what you play, play it back, keep it as a song, and save it as a music
// file to play anywhere.
// Says on the console: OLLIE-ACTION note INSTRUMENT N (N: 0 = C4 ... 24 = C6),
// drums NAME..., save NAME (the song: then its JSON on the next line as
// OLLIE-SONG JSON), export NAME, open NAME, trophy ....
// Arguments: songs=FILE (the saved songs' names, one a line), song=FILE (one
// to open, written by ollie-music when asked).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1080
    height: 720
    visible: true
    title: "Music Maker"
    color: "#1e1b4b"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }
    function read(f) {
        const x = new XMLHttpRequest();
        x.open("GET", "file://" + f, false);
        try { x.send(); return x.responseText; } catch (e) { return ""; }
    }

    readonly property var instruments: [["piano", "🎹 Piano"], ["xylo", "🔔 Xylophone"], ["synth", "👾 8-bit"]]
    property string instrument: "piano"
    readonly property var drums: [["kick", "🥁 Boom"], ["snare", "🪘 Tsss"], ["hihat", "✨ Tick"], ["clap", "👏 Clap"]]
    property var beat: [[], [], [], []]       // drum -> steps on
    property int tempo: 110
    property int step: -1
    property bool beatOn: false
    // Recording: [ms from the start, note, instrument]
    property bool recording: false
    property bool playing: false
    property var notes: []
    property real startedAt: 0
    property var pressed: ({})
    property string songName: ""
    property string message: ""

    // A note: N from 0 (C4) to 24 (C6).
    readonly property var names: ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"]
    function isBlack(n) { return [1, 3, 6, 8, 10].includes(n % 12) }
    // The computer keyboard's keys for the notes (C4 to F5).
    readonly property var keyMap: ({ "a": 0, "w": 1, "s": 2, "e": 3, "d": 4, "f": 5, "t": 6, "g": 7, "y": 8, "h": 9, "u": 10, "j": 11,
                                     "k": 12, "o": 13, "l": 14, "p": 15, ";": 16, "'": 17 })
    function keyOf(n) { for (const k in keyMap) if (keyMap[k] === n) return k.toUpperCase(); return ""; }

    function play(n) {
        act("note " + instrument + " " + n);
        const p = Object.assign({}, pressed); p[n] = true; pressed = p;
        unpress.restart();
        if (recording) notes = notes.concat([[Date.now() - startedAt, n, instrument]]);
    }
    Timer { id: unpress; interval: 160; onTriggered: win.pressed = ({}) }

    // The beat maker.
    Timer {
        interval: 60000 / win.tempo / 4; running: win.beatOn; repeat: true
        onTriggered: {
            win.step = (win.step + 1) % 16;
            const hit = win.drums.filter((d, i) => win.beat[i].includes(win.step)).map(d => d[0]);
            if (hit.length) win.act("drums " + hit.join(" "));
        }
        onRunningChanged: if (!running) win.step = -1
    }
    function toggleStep(d, s) {
        const b = beat.map(r => r.slice());
        const i = b[d].indexOf(s);
        if (i >= 0) b[d].splice(i, 1); else b[d].push(s);
        beat = b;
        if (!beatOn && i < 0) act("drums " + drums[d][0]);
    }

    // Recording and playing back.
    function record() {
        notes = []; recording = true; playing = false; startedAt = Date.now(); message = "⏺ Recording... play something!";
        if (!beatOn && beat.some(r => r.length)) beatOn = true;
    }
    function stop() {
        if (recording) message = notes.length ? "Got it! " + notes.length + " notes. ▶ to hear it, 💾 to keep it." : "";
        recording = false; playing = false; playback.stop();
    }
    property int playAt: 0
    function playBack() {
        if (!notes.length) { message = "Record something first! ⏺"; return; }
        recording = false; playing = true; playAt = 0; startedAt = Date.now(); playback.start();
        if (beat.some(r => r.length)) beatOn = true;
    }
    Timer {
        id: playback
        interval: 10; repeat: true
        onTriggered: {
            const t = Date.now() - win.startedAt;
            while (win.playAt < win.notes.length && win.notes[win.playAt][0] <= t) {
                const n = win.notes[win.playAt];
                win.act("note " + n[2] + " " + n[1]);
                const p = Object.assign({}, win.pressed); p[n[1]] = true; win.pressed = p; unpress.restart();
                win.playAt++;
            }
            if (win.playAt >= win.notes.length) { stop(); win.playing = false; win.beatOn = false; }
        }
    }
    function song() { return JSON.stringify({ tempo: tempo, beat: beat, notes: notes, instrument: instrument }) }
    function save(alsoFile) {
        const n = nameField.text.trim().replace(/[\/\\:*?"<>|]/g, "").slice(0, 40);
        if (!n) { message = "Give your song a name first ✏️"; return; }
        if (!notes.length && !beat.some(r => r.length)) { message = "Play or record something first! 🎹"; return; }
        songName = n;
        console.log("OLLIE-SONG " + n + "\t" + song());
        act("save " + n);
        if (alsoFile) act("export " + n);
        act("trophy count song");
        message = alsoFile ? "🎵 Saving \"" + n + "\" as a music file, in Music, Ollie Songs..." : "💾 Saved \"" + n + "\"!";
        songs = songs.includes(n) ? songs : songs.concat([n]);
    }
    property var songs: []
    Component.onCompleted: songs = read(arg("songs")).split("\n").filter(l => l !== "")
    function open(n) {
        act("open " + n);
        loadTimer.name = n; loadTimer.start();
    }
    Timer {
        id: loadTimer
        property string name
        interval: 400; repeat: true
        property int tries: 0
        onTriggered: {
            let s;
            try { s = JSON.parse(win.read(win.arg("song"))); } catch (e) { if (++tries > 10) { stop(); tries = 0; } return; }
            if (s.name !== name) { if (++tries > 10) { stop(); tries = 0; } return; }
            stop(); tries = 0;
            win.tempo = s.tempo || 110; win.beat = s.beat || [[], [], [], []]; win.notes = s.notes || []; win.instrument = s.instrument || "piano";
            win.songName = name; nameField.text = name; win.message = "📂 \"" + name + "\" is ready: ▶ to hear it.";
        }
    }

    Item {
        anchors.fill: parent
        focus: true
        Keys.onPressed: (k) => {
            if (nameField.activeFocus) return;
            const n = win.keyMap[k.text.toLowerCase()];
            if (n !== undefined && !k.isAutoRepeat) { win.play(n); k.accepted = true; }
        }
        MouseArea { anchors.fill: parent; onPressed: (m) => { parent.forceActiveFocus(); m.accepted = false; } }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 12

        RowLayout {
            spacing: 10
            Label { text: "🎵 Music Maker"; color: "white"; font.pixelSize: 30; font.bold: true }
            Item { width: 12 }
            Repeater {
                model: win.instruments
                delegate: Button {
                    required property var modelData
                    text: modelData[1]; font.pixelSize: 15
                    highlighted: win.instrument === modelData[0]
                    onClicked: { win.instrument = modelData[0]; win.play(12); }
                }
            }
            Item { Layout.fillWidth: true }
            Label { text: win.message; color: "#fde68a"; font.pixelSize: 15 }
        }

        // The beat maker.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 250
            radius: 18
            color: "#2e2a6b"
            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 14
                spacing: 8
                RowLayout {
                    Label { text: "🥁 Beat maker"; color: "white"; font.pixelSize: 18; font.bold: true }
                    Button { text: win.beatOn ? "⏸ Stop the beat" : "▶ Play the beat"; onClicked: win.beatOn = !win.beatOn }
                    Button { text: "🧹 Clear"; onClicked: win.beat = [[], [], [], []] }
                    Button { text: "🎲 Surprise me"; onClicked: win.beat = [[0, 8], [4, 12], [0, 2, 4, 6, 8, 10, 12, 14], Math.random() < 0.5 ? [12] : [6, 14]] }
                    Item { Layout.fillWidth: true }
                    Label { text: "🐢"; font.pixelSize: 18 }
                    Slider { from: 60; to: 180; stepSize: 5; value: win.tempo; onMoved: win.tempo = value; Layout.preferredWidth: 180 }
                    Label { text: "🐇 " + win.tempo; color: "#c7d2fe"; font.pixelSize: 15 }
                }
                Repeater {
                    model: win.drums
                    delegate: RowLayout {
                        id: drumRow
                        required property var modelData
                        required property int index
                        spacing: 5
                        Label { text: drumRow.modelData[1]; color: "white"; font.pixelSize: 15; Layout.preferredWidth: 90 }
                        Repeater {
                            model: 16
                            delegate: Rectangle {
                                required property int index
                                readonly property bool on: win.beat[drumRow.index].includes(index)
                                Layout.fillWidth: true
                                Layout.preferredHeight: 34
                                radius: 8
                                color: on ? ["#f43f5e", "#f59e0b", "#22d3ee", "#a3e635"][drumRow.index] : index % 4 === 0 ? "#4338ca" : "#3730a3"
                                border.color: win.step === index ? "white" : "transparent"; border.width: 3
                                MouseArea { anchors.fill: parent; onClicked: win.toggleStep(drumRow.index, index) }
                            }
                        }
                    }
                }
            }
        }

        // Recording and songs.
        RowLayout {
            spacing: 8
            Button { text: win.recording ? "⏺ Recording..." : "⏺ Record"; highlighted: win.recording; onClicked: win.recording ? win.stop() : win.record() }
            Button { text: "⏹ Stop"; onClicked: { win.stop(); win.beatOn = false; } }
            Button { text: win.playing ? "🔊 Playing..." : "▶ Play my song"; onClicked: win.playBack() }
            Item { width: 10 }
            TextField { id: nameField; placeholderText: "My song's name"; maximumLength: 40; Layout.preferredWidth: 190 }
            Button { text: "💾 Keep it"; onClicked: win.save(false) }
            Button { text: "🎵 Save as music file"; onClicked: win.save(true) }
            Item { Layout.fillWidth: true }
            ComboBox {
                visible: win.songs.length > 0
                model: ["📂 My songs"].concat(win.songs)
                Layout.preferredWidth: 190
                onActivated: (i) => { if (i > 0) win.open(win.songs[i - 1]); currentIndex = 0; }
            }
        }

        // The piano.
        Item {
            id: piano
            Layout.fillWidth: true
            Layout.fillHeight: true
            readonly property int whites: 15
            readonly property real w: width / whites
            Repeater { // white keys
                model: 25
                delegate: Rectangle {
                    required property int index
                    readonly property int white: [0, 2, 4, 5, 7, 9, 11].indexOf(index % 12)
                    visible: !win.isBlack(index)
                    x: (Math.floor(index / 12) * 7 + white) * piano.w
                    width: piano.w - 3; height: piano.height
                    radius: 8
                    color: win.pressed[index] ? "#fde047" : "white"
                    border.color: "#1e1b4b"
                    Column {
                        anchors.bottom: parent.bottom; anchors.bottomMargin: 10; anchors.horizontalCenter: parent.horizontalCenter
                        Label { text: win.names[index % 12]; color: "#6d28d9"; font.pixelSize: 16; font.bold: true; anchors.horizontalCenter: parent.horizontalCenter }
                        Label { text: win.keyOf(index); color: "#94a3b8"; font.pixelSize: 13; anchors.horizontalCenter: parent.horizontalCenter }
                    }
                    MouseArea { anchors.fill: parent; onPressed: win.play(index) }
                }
            }
            Repeater { // black keys
                model: 25
                delegate: Rectangle {
                    required property int index
                    readonly property int whiteBefore: [0, 2, 4, 5, 7, 9, 11].indexOf((index - 1) % 12)
                    visible: win.isBlack(index)
                    x: (Math.floor(index / 12) * 7 + whiteBefore + 1) * piano.w - piano.w * 0.32
                    width: piano.w * 0.62; height: piano.height * 0.6
                    radius: 6
                    color: win.pressed[index] ? "#f59e0b" : "#111827"
                    Label { anchors.bottom: parent.bottom; anchors.bottomMargin: 8; anchors.horizontalCenter: parent.horizontalCenter
                        text: win.keyOf(index); color: "#9ca3af"; font.pixelSize: 12 }
                    MouseArea { anchors.fill: parent; onPressed: win.play(index) }
                }
            }
        }
    }
}

// Ollie Animator (ollie-animator): draw a cartoon frame by frame, like a
// flip book. The frame before shows faintly while you draw the next
// ("onion skin"); Play shows it moving; Save makes a GIF in
// Pictures/Animations. Says on the console: OLLIE-ACTION gif DIR FPS (the
// frames are DIR/frame-000.png ...), say TEXT.
// Argument: frames=DIR (an empty folder for the frames while saving).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1100
    height: 760
    visible: true
    title: "Ollie Animator"
    color: "#ecfeff"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }

    property var frames: [[]]          // each: [{color, size, pts: [{x, y}]}]
    property int cur: 0
    property string ink: "#111827"
    property int brush: 8
    property bool onion: true
    property bool playing: false
    property bool exporting: false
    property int fps: 6

    function setFrames(f) { frames = f; canvas.requestPaint(); }
    function newFrame(copy) {
        const f = frames.slice();
        f.splice(cur + 1, 0, copy ? JSON.parse(JSON.stringify(frames[cur])) : []);
        frames = f; cur++; canvas.requestPaint();
    }
    function deleteFrame() {
        if (frames.length === 1) { setFrames([[]]); return; }
        const f = frames.slice(); f.splice(cur, 1);
        frames = f; cur = Math.min(cur, frames.length - 1); canvas.requestPaint();
    }
    function drawStrokes(c, strokes, alpha) {
        c.globalAlpha = alpha;
        c.lineCap = "round"; c.lineJoin = "round";
        for (const s of strokes) {
            c.strokeStyle = s.color; c.lineWidth = s.size;
            c.beginPath();
            for (let i = 0; i < s.pts.length; i++) i ? c.lineTo(s.pts[i].x, s.pts[i].y) : c.moveTo(s.pts[i].x, s.pts[i].y);
            if (s.pts.length === 1) c.lineTo(s.pts[0].x + 0.1, s.pts[0].y);
            c.stroke();
        }
        c.globalAlpha = 1;
    }

    // Saving: each frame, drawn and grabbed in turn, then the GIF.
    property int exportAt: 0
    function startExport() {
        playing = false; exporting = true; exportAt = 0; cur = 0; canvas.requestPaint();
    }
    function exported() {
        canvas.grabToImage(r => {
            r.saveToFile(arg("frames") + "/frame-" + ("00" + exportAt).slice(-3) + ".png");
            exportAt++;
            if (exportAt < frames.length) { cur = exportAt; canvas.requestPaint(); }
            else { exporting = false; cur = 0; canvas.requestPaint(); act("gif " + arg("frames") + " " + fps); note.text = "Saved in Pictures, Animations!"; }
        });
    }

    Timer {
        interval: 1000 / win.fps
        running: win.playing
        repeat: true
        onTriggered: { win.cur = (win.cur + 1) % win.frames.length; canvas.requestPaint(); }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 14
        spacing: 10

        RowLayout {
            Label { text: "🎬 Ollie Animator"; font.pixelSize: 28; font.bold: true; color: "#0e7490" }
            Item { Layout.fillWidth: true }
            Label { text: "Frame " + (win.cur + 1) + " of " + win.frames.length; font.pixelSize: 18; color: "#155e75" }
        }

        RowLayout {
            spacing: 14
            Layout.fillHeight: true
            // The page.
            Rectangle {
                Layout.preferredWidth: 740
                Layout.preferredHeight: 500
                color: "white"
                border.color: "#67e8f9"
                border.width: 3
                Canvas {
                    id: canvas
                    anchors.fill: parent
                    anchors.margins: 3
                    onPaint: {
                        const c = getContext("2d");
                        c.reset();
                        c.fillStyle = "white"; c.fillRect(0, 0, width, height);
                        if (win.onion && !win.playing && !win.exporting && win.cur > 0) win.drawStrokes(c, win.frames[win.cur - 1], 0.22);
                        win.drawStrokes(c, win.frames[win.cur], 1);
                    }
                    onPainted: if (win.exporting) Qt.callLater(win.exported)
                    MouseArea {
                        anchors.fill: parent
                        enabled: !win.playing && !win.exporting
                        onPressed: m => {
                            const f = win.frames.slice();
                            f[win.cur] = f[win.cur].concat([{ color: win.ink, size: win.brush, pts: [{ x: m.x, y: m.y }] }]);
                            win.frames = f; canvas.requestPaint();
                        }
                        onPositionChanged: m => {
                            const s = win.frames[win.cur];
                            s[s.length - 1].pts.push({ x: m.x, y: m.y });
                            canvas.requestPaint();
                        }
                    }
                }
            }
            // The tools.
            ColumnLayout {
                Layout.fillWidth: true
                spacing: 8
                Label { text: "Colour"; font.bold: true }
                GridLayout {
                    columns: 5
                    Repeater {
                        model: ["#111827", "#ef4444", "#f97316", "#facc15", "#22c55e", "#3b82f6", "#a855f7", "#ec4899", "#8b5a2b", "#ffffff"]
                        delegate: Rectangle { required property string modelData; width: 36; height: 36; radius: 18; color: modelData
                            border.width: win.ink === modelData ? 4 : 1; border.color: "#0e7490"
                            Text { anchors.centerIn: parent; text: parent.modelData === "#ffffff" ? "🧽" : ""; font.pixelSize: 16 }
                            MouseArea { anchors.fill: parent; onClicked: win.ink = parent.modelData } }
                    }
                }
                Label { text: "Brush"; font.bold: true }
                GridLayout {
                    columns: 2
                    Repeater {
                        model: [4, 8, 16, 30]
                        delegate: Button { required property int modelData; text: ["Thin", "Medium", "Thick", "Huge"][[4, 8, 16, 30].indexOf(modelData)]
                            highlighted: win.brush === modelData; onClicked: win.brush = modelData }
                    }
                }
                RowLayout {
                    Button { text: "↩️ Undo"; onClicked: { const f = win.frames.slice(); f[win.cur] = f[win.cur].slice(0, -1); win.setFrames(f); } }
                    Button { text: "🧽 Clear frame"; onClicked: { const f = win.frames.slice(); f[win.cur] = []; win.setFrames(f); } }
                }
                CheckBox { text: "See the frame before (faintly)"; checked: win.onion; onToggled: { win.onion = checked; canvas.requestPaint(); } }
                Label { text: "Speed: " + win.fps + " frames a second"; font.bold: true }
                Slider { from: 2; to: 12; stepSize: 1; value: win.fps; onMoved: win.fps = value; Layout.fillWidth: true }
                Item { Layout.fillHeight: true }
                Button {
                    text: win.playing ? "⏹ Stop" : "▶ Play"
                    font.pixelSize: 20
                    Layout.fillWidth: true
                    enabled: win.frames.length > 1
                    onClicked: { win.playing = !win.playing; if (!win.playing) { win.cur = 0; canvas.requestPaint(); } }
                }
                Button {
                    text: "💾 Save as a GIF"
                    highlighted: true
                    font.pixelSize: 18
                    Layout.fillWidth: true
                    enabled: !win.exporting
                    onClicked: win.startExport()
                }
                Label { id: note; color: "#15803d" }
            }
        }

        // The frames.
        RowLayout {
            Layout.fillWidth: true
            spacing: 8
            Button { text: "➕ New frame"; onClicked: win.newFrame(false) }
            Button { text: "📄 Copy this frame"; onClicked: win.newFrame(true) }
            Button { text: "🗑️ Delete frame"; onClicked: win.deleteFrame() }
            ScrollView {
                Layout.fillWidth: true
                Layout.preferredHeight: 92
                clip: true
                Row {
                    spacing: 6
                    Repeater {
                        model: win.frames.length
                        delegate: Rectangle {
                            required property int index
                            width: 100; height: 72
                            color: "white"
                            border.color: win.cur === index ? "#0891b2" : "#a5f3fc"
                            border.width: win.cur === index ? 4 : 2
                            Canvas {
                                id: thumb
                                anchors.fill: parent; anchors.margins: 3
                                property var strokes: win.frames[index]
                                onStrokesChanged: requestPaint()
                                Connections { target: canvas; function onPainted() { if (index === win.cur && !win.playing) thumb.requestPaint(); } }
                                onPaint: { const c = getContext("2d"); c.reset(); c.scale(width / 734, height / 494); win.drawStrokes(c, strokes || [], 1); }
                            }
                            Text { anchors.right: parent.right; anchors.bottom: parent.bottom; anchors.margins: 4; text: index + 1; color: "#0e7490"; font.bold: true }
                            MouseArea { anchors.fill: parent; onClicked: { win.playing = false; win.cur = index; canvas.requestPaint(); } }
                        }
                    }
                }
            }
        }
    }
}

// Ollie Paint (ollie-paint): draw with the mouse. Brush, pencil, eraser,
// fill bucket, line, rectangle and circle; a palette and any colour you
// like; sizes; undo and redo (Ctrl+Z, Ctrl+Y); a new page, a picture to
// draw on, and Save (a PNG in Pictures/Drawings).
// Says on the console: OLLIE-ACTION saved FILE.
// Argument: out=DIR (where drawings are saved).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Dialogs

ApplicationWindow {
    id: win
    width: 1180
    height: 800
    visible: true
    title: "Ollie Paint"
    color: "#E4E1DA" // a desk, with the paper on it

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }

    property string tool: "brush"      // brush, pencil, eraser, fill, line, rect, circle
    property color ink: "#22252B"
    property int size: 10
    property bool filled: false        // rectangles and circles: filled or outlined
    property string note: ""
    readonly property var colours: ["#22252B", "#6B6F78", "#FFFFFF", "#7E5634", "#C8473D", "#E07A2F", "#D9A021", "#F2D06B",
        "#4F8A45", "#8DB86B", "#257A78", "#7FC4C0", "#336B9E", "#8FB7DD", "#6A4E9C", "#D0587E"]
    readonly property var tools: [["brush", "Brush"], ["pencil", "Pencil"], ["eraser", "Eraser"], ["fill", "Fill"],
        ["line", "Line"], ["rect", "Rectangle"], ["circle", "Circle"]]

    // --- Undo and redo: the page's pixels before each change ---
    property var undos: []
    property var redos: []
    function remember() {
        const c = paper.getContext("2d");
        const u = undos.slice(-24); u.push(c.getImageData(0, 0, paper.width, paper.height));
        undos = u; redos = [];
    }
    function undo() {
        if (!undos.length) return;
        const c = paper.getContext("2d");
        const r = redos.slice(); r.push(c.getImageData(0, 0, paper.width, paper.height)); redos = r;
        const u = undos.slice(); c.drawImage(u.pop(), 0, 0); undos = u;
        paper.requestPaint();
    }
    function redo() {
        if (!redos.length) return;
        const c = paper.getContext("2d");
        const u = undos.slice(); u.push(c.getImageData(0, 0, paper.width, paper.height)); undos = u;
        const r = redos.slice(); c.drawImage(r.pop(), 0, 0); redos = r;
        paper.requestPaint();
    }
    function newPage() {
        remember();
        const c = paper.getContext("2d");
        c.fillStyle = "white"; c.fillRect(0, 0, paper.width, paper.height);
        paper.requestPaint(); note = "";
    }
    function save() {
        const d = new Date(), two = n => (n < 10 ? "0" : "") + n;
        const f = arg("out") + "/drawing-" + d.getFullYear() + "-" + two(d.getMonth() + 1) + "-" + two(d.getDate())
            + "-" + two(d.getHours()) + two(d.getMinutes()) + two(d.getSeconds()) + ".png";
        if (paper.save(f)) { note = "Saved in Pictures, Drawings."; console.log("OLLIE-ACTION saved " + f); }
        else note = "It could not be saved.";
    }

    // Fill: the area of one colour around (x, y), in the ink (line by line).
    function fill(x, y) {
        const c = paper.getContext("2d"), w = paper.width, h = paper.height;
        x = Math.floor(x); y = Math.floor(y);
        if (x < 0 || y < 0 || x >= w || y >= h) return;
        const img = c.getImageData(0, 0, w, h), px = img.data;
        const at = (x + y * w) * 4;
        const t = [px[at], px[at + 1], px[at + 2], px[at + 3]];
        const ink = Qt.color(win.ink), n = [Math.round(ink.r * 255), Math.round(ink.g * 255), Math.round(ink.b * 255), 255];
        if (t[0] === n[0] && t[1] === n[1] && t[2] === n[2] && t[3] === n[3]) return;
        // Close colours count too (the edges of soft lines).
        const same = i => Math.abs(px[i] - t[0]) + Math.abs(px[i + 1] - t[1]) + Math.abs(px[i + 2] - t[2]) + Math.abs(px[i + 3] - t[3]) < 200;
        const set = i => { px[i] = n[0]; px[i + 1] = n[1]; px[i + 2] = n[2]; px[i + 3] = 255; };
        const stack = [[x, y]];
        while (stack.length) {
            let [sx, sy] = stack.pop();
            let i = (sx + sy * w) * 4;
            while (sx >= 0 && same(i)) { sx--; i -= 4; }
            sx++; i += 4;
            let up = false, down = false;
            while (sx < w && same(i)) {
                set(i);
                if (sy > 0) { const u = same(i - w * 4); if (u && !up) { stack.push([sx, sy - 1]); up = true; } else if (!u) up = false; }
                if (sy < h - 1) { const d = same(i + w * 4); if (d && !down) { stack.push([sx, sy + 1]); down = true; } else if (!d) down = false; }
                sx++; i += 4;
            }
        }
        c.drawImage(img, 0, 0); // (putImageData does nothing in Qt's canvas here)
        paper.requestPaint();
    }

    Shortcut { sequences: [StandardKey.Undo]; onActivated: win.undo() }
    Shortcut { sequences: [StandardKey.Redo]; onActivated: win.redo() }
    Shortcut { sequence: "Ctrl+Y"; onActivated: win.redo() }
    Shortcut { sequences: [StandardKey.Save]; onActivated: win.save() }

    FileDialog {
        id: picker
        title: "Pick a picture to draw on"
        nameFilters: ["Pictures (*.png *.jpg *.jpeg *.bmp *.gif *.webp)"]
        onAccepted: { win.remember(); paper.loadImage(selectedFile); paper.pending = selectedFile.toString(); }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // Top bar.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 56
            color: "#F4F2EE"
            border.color: "#D3CEC4"
            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 16; anchors.rightMargin: 16
                spacing: 8
                Image { source: "file:///usr/share/icons/hicolor/64x64/apps/ollie-paint.png"; sourceSize: Qt.size(32, 32) }
                Label { text: "Ollie Paint"; color: "#22252B"; font.pixelSize: 20; font.bold: true; rightPadding: 16 }
                Button { text: "New"; onClicked: win.newPage() }
                Button { text: "Open a picture"; onClicked: picker.open() }
                Button { text: "Undo"; enabled: win.undos.length > 0; onClicked: win.undo() }
                Button { text: "Redo"; enabled: win.redos.length > 0; onClicked: win.redo() }
                Item { Layout.fillWidth: true }
                Label { text: win.note; color: "#4F8A45"; font.pixelSize: 14 }
                Button { text: "Save"; highlighted: true; onClicked: win.save() }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 0

            // The tools, down the side.
            Rectangle {
                Layout.fillHeight: true
                Layout.preferredWidth: 132
                color: "#F4F2EE"
                border.color: "#D3CEC4"
                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 10
                    spacing: 4
                    Repeater {
                        model: win.tools
                        delegate: Rectangle {
                            required property var modelData
                            Layout.fillWidth: true
                            height: 40; radius: 4
                            color: win.tool === modelData[0] ? "#22252B" : toolArea.containsMouse ? "#E4E1DA" : "transparent"
                            Label {
                                anchors.verticalCenter: parent.verticalCenter; x: 12
                                text: modelData[1]; font.pixelSize: 15
                                color: win.tool === modelData[0] ? "white" : "#22252B"
                            }
                            MouseArea { id: toolArea; anchors.fill: parent; hoverEnabled: true; onClicked: win.tool = modelData[0] }
                        }
                    }
                    CheckBox {
                        visible: win.tool === "rect" || win.tool === "circle"
                        checked: win.filled; onToggled: win.filled = checked
                        contentItem: Label { text: "Filled"; leftPadding: 26; color: "#22252B"; verticalAlignment: Text.AlignVCenter }
                    }
                    Item { Layout.fillHeight: true }
                    Label { text: "Size " + win.size; color: "#6B6F78"; font.pixelSize: 13 }
                    Slider { Layout.fillWidth: true; from: 1; to: 60; stepSize: 1; value: win.size; onMoved: win.size = value }
                    // How big the brush is, drawn.
                    Item {
                        Layout.fillWidth: true; Layout.preferredHeight: 66
                        Rectangle {
                            anchors.centerIn: parent
                            width: Math.max(2, win.size); height: width; radius: width / 2
                            color: win.tool === "eraser" ? "white" : win.ink
                            border.color: "#B9B4AA"; border.width: win.tool === "eraser" ? 1 : 0
                        }
                    }
                }
            }

            // The paper.
            Item {
                id: desk
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true
                Rectangle { // its shadow
                    x: paper.x + 4; y: paper.y + 4; width: paper.width; height: paper.height; color: "#C8C3B9"
                }
                Canvas {
                    id: paper
                    // A fixed size (a canvas that changes size loses its drawing).
                    width: 1000; height: 620
                    anchors.centerIn: parent
                    property bool ready: false
                    property string pending: ""
                    property real lx: 0
                    property real ly: 0
                    onPaint: {
                        if (ready) return;
                        const c = getContext("2d");
                        c.fillStyle = "white"; c.fillRect(0, 0, width, height);
                        ready = true;
                    }
                    onImageLoaded: {
                        if (!pending) return;
                        const c = getContext("2d");
                        c.fillStyle = "white"; c.fillRect(0, 0, width, height);
                        // The picture, as big as fits, in the middle.
                        const img = c.createImageData(pending);
                        const iw = img.width || width, ih = img.height || height, k = Math.min(width / iw, height / ih, 1);
                        c.drawImage(pending, (width - iw * k) / 2, (height - ih * k) / 2, iw * k, ih * k);
                        unloadImage(pending); pending = "";
                        requestPaint();
                    }
                    function line(c, x1, y1, x2, y2) {
                        c.beginPath(); c.moveTo(x1, y1); c.lineTo(x2, y2); c.stroke();
                    }
                }
                // A shape, while it is being drawn (on top; then onto the paper).
                Canvas {
                    id: preview
                    anchors.fill: paper
                    property real x1: 0
                    property real y1: 0
                    property real x2: 0
                    property real y2: 0
                    property bool on: false
                    function shape(c) {
                        c.lineWidth = win.size; c.lineCap = "round"; c.lineJoin = "round";
                        c.strokeStyle = win.ink; c.fillStyle = win.ink;
                        c.beginPath();
                        if (win.tool === "line") { c.moveTo(x1, y1); c.lineTo(x2, y2); c.stroke(); return; }
                        if (win.tool === "rect") c.rect(Math.min(x1, x2), Math.min(y1, y2), Math.abs(x2 - x1), Math.abs(y2 - y1));
                        else c.ellipse(Math.min(x1, x2), Math.min(y1, y2), Math.abs(x2 - x1), Math.abs(y2 - y1));
                        if (win.filled) c.fill(); else c.stroke();
                    }
                    onPaint: {
                        const c = getContext("2d");
                        c.clearRect(0, 0, width, height);
                        if (on) shape(c);
                    }
                }
                MouseArea {
                    anchors.fill: paper
                    cursorShape: win.tool === "fill" ? Qt.PointingHandCursor : Qt.CrossCursor
                    onPressed: (m) => {
                        win.remember();
                        if (win.tool === "fill") { win.fill(m.x, m.y); return; }
                        if (["line", "rect", "circle"].includes(win.tool)) {
                            preview.x1 = preview.x2 = m.x; preview.y1 = preview.y2 = m.y; preview.on = true; preview.requestPaint();
                            return;
                        }
                        paper.lx = m.x; paper.ly = m.y;
                        dot(m.x, m.y);
                    }
                    onPositionChanged: (m) => {
                        if (preview.on) { preview.x2 = m.x; preview.y2 = m.y; preview.requestPaint(); return; }
                        if (win.tool === "fill") return;
                        const c = paper.getContext("2d");
                        brush(c);
                        paper.line(c, paper.lx, paper.ly, m.x, m.y);
                        paper.lx = m.x; paper.ly = m.y;
                        paper.requestPaint();
                    }
                    onReleased: {
                        if (!preview.on) return;
                        preview.on = false;
                        preview.shape(paper.getContext("2d"));
                        preview.requestPaint(); paper.requestPaint();
                    }
                    function brush(c) {
                        c.lineCap = win.tool === "pencil" ? "square" : "round"; c.lineJoin = "round";
                        c.lineWidth = win.tool === "pencil" ? Math.max(1, Math.round(win.size / 3)) : win.size;
                        c.strokeStyle = win.tool === "eraser" ? "white" : win.ink;
                        c.fillStyle = c.strokeStyle;
                    }
                    function dot(x, y) {
                        const c = paper.getContext("2d");
                        brush(c);
                        c.beginPath();
                        if (win.tool === "pencil") c.fillRect(x - c.lineWidth / 2, y - c.lineWidth / 2, c.lineWidth, c.lineWidth);
                        else { c.arc(x, y, c.lineWidth / 2, 0, 2 * Math.PI); c.fill(); }
                        paper.requestPaint();
                    }
                }
            }
        }

        // The colours along the bottom.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 64
            color: "#F4F2EE"
            border.color: "#D3CEC4"
            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 16; anchors.rightMargin: 16
                spacing: 6
                Rectangle { // the colour now
                    width: 44; height: 44; radius: 4; color: win.ink
                    border.color: "#22252B"; border.width: 2
                }
                Item { width: 10 }
                Repeater {
                    model: win.colours
                    delegate: Rectangle {
                        required property string modelData
                        width: 34; height: 34; radius: 4
                        color: modelData
                        border.color: Qt.colorEqual(win.ink, modelData) ? "#22252B" : "#C8C3B9"
                        border.width: Qt.colorEqual(win.ink, modelData) ? 3 : 1
                        MouseArea { anchors.fill: parent; onClicked: { win.ink = modelData; if (win.tool === "eraser") win.tool = "brush"; } }
                    }
                }
                Button { text: "More colours"; onClicked: { colourDialog.selectedColor = win.ink; colourDialog.open(); } }
                Item { Layout.fillWidth: true }
            }
        }
    }
    ColorDialog {
        id: colourDialog
        title: "Pick a colour"
        onAccepted: { win.ink = selectedColor; if (win.tool === "eraser") win.tool = "brush"; }
    }
}

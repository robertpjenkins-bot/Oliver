// Mouse Maker (ollie-mouse): draw your own mouse pointer, square by square,
// try it, and use it as the computer's real one.
// Says on the console: OLLIE-ACTION use X Y GRID links|arrow (X, Y: the tip,
// the square that clicks; GRID: 24 x 24 squares, "RRGGBB" or "......" each),
// and OLLIE-ACTION normal (back to the pointer from before).
// Argument: saved="X Y GRID links|arrow" (the last one used).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1100
    height: 760
    visible: true
    title: "Mouse Maker"
    color: "#2F5D50" // a green cutting mat

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }

    readonly property int n: 24
    readonly property string outlineColour: "#22252B"
    property var cells: []          // n * n: "" (see-through) or "#RRGGBB"
    property int hx: 0              // the tip: the square that clicks
    property int hy: 0
    property string tool: "pen"     // pen, eraser, fill, tip
    property color ink: "#C8473D"
    property bool links: true
    property string note: ""
    property var undos: []
    readonly property var colours: ["#22252B", "#6B6F78", "#FFFFFF", "#7E5634", "#C8473D", "#E07A2F", "#D9A021", "#F2D06B",
        "#4F8A45", "#8DB86B", "#257A78", "#7FC4C0", "#336B9E", "#8FB7DD", "#6A4E9C", "#D0587E"]
    readonly property var tools: [["pen", "Pen"], ["eraser", "Eraser"], ["fill", "Fill"], ["tip", "Tip (what clicks)"]]

    function changed() { grid.requestPaint(); small.requestPaint(); big.requestPaint(); dark.requestPaint(); tryIt.requestPaint(); }
    function remember() { const u = undos.slice(-30); u.push({ c: cells.slice(), x: hx, y: hy }); undos = u; }
    function undo() {
        if (!undos.length) return;
        const u = undos.slice(), s = u.pop();
        cells = s.c; hx = s.x; hy = s.y; undos = u; changed();
    }

    // --- Ready-made ones to start from ---
    function blank() { const c = []; for (let i = 0; i < n * n; i++) c.push(""); return c; }
    function arrow() {
        const rows = ["X", "XX", "X.X", "X..X", "X...X", "X....X", "X.....X", "X......X", "X.......X", "X........X",
            "X.........X", "X......XXXXX", "X...X..X", "X..XX..X", "X.X  X..X", "XX   X..X", "X     X..X", "      X..X", "       XX"];
        const c = blank();
        for (let y = 0; y < rows.length; y++)
            for (let x = 0; x < rows[y].length; x++)
                if (rows[y][x] !== " ") c[y * n + x] = rows[y][x] === "X" ? outlineColour : "#FFFFFF";
        return { c: c, x: 0, y: 0 };
    }
    function star() {
        const c = blank(), pts = [];
        for (let i = 0; i < 10; i++) {
            const r = i % 2 ? 4.6 : 11, a = Math.PI * i / 5 - Math.PI / 2;
            pts.push([12 + r * Math.cos(a), 12.4 + r * Math.sin(a)]);
        }
        for (let y = 0; y < n; y++)
            for (let x = 0; x < n; x++) {
                let inside = false;
                const px = x + 0.5, py = y + 0.5;
                for (let i = 0, j = pts.length - 1; i < pts.length; j = i++)
                    if ((pts[i][1] > py) !== (pts[j][1] > py)
                            && px < (pts[j][0] - pts[i][0]) * (py - pts[i][1]) / (pts[j][1] - pts[i][1]) + pts[i][0])
                        inside = !inside;
                if (inside) c[y * n + x] = "#D9A021";
            }
        return { c: outlined(c), x: 12, y: 2 };
    }
    function start(s) { remember(); cells = s.c; hx = s.x; hy = s.y; changed(); }

    // A dark edge round the drawing, so it shows on light and dark things.
    function outlined(c) {
        const o = c.slice();
        for (let y = 0; y < n; y++)
            for (let x = 0; x < n; x++) {
                if (c[y * n + x]) continue;
                if ((x > 0 && c[y * n + x - 1]) || (x < n - 1 && c[y * n + x + 1])
                        || (y > 0 && c[(y - 1) * n + x]) || (y < n - 1 && c[(y + 1) * n + x]))
                    o[y * n + x] = outlineColour;
            }
        return o;
    }

    function fill(x, y, to) {
        const from = cells[y * n + x];
        if (from === to) return;
        const stack = [[x, y]];
        while (stack.length) {
            const [px, py] = stack.pop();
            if (px < 0 || py < 0 || px >= n || py >= n || cells[py * n + px] !== from) continue;
            cells[py * n + px] = to;
            stack.push([px + 1, py], [px - 1, py], [px, py + 1], [px, py - 1]);
        }
    }

    function hexGrid() {
        let s = "";
        for (let i = 0; i < n * n; i++)
            s += cells[i] ? cells[i].substring(1).toUpperCase() : "......";
        return s;
    }
    function load(text) {
        const p = text.trim().split(" ");
        if (p.length < 3 || p[2].length !== n * n * 6) return false;
        const c = [];
        for (let i = 0; i < n * n; i++) {
            const h = p[2].substring(i * 6, i * 6 + 6);
            c.push(h === "......" ? "" : "#" + h);
        }
        cells = c; hx = parseInt(p[0]) || 0; hy = parseInt(p[1]) || 0; links = p[3] !== "arrow";
        return true;
    }
    function use() {
        if (cells.every(c => !c)) { note = "Draw something first."; return; }
        if (!cells[hy * n + hx]) { note = "The tip is on an empty square: put it on your drawing (Tip)."; return; }
        console.log("OLLIE-ACTION use " + hx + " " + hy + " " + hexGrid() + " " + (links ? "links" : "arrow"));
        note = "This is your mouse now. (It can take a moment.)";
    }

    // Draws the pointer: each square as k x k pixels, at (ox, oy).
    function paintPointer(c, k, ox, oy) {
        for (let y = 0; y < n; y++)
            for (let x = 0; x < n; x++) {
                const v = cells[y * n + x];
                if (!v) continue;
                c.fillStyle = v; c.fillRect(ox + x * k, oy + y * k, k, k);
            }
    }

    Component.onCompleted: {
        if (!load(arg("saved"))) { const a = arrow(); cells = a.c; hx = a.x; hy = a.y; }
        changed();
    }

    Shortcut { sequences: [StandardKey.Undo]; onActivated: win.undo() }

    component Panel: Rectangle { color: "#F7F4EC"; radius: 6; border.color: "#1F4237"; border.width: 1 }
    component Pick: Rectangle { // a tool or a ready-made one
        id: pick
        property string text
        property bool on: false
        signal clicked
        Layout.fillWidth: true
        height: 38; radius: 4
        color: on ? "#2F5D50" : pickArea.containsMouse ? "#E6E0D2" : "transparent"
        border.color: on ? "#2F5D50" : "#D3CBB8"
        Label { anchors.centerIn: parent; text: pick.text; font.pixelSize: 15; color: pick.on ? "white" : "#22252B" }
        MouseArea { id: pickArea; anchors.fill: parent; hoverEnabled: true; onClicked: pick.clicked() }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 14

        RowLayout {
            spacing: 12
            Image { source: "file:///usr/share/icons/hicolor/64x64/apps/ollie-mouse.png"; sourceSize: Qt.size(40, 40) }
            ColumnLayout {
                spacing: 0
                Label { text: "Mouse Maker"; color: "white"; font.pixelSize: 24; font.bold: true }
                Label { text: "Draw your own mouse pointer, then use it."; color: "#CFE0D8"; font.pixelSize: 14 }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 14

            // Tools and the ready-made ones.
            Panel {
                Layout.preferredWidth: 190
                Layout.fillHeight: true
                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 12
                    spacing: 6
                    Label { text: "Tools"; font.bold: true; color: "#22252B" }
                    Label { text: "The orange square is the tip: the part that clicks."; color: "#6B6F78"; font.pixelSize: 12; wrapMode: Text.Wrap; Layout.fillWidth: true }
                    Repeater {
                        model: win.tools
                        delegate: Pick { required property var modelData; text: modelData[1]; on: win.tool === modelData[0]; onClicked: win.tool = modelData[0] }
                    }
                    Pick { text: "Undo"; enabled: win.undos.length > 0; opacity: enabled ? 1 : 0.4; onClicked: win.undo() }
                    Item { height: 8 }
                    Label { text: "Start from"; font.bold: true; color: "#22252B" }
                    Pick { text: "An arrow"; onClicked: win.start(win.arrow()) }
                    Pick { text: "A star"; onClicked: win.start(win.star()) }
                    Pick { text: "Nothing"; onClicked: win.start({ c: win.blank(), x: 0, y: 0 }) }
                    Item { height: 8 }
                    Pick { text: "Add an outline"; onClicked: { win.remember(); win.cells = win.outlined(win.cells); win.changed(); } }
                    Item { Layout.fillHeight: true }
                }
            }

            // The drawing, big.
            Panel {
                Layout.preferredWidth: grid.width + 32
                Layout.fillHeight: true
                Canvas {
                    id: grid
                    readonly property int k: 22
                    width: win.n * k; height: width
                    anchors.centerIn: parent
                    onPaint: {
                        const c = getContext("2d");
                        for (let y = 0; y < win.n; y++)
                            for (let x = 0; x < win.n; x++) {
                                const v = win.cells[y * win.n + x];
                                c.fillStyle = v ? v : ((x + y) % 2 ? "#E9E4D8" : "#FFFFFF"); // see-through: a check
                                c.fillRect(x * k, y * k, k, k);
                            }
                        c.strokeStyle = "rgba(34,37,43,0.12)"; c.lineWidth = 1;
                        c.beginPath();
                        for (let i = 0; i <= win.n; i++) {
                            c.moveTo(i * k + 0.5, 0); c.lineTo(i * k + 0.5, height);
                            c.moveTo(0, i * k + 0.5); c.lineTo(width, i * k + 0.5);
                        }
                        c.stroke();
                        // The tip: its square, marked (orange, with white round it).
                        c.lineWidth = 4; c.strokeStyle = "#FFFFFF";
                        c.strokeRect(win.hx * k + 2, win.hy * k + 2, k - 4, k - 4);
                        c.lineWidth = 2; c.strokeStyle = "#E07A2F";
                        c.strokeRect(win.hx * k + 2, win.hy * k + 2, k - 4, k - 4);
                    }
                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.CrossCursor
                        property int last: -1
                        function at(m) {
                            const x = Math.floor(m.x / grid.k), y = Math.floor(m.y / grid.k);
                            return x < 0 || y < 0 || x >= win.n || y >= win.n ? -1 : y * win.n + x;
                        }
                        function put(i) {
                            if (i < 0 || i === last) return;
                            last = i;
                            win.cells[i] = win.tool === "eraser" ? "" : win.ink.toString().toUpperCase().substring(0, 7);
                            win.changed();
                        }
                        onPressed: (m) => {
                            const i = at(m);
                            if (i < 0) return;
                            win.remember(); last = -1;
                            if (win.tool === "tip") { win.hx = i % win.n; win.hy = Math.floor(i / win.n); win.changed(); return; }
                            if (win.tool === "fill") { win.fill(i % win.n, Math.floor(i / win.n), win.ink.toString().toUpperCase().substring(0, 7)); win.changed(); return; }
                            put(i);
                        }
                        onPositionChanged: (m) => { if (win.tool === "pen" || win.tool === "eraser") put(at(m)); }
                    }
                }
            }

            // How it looks, and trying it.
            ColumnLayout {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 14
                Panel {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 150
                    ColumnLayout {
                        anchors.fill: parent; anchors.margins: 12; spacing: 8
                        Label { text: "How it looks"; font.bold: true; color: "#22252B" }
                        RowLayout {
                            spacing: 14
                            Rectangle { // real size, on white
                                width: 72; height: 72; color: "white"; border.color: "#D3CBB8"
                                Canvas { id: small; anchors.centerIn: parent; width: win.n; height: win.n
                                    onPaint: { const c = getContext("2d"); c.clearRect(0, 0, width, height); win.paintPointer(c, 1, 0, 0); } }
                            }
                            Rectangle { // on a dark window
                                width: 72; height: 72; color: "#22252B"
                                Canvas { id: dark; anchors.centerIn: parent; width: win.n; height: win.n
                                    onPaint: { const c = getContext("2d"); c.clearRect(0, 0, width, height); win.paintPointer(c, 1, 0, 0); } }
                            }
                            Rectangle { // twice as big (big screens)
                                width: 72; height: 72; color: "#8FB7DD"
                                Canvas { id: big; anchors.centerIn: parent; width: win.n * 2; height: win.n * 2
                                    onPaint: { const c = getContext("2d"); c.clearRect(0, 0, width, height); win.paintPointer(c, 2, 0, 0); } }
                            }
                        }
                    }
                }
                Panel {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    ColumnLayout {
                        anchors.fill: parent; anchors.margins: 12; spacing: 8
                        Label { text: "Try it: move the mouse in here, and click the buttons"; font.bold: true; color: "#22252B"; wrapMode: Text.Wrap; Layout.fillWidth: true }
                        Rectangle {
                            id: pad
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            color: "#FFFFFF"; border.color: "#D3CBB8"
                            clip: true
                            property int hits: 0
                            Row {
                                anchors.centerIn: parent
                                spacing: 10
                                Repeater {
                                    model: 3
                                    delegate: Rectangle {
                                        required property int index
                                        width: 46; height: 46; radius: 23
                                        color: ["#C8473D", "#4F8A45", "#336B9E"][index]
                                        scale: hitArea.pressed ? 0.85 : 1
                                        MouseArea { id: hitArea; anchors.fill: parent; cursorShape: Qt.BlankCursor; onClicked: pad.hits++ }
                                    }
                                }
                            }
                            Label { anchors.bottom: parent.bottom; anchors.horizontalCenter: parent.horizontalCenter; bottomPadding: 8
                                text: pad.hits ? "Clicks: " + pad.hits : ""; color: "#6B6F78" }
                            MouseArea {
                                id: padArea
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.BlankCursor
                                acceptedButtons: Qt.NoButton
                            }
                            Canvas {
                                id: tryIt
                                width: win.n; height: win.n
                                visible: padArea.containsMouse
                                x: padArea.mouseX - win.hx; y: padArea.mouseY - win.hy
                                onPaint: { const c = getContext("2d"); c.clearRect(0, 0, width, height); win.paintPointer(c, 1, 0, 0); }
                            }
                        }
                    }
                }
            }
        }

        // The colours, and using it.
        Panel {
            Layout.fillWidth: true
            Layout.preferredHeight: 116
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 12; spacing: 10
                RowLayout {
                    spacing: 6
                    Rectangle { width: 40; height: 40; radius: 4; color: win.tool === "eraser" ? "transparent" : win.ink; border.color: "#22252B"; border.width: 2 }
                    Item { width: 8 }
                    Repeater {
                        model: win.colours
                        delegate: Rectangle {
                            required property string modelData
                            width: 32; height: 32; radius: 4
                            color: modelData
                            border.color: Qt.colorEqual(win.ink, modelData) ? "#22252B" : "#C8C3B9"
                            border.width: Qt.colorEqual(win.ink, modelData) ? 3 : 1
                            MouseArea { anchors.fill: parent; onClicked: { win.ink = modelData; if (win.tool !== "fill") win.tool = "pen"; } }
                        }
                    }
                }
                RowLayout {
                    Layout.fillWidth: true
                    spacing: 10
                    CheckBox { id: linksBox; checked: win.links; onToggled: win.links = checked }
                    Label { text: "Over links and buttons too"; color: "#22252B"; MouseArea { anchors.fill: parent; onClicked: win.links = !win.links } }
                    Label { text: win.note; color: "#2F5D50"; font.bold: true; Layout.fillWidth: true; horizontalAlignment: Text.AlignRight; elide: Text.ElideRight }
                    Button { text: "Back to the normal mouse"; onClicked: { console.log("OLLIE-ACTION normal"); win.note = "The normal mouse is back."; } }
                    Button { text: "Use this mouse"; highlighted: true; font.bold: true; onClicked: win.use() }
                }
            }
        }
    }
}

// Wallpaper Maker (ollie-wallpaper-maker): type your name and it makes a
// background like Ollie OS's own, "<name> OS", in the colours you pick; draw
// on it, or on a plain background, with the mouse or a finger. "Make it my
// background" saves it (as the file named last on the command line) and says
// what was chosen on the console: OLLIE-RESULT rename|keep NAME.
// For tests: --auto-save (with a drawing) saves it straight away.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1100
    height: 800
    visible: true
    title: "Wallpaper Maker"

    readonly property var args: Qt.application.arguments
    readonly property string outPath: args[args.length - 1]
    readonly property string osName: (nameField.text.trim() || "Ollie") + " OS"
    readonly property var themes: [
        { name: "Teal", c1: "#14b8a6", c2: "#0f766e", c3: "#083c38", glow: "rgba(94,234,212,0.55)", top: "#ffffff", left: "#9eeee4", right: "#2dd4bf", line: "#0b4f4a" },
        { name: "Purple", c1: "#a855f7", c2: "#7e22ce", c3: "#2e1065", glow: "rgba(216,180,254,0.55)", top: "#ffffff", left: "#e9d5ff", right: "#c084fc", line: "#3b0764" },
        { name: "Blue", c1: "#3b82f6", c2: "#1d4ed8", c3: "#172554", glow: "rgba(147,197,253,0.55)", top: "#ffffff", left: "#dbeafe", right: "#60a5fa", line: "#172554" },
        { name: "Red", c1: "#f43f5e", c2: "#be123c", c3: "#4c0519", glow: "rgba(253,164,175,0.55)", top: "#ffffff", left: "#ffe4e6", right: "#fb7185", line: "#4c0519" },
        { name: "Green", c1: "#22c55e", c2: "#15803d", c3: "#052e16", glow: "rgba(134,239,172,0.55)", top: "#ffffff", left: "#dcfce7", right: "#4ade80", line: "#052e16" },
        { name: "Orange", c1: "#fb923c", c2: "#ea580c", c3: "#431407", glow: "rgba(253,186,116,0.55)", top: "#ffffff", left: "#ffedd5", right: "#fdba74", line: "#431407" },
        { name: "Night", c1: "#4b5563", c2: "#1f2937", c3: "#030712", glow: "rgba(156,163,175,0.45)", top: "#ffffff", left: "#e5e7eb", right: "#9ca3af", line: "#030712" }
    ]
    property int theme: 0
    readonly property var inks: ["#ffffff", "#111827", "#ef4444", "#f97316", "#facc15", "#22c55e", "#3b82f6", "#a855f7", "#ec4899", "#8b5a2b"]
    property string ink: "#ffffff"
    property int brush: 12
    property var strokes: []

    function redrawAll() { bg.requestPaint(); sketch.requestPaint() }
    onThemeChanged: bg.requestPaint()
    onOsNameChanged: bg.requestPaint()

    function makeIt() {
        art.grabToImage(function (result) {
            if (!result.saveToFile(outPath)) {
                console.log("OLLIE-RESULT failed");
                return;
            }
            console.log("OLLIE-RESULT " + (renameBox.checked ? "rename " : "keep ") + osName);
            Qt.quit();
        }, Qt.size(1920, 1080));
    }

    Component.onCompleted: {
        if (args.indexOf("--auto-save") >= 0) {
            nameField.text = "Bobby";
            theme = 1;
            strokes = [{ color: "#facc15", size: 18, pts: [{ x: 200, y: 200 }, { x: 400, y: 320 }, { x: 600, y: 200 }] }];
            redrawAll();
            autoSave.start();
        }
    }
    Timer { id: autoSave; interval: 1500; onTriggered: win.makeIt() }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 12
        spacing: 10

        TabBar {
            id: tabs
            Layout.fillWidth: true
            TabButton { text: "Your name" }
            TabButton { text: "Draw" }
        }

        StackLayout {
            currentIndex: tabs.currentIndex
            Layout.fillWidth: true
            Layout.preferredHeight: 96

            ColumnLayout {
                RowLayout {
                    Label { text: "Your name:" }
                    TextField {
                        id: nameField
                        text: "Ollie"
                        placeholderText: "Type your name"
                        maximumLength: 20
                        validator: RegularExpressionValidator { regularExpression: /[A-Za-z0-9 ._!-]*/ }
                        Layout.preferredWidth: 260
                    }
                    Label { text: "→ " + win.osName; font.bold: true }
                }
                RowLayout {
                    Label { text: "Colours:" }
                    Repeater {
                        model: win.themes
                        delegate: Button {
                            required property var modelData
                            required property int index
                            text: modelData.name
                            highlighted: win.theme === index
                            onClicked: win.theme = index
                        }
                    }
                }
                RowLayout {
                    CheckBox { id: cubeBox; text: "The cube"; checked: true; onToggled: bg.requestPaint() }
                    CheckBox { id: nameBox; text: "The name"; checked: true; onToggled: bg.requestPaint() }
                }
            }

            ColumnLayout {
                RowLayout {
                    Label { text: "Colour:" }
                    Repeater {
                        model: win.inks
                        delegate: Rectangle {
                            required property string modelData
                            width: 34; height: 34; radius: 17
                            color: modelData
                            border.width: win.ink === modelData ? 4 : 1
                            border.color: win.ink === modelData ? "#22d3ee" : "#6b7280"
                            MouseArea { anchors.fill: parent; onClicked: win.ink = parent.modelData }
                        }
                    }
                }
                RowLayout {
                    Label { text: "Brush:" }
                    Button { text: "Thin"; highlighted: win.brush === 5; onClicked: win.brush = 5 }
                    Button { text: "Medium"; highlighted: win.brush === 12; onClicked: win.brush = 12 }
                    Button { text: "Thick"; highlighted: win.brush === 30; onClicked: win.brush = 30 }
                    Button { text: "Huge"; highlighted: win.brush === 70; onClicked: win.brush = 70 }
                    Item { width: 20 }
                    Button { text: "Undo"; enabled: win.strokes.length > 0; onClicked: { win.strokes.pop(); win.strokes = win.strokes; sketch.requestPaint() } }
                    Button { text: "Clear drawing"; enabled: win.strokes.length > 0; onClicked: { win.strokes = []; sketch.requestPaint() } }
                }
            }
        }

        // The background, 1920 x 1080, shrunk to fit the window.
        Item {
            id: frame
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            Item {
                id: art
                width: 1920
                height: 1080
                anchors.centerIn: parent
                scale: Math.min(frame.width / 1920, frame.height / 1080)

                Canvas {
                    id: bg
                    anchors.fill: parent
                    onPaint: {
                        const ctx = getContext("2d");
                        const t = win.themes[win.theme];
                        const g = ctx.createRadialGradient(960, 454, 0, 960, 454, 1100);
                        g.addColorStop(0, t.c1);
                        g.addColorStop(0.55, t.c2);
                        g.addColorStop(1, t.c3);
                        ctx.fillStyle = g;
                        ctx.fillRect(0, 0, 1920, 1080);
                        if (cubeBox.checked) {
                            const glow = ctx.createRadialGradient(960, 450, 0, 960, 450, 330);
                            glow.addColorStop(0, t.glow);
                            glow.addColorStop(1, "rgba(0,0,0,0)");
                            ctx.fillStyle = glow;
                            ctx.fillRect(630, 120, 660, 660);
                            const s = 1.9;
                            const face = (pts, colour) => {
                                ctx.beginPath();
                                ctx.moveTo(960 + pts[0] * s, 450 + pts[1] * s);
                                for (let i = 2; i < pts.length; i += 2)
                                    ctx.lineTo(960 + pts[i] * s, 450 + pts[i + 1] * s);
                                ctx.closePath();
                                ctx.fillStyle = colour;
                                ctx.fill();
                                ctx.lineWidth = 3 * s;
                                ctx.lineJoin = "round";
                                ctx.strokeStyle = t.line;
                                ctx.stroke();
                            };
                            face([0, -100, 86.6, -50, 0, 0, -86.6, -50], t.top);
                            face([-86.6, -50, 0, 0, 0, 100, -86.6, 50], t.left);
                            face([0, 0, 86.6, -50, 86.6, 50, 0, 100], t.right);
                        }
                        if (nameBox.checked) {
                            ctx.fillStyle = "#ffffff";
                            ctx.font = "bold 84px 'Noto Sans'";
                            ctx.textAlign = "center";
                            ctx.fillText(win.osName, 960, cubeBox.checked ? 760 : 570);
                        }
                    }
                }

                Canvas {
                    id: sketch
                    anchors.fill: parent
                    onPaint: {
                        const ctx = getContext("2d");
                        ctx.clearRect(0, 0, 1920, 1080);
                        ctx.lineCap = "round";
                        ctx.lineJoin = "round";
                        for (const st of win.strokes) {
                            ctx.strokeStyle = st.color;
                            ctx.fillStyle = st.color;
                            ctx.lineWidth = st.size;
                            if (st.pts.length === 1) {
                                ctx.beginPath();
                                ctx.arc(st.pts[0].x, st.pts[0].y, st.size / 2, 0, 2 * Math.PI);
                                ctx.fill();
                                continue;
                            }
                            ctx.beginPath();
                            ctx.moveTo(st.pts[0].x, st.pts[0].y);
                            for (let i = 1; i < st.pts.length; i++)
                                ctx.lineTo(st.pts[i].x, st.pts[i].y);
                            ctx.stroke();
                        }
                    }
                    MouseArea {
                        anchors.fill: parent
                        enabled: tabs.currentIndex === 1
                        cursorShape: enabled ? Qt.CrossCursor : Qt.ArrowCursor
                        onPressed: (m) => {
                            win.strokes.push({ color: win.ink, size: win.brush, pts: [{ x: m.x, y: m.y }] });
                            sketch.requestPaint();
                        }
                        onPositionChanged: (m) => {
                            win.strokes[win.strokes.length - 1].pts.push({ x: m.x, y: m.y });
                            sketch.requestPaint();
                        }
                        onReleased: win.strokes = win.strokes
                    }
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            CheckBox {
                id: renameBox
                text: "Also call this computer's OS \"" + win.osName + "\" (start-up menu and About This System)"
            }
            Item { Layout.fillWidth: true }
            Button {
                text: "Make it my background"
                highlighted: true
                font.bold: true
                onClicked: win.makeIt()
            }
        }
    }
}

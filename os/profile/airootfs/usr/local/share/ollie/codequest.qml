// Coding Quest (ollie-codequest): learn coding by guiding the Ollie cube to
// the star with code blocks: Forward, Turn left, Turn right, and (from
// level 4) Repeat, which does the next block several times. Build the
// program, press Run, and watch it go. 10 levels; fewer blocks, more stars.
// Says on the console: OLLIE-ACTION trophy ...
// Argument: config=FILE (the stars won).
// (Every level was checked to have a way to the star; "par" is the fewest
// blocks for it.)
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 1120
    height: 720
    visible: true
    title: "Coding Quest"
    color: "#f0fdf4"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function trophy(w) { console.log("OLLIE-ACTION trophy " + w) }

    readonly property var levels: [{"name": "Walk to the star", "repeat": false, "rows": ["........", "........", "S..*....", "........", "........", "........"], "par": 3}, {"name": "Turn the corner", "repeat": false, "rows": ["S.##....", "........", ".*......", "........", "........", "........"], "par": 4}, {"name": "Zig zag", "repeat": false, "rows": ["S.......", "..#.....", "....*...", "........", "........", "........"], "par": 7}, {"name": "Repeat!", "repeat": true, "rows": ["........", "........", "........", "S......*", "........", "........"], "par": 2}, {"name": "Big L", "repeat": true, "rows": ["S.......", "........", "........", "........", ".....*..", "........"], "par": 5}, {"name": "Around the rocks", "repeat": true, "rows": ["S..#....", "...#....", "...#..*.", "........", "........", "........"], "par": 8}, {"name": "The stairs", "repeat": true, "rows": ["S#######", ".......#", "######.#", "#......#", "#.######", "#......*"], "par": 17}, {"name": "Snake", "repeat": true, "rows": ["S......#", "######.#", "#......#", "#.######", "#......*", "########"], "par": 14}, {"name": "Spiral", "repeat": true, "rows": ["S.......", "#######.", "#.....#.", "#.#*..#.", "#.####..", "#......."], "par": 16}, {"name": "The big maze", "repeat": true, "rows": ["S#......", ".#.####.", ".#.#..#.", ".#.#*.#.", "...#..#.", "#......."], "par": 13}]
    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property string stars: "[]"
    }
    property var stars: { try { return JSON.parse(saved.stars); } catch (e) { return []; } }
    function starsOf(i) { return stars[i] || 0 }
    readonly property int unlocked: { let u = 0; while (u < levels.length - 1 && starsOf(u) > 0) u++; return u; }

    property int level: Math.min(unlocked, levels.length - 1)
    readonly property var lv: levels[level]
    readonly property int cols: 8
    readonly property int rowsN: 6
    readonly property int cell: 64
    function at(x, y) { return x < 0 || y < 0 || x >= cols || y >= rowsN ? "#" : lv.rows[y][x] }
    function find(ch) { for (let y = 0; y < rowsN; y++) for (let x = 0; x < cols; x++) if (lv.rows[y][x] === ch) return { x: x, y: y }; return { x: 0, y: 0 }; }

    property var program: []            // {k: F|L|R|P, n}
    property int px: 0
    property int py: 0
    property int dir: 0                 // 0 right, 1 down, 2 left, 3 up
    property bool running: false
    property var steps: []
    property int stepAt: 0
    property int activeBlock: -1
    property string message: ""
    readonly property var dx: [1, 0, -1, 0]
    readonly property var dy: [0, 1, 0, -1]

    function reset() { const s = find("S"); px = s.x; py = s.y; dir = 0; running = false; activeBlock = -1; }
    onLevelChanged: { program = []; reset(); message = lv.repeat && level === 3 ? "New block: 🔁 Repeat does the next block again and again!" : ""; }
    Component.onCompleted: reset()
    function addBlock(k) { if (!running) { program = program.concat([{ k: k, n: 2 }]); message = ""; } }

    function run() {
        reset();
        const s = [];
        for (let i = 0; i < program.length; i++) {
            const b = program[i];
            if (b.k === "P") {
                const next = program[i + 1];
                if (next && next.k !== "P") { for (let r = 0; r < b.n; r++) s.push({ k: next.k, block: i + 1 }); i++; }
            } else s.push({ k: b.k, block: i });
        }
        steps = s; stepAt = 0; running = true; message = "";
    }
    Timer {
        interval: 380
        running: win.running
        repeat: true
        onTriggered: {
            if (win.stepAt >= win.steps.length) {
                win.running = false; win.activeBlock = -1;
                if (win.at(win.px, win.py) === "*") win.won();
                else { win.message = "Not at the star yet. Add more blocks!"; }
                return;
            }
            const st = win.steps[win.stepAt++];
            win.activeBlock = st.block;
            if (st.k === "L") win.dir = (win.dir + 3) % 4;
            else if (st.k === "R") win.dir = (win.dir + 1) % 4;
            else {
                const nx = win.px + win.dx[win.dir], ny = win.py + win.dy[win.dir];
                if (win.at(nx, ny) === "#") {
                    win.running = false; bump.start();
                    win.message = "Bump! A rock is in the way. Try again!";
                    return;
                }
                win.px = nx; win.py = ny;
                if (win.at(nx, ny) === "*" && win.stepAt >= win.steps.length) { /* checked next tick */ }
            }
        }
    }
    function won() {
        const n = program.length, got = n <= lv.par ? 3 : n <= lv.par + 3 ? 2 : 1;
        const s = stars.slice(); s[level] = Math.max(s[level] || 0, got);
        saved.stars = JSON.stringify(s); stars = s;
        trophy("max codequest-level " + s.filter(x => x > 0).length);
        message = "🎉 You did it with " + n + " blocks! " + "⭐".repeat(got) + (got < 3 ? "  (Can you do it with " + lv.par + "?)" : "");
        cheer.start();
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 18

        ColumnLayout {
            spacing: 10
            RowLayout {
                Label { text: "🧩 Coding Quest"; font.pixelSize: 28; font.bold: true; color: "#166534" }
                Item { Layout.fillWidth: true }
                Label { text: "Level " + (win.level + 1) + ": " + win.lv.name; font.pixelSize: 18; font.bold: true; color: "#15803d" }
            }
            // The board.
            Rectangle {
                width: win.cols * win.cell + 8; height: win.rowsN * win.cell + 8
                radius: 12; color: "#86efac"; border.color: "#166534"; border.width: 4
                Grid {
                    x: 4; y: 4; columns: win.cols
                    Repeater {
                        model: win.cols * win.rowsN
                        delegate: Rectangle {
                            required property int index
                            readonly property string c: win.lv.rows[Math.floor(index / win.cols)][index % win.cols]
                            width: win.cell; height: win.cell
                            color: (index % win.cols + Math.floor(index / win.cols)) % 2 ? "#bbf7d0" : "#a7f3d0"
                            Text { anchors.centerIn: parent; text: parent.c === "#" ? "🪨" : parent.c === "*" ? "⭐" : ""; font.pixelSize: 40 }
                        }
                    }
                }
                Item {
                    id: hero
                    x: 4 + win.px * win.cell; y: 4 + win.py * win.cell
                    width: win.cell; height: win.cell
                    Behavior on x { NumberAnimation { duration: 300; easing.type: Easing.InOutQuad } }
                    Behavior on y { NumberAnimation { duration: 300; easing.type: Easing.InOutQuad } }
                    PetBody {
                        anchors.centerIn: parent
                        scale: 0.44
                        mood: "happy"
                        dir: win.dir === 2 ? -1 : 1
                        stepping: win.running && win.stepAt % 2 === 0
                    }
                    Text { // which way it faces
                        text: ["▶", "▼", "◀", "▲"][win.dir]
                        color: "#166534"; font.pixelSize: 22; font.bold: true
                        x: [win.cell - 18, win.cell / 2 - 8, -4, win.cell / 2 - 8][win.dir]
                        y: [win.cell / 2 - 14, win.cell - 22, win.cell / 2 - 14, -10][win.dir]
                    }
                    SequentialAnimation on rotation { id: bump; running: false
                        NumberAnimation { to: -12; duration: 70 } NumberAnimation { to: 12; duration: 70 } NumberAnimation { to: 0; duration: 70 } }
                    SequentialAnimation on scale { id: cheer; running: false
                        NumberAnimation { to: 1.4; duration: 200 } NumberAnimation { to: 1; duration: 200 } NumberAnimation { to: 1.4; duration: 200 } NumberAnimation { to: 1; duration: 200 } }
                }
            }
            Label { text: win.message; font.pixelSize: 18; color: "#14532d"; wrapMode: Text.WordWrap; Layout.preferredWidth: 520 }
            // Levels.
            RowLayout {
                spacing: 4
                Repeater {
                    model: win.levels.length
                    delegate: Button {
                        required property int index
                        text: (index + 1) + (win.starsOf(index) ? "\n" + "⭐".repeat(win.starsOf(index)) : index > win.unlocked ? "\n🔒" : "\n ")
                        enabled: index <= win.unlocked && !win.running
                        highlighted: index === win.level
                        font.pixelSize: 11
                        Layout.preferredWidth: 48
                        onClicked: win.level = index
                    }
                }
            }
        }

        // The code.
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 10
            Label { text: "Blocks"; font.bold: true; font.pixelSize: 18; color: "#166534" }
            Flow {
                Layout.fillWidth: true
                spacing: 8
                Button { text: "⬆️ Forward"; font.pixelSize: 16; onClicked: win.addBlock("F") }
                Button { text: "↩️ Turn left"; font.pixelSize: 16; onClicked: win.addBlock("L") }
                Button { text: "↪️ Turn right"; font.pixelSize: 16; onClicked: win.addBlock("R") }
                Button { text: "🔁 Repeat"; font.pixelSize: 16; visible: win.lv.repeat; onClicked: win.addBlock("P") }
            }
            Label { text: "Your program (click a block to take it out" + (win.lv.repeat ? "; click 🔁's number to change it" : "") + ")"; color: "#15803d"; wrapMode: Text.WordWrap; Layout.fillWidth: true }
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                radius: 14
                color: "white"
                border.color: "#86efac"; border.width: 3
                Flow {
                    anchors.fill: parent
                    anchors.margins: 12
                    spacing: 8
                    Repeater {
                        model: win.program
                        delegate: Rectangle {
                            required property var modelData
                            required property int index
                            width: blockText.width + 24; height: 44; radius: 10
                            color: win.activeBlock === index ? "#facc15" : modelData.k === "P" ? "#c084fc" : modelData.k === "F" ? "#60a5fa" : "#fb923c"
                            Text {
                                id: blockText
                                anchors.centerIn: parent
                                text: modelData.k === "F" ? "⬆️ Forward" : modelData.k === "L" ? "↩️ Left" : modelData.k === "R" ? "↪️ Right" : "🔁 ×" + modelData.n
                                font.pixelSize: 16; font.bold: true; color: "white"
                            }
                            MouseArea {
                                anchors.fill: parent
                                enabled: !win.running
                                onClicked: {
                                    const p = win.program.slice();
                                    if (modelData.k === "P") { p[index] = { k: "P", n: modelData.n >= 9 ? 2 : modelData.n + 1 }; }
                                    else p.splice(index, 1);
                                    win.program = p;
                                }
                                onPressAndHold: { const p = win.program.slice(); p.splice(index, 1); win.program = p; }
                            }
                        }
                    }
                }
            }
            Label { text: win.program.length + " blocks" + "   (3 stars: " + win.lv.par + " or fewer)"; color: "#166534" }
            RowLayout {
                Button { text: "▶ Run"; font.pixelSize: 20; highlighted: true; enabled: !win.running && win.program.length > 0; onClicked: win.run() }
                Button { text: "🔄 Start again"; enabled: !win.running; onClicked: { win.reset(); win.message = ""; } }
                Button { text: "🗑️ Clear"; enabled: !win.running; onClicked: { win.program = []; win.reset(); } }
                Item { Layout.fillWidth: true }
                Button {
                    text: "Next level →"
                    visible: win.starsOf(win.level) > 0 && win.level < win.levels.length - 1
                    enabled: !win.running
                    onClicked: win.level++
                }
            }
        }
    }
}

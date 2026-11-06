// Math Quest (ollie-mathquest): 8 levels of 10 sums each, from adding up to
// 10 to sharing; stars for each level (10 right: 3 stars, 8: 2, 6: 1, and
// the next level opens). "Show me" draws the sum with dots for the small
// ones, or gives a tip. Type the answer or press the big number buttons.
// Says on the console: OLLIE-ACTION trophy ... (ollie-trophy).
// Argument: config=FILE (the stars won).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 900
    height: 700
    visible: true
    title: "Math Quest"
    color: "#ecfeff"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function trophy(w) { console.log("OLLIE-ACTION trophy " + w) }
    function rnd(a, b) { return a + Math.floor(Math.random() * (b - a + 1)) }

    readonly property var levels: [
        { name: "Adding up to 10", emoji: "➕" },
        { name: "Taking away from 10", emoji: "➖" },
        { name: "Adding up to 20", emoji: "➕" },
        { name: "Taking away from 20", emoji: "➖" },
        { name: "Up to 100", emoji: "💯" },
        { name: "Times 2, 5 and 10", emoji: "✖️" },
        { name: "All the times tables", emoji: "✖️" },
        { name: "Sharing", emoji: "➗" }
    ]
    // One sum for a level: { a, op, b, answer }.
    function make(l) {
        let a, b;
        switch (l) {
        case 0: a = rnd(0, 9); b = rnd(1, 10 - a); return { a: a, op: "+", b: b, answer: a + b };
        case 1: a = rnd(2, 10); b = rnd(1, a); return { a: a, op: "−", b: b, answer: a - b };
        case 2: a = rnd(5, 15); b = rnd(2, 20 - a); return { a: a, op: "+", b: b, answer: a + b };
        case 3: a = rnd(11, 20); b = rnd(2, a); return { a: a, op: "−", b: b, answer: a - b };
        case 4:
            if (Math.random() < 0.5) { a = rnd(10, 80); b = rnd(5, 100 - a); return { a: a, op: "+", b: b, answer: a + b }; }
            a = rnd(30, 100); b = rnd(5, a - 5); return { a: a, op: "−", b: b, answer: a - b };
        case 5: a = [2, 5, 10][rnd(0, 2)]; b = rnd(1, 10); return Math.random() < 0.5 ? { a: a, op: "×", b: b, answer: a * b } : { a: b, op: "×", b: a, answer: a * b };
        case 6: a = rnd(2, 10); b = rnd(2, 10); return { a: a, op: "×", b: b, answer: a * b };
        default: b = rnd(2, 10); const q = rnd(1, 10); return { a: b * q, op: "÷", b: b, answer: q };
        }
    }
    function words(s) {
        return s.a + " " + ({ "+": "plus", "−": "take away", "×": "times", "÷": "shared by" })[s.op] + " " + s.b;
    }

    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property string stars: "[]"
    }
    property var stars: { try { return JSON.parse(saved.stars); } catch (e) { return []; } }
    readonly property int unlocked: { let u = 0; while (u < levels.length - 1 && (stars[u] || 0) > 0) u++; return u; }

    property int level: -1
    property int n: 0
    property int right: 0
    property string state2: "ask"   // ask, right, wrong, done
    property bool hint: false
    property var sum: ({ a: 0, op: "+", b: 0, answer: 0 })
    property string typed: ""

    function start(l) {
        level = l; n = 0; right = 0; ask();
    }
    function ask() {
        sum = make(level); typed = ""; hint = false; state2 = "ask";
        keys.forceActiveFocus();
    }
    function check() {
        if (state2 !== "ask" || typed === "") return;
        if (parseInt(typed) === sum.answer) {
            right++; state2 = "right"; trophy("count math-right");
        } else {
            state2 = "wrong";
        }
    }
    function next() {
        n++;
        if (n >= 10) {
            state2 = "done";
            const got = right === 10 ? 3 : right >= 8 ? 2 : right >= 6 ? 1 : 0;
            const s = stars.slice(); s[level] = Math.max(s[level] || 0, got); saved.stars = JSON.stringify(s); stars = s;
            trophy("max math-level " + (unlocked + 1));
            return;
        }
        ask();
    }
    function press(k) {
        if (state2 !== "ask") { if (k === "ok") next(); return; }
        if (k === "ok") check();
        else if (k === "del") typed = typed.slice(0, -1);
        else if (typed.length < 4) typed += k;
    }

    // --- Choosing a level ---
    ColumnLayout {
        anchors.centerIn: parent
        spacing: 18
        visible: win.level < 0 || win.state2 === "done"
        Label { text: "🧮 Math Quest"; font.pixelSize: 44; font.bold: true; color: "#0e7490"; Layout.alignment: Qt.AlignHCenter }
        Label {
            visible: win.state2 === "done"
            text: "You got " + win.right + " of 10 right! " + "⭐".repeat(win.right === 10 ? 3 : win.right >= 8 ? 2 : win.right >= 6 ? 1 : 0)
            font.pixelSize: 24; color: "#155e75"; Layout.alignment: Qt.AlignHCenter
        }
        Label { text: "Pick a level:"; font.pixelSize: 18; color: "#155e75"; Layout.alignment: Qt.AlignHCenter }
        GridLayout {
            columns: 4
            columnSpacing: 12; rowSpacing: 12
            Layout.alignment: Qt.AlignHCenter
            Repeater {
                model: win.levels
                delegate: Button {
                    required property var modelData
                    required property int index
                    Layout.preferredWidth: 180; Layout.preferredHeight: 96
                    enabled: index <= win.unlocked
                    font.pixelSize: 15
                    text: modelData.emoji + " Level " + (index + 1) + "\n" + modelData.name + "\n"
                        + (index > win.unlocked ? "🔒" : (win.stars[index] ? "⭐".repeat(win.stars[index]) : "☆"))
                    onClicked: win.start(index)
                }
            }
        }
    }

    // --- The sums ---
    Item {
        id: keys
        anchors.fill: parent
        visible: win.level >= 0 && win.state2 !== "done"
        focus: true
        Keys.onPressed: (e) => {
            if (e.key >= Qt.Key_0 && e.key <= Qt.Key_9) win.press(String(e.key - Qt.Key_0));
            else if (e.key === Qt.Key_Backspace) win.press("del");
            else if (e.key === Qt.Key_Return || e.key === Qt.Key_Enter) win.press("ok");
        }
        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 26
            spacing: 14
            RowLayout {
                Label { text: "🧮 Level " + (win.level + 1) + ": " + (win.level >= 0 ? win.levels[win.level].name : ""); font.pixelSize: 22; font.bold: true; color: "#0e7490" }
                Item { Layout.fillWidth: true }
                Label { text: "Sum " + Math.min(10, win.n + 1) + " of 10    ✅ " + win.right; font.pixelSize: 20; color: "#155e75" }
                Button { text: "✖"; onClicked: win.level = -1 }
            }
            RowLayout {
                Layout.alignment: Qt.AlignHCenter
                spacing: 18
                Label { text: win.sum.a + " " + win.sum.op + " " + win.sum.b + " ="; font.pixelSize: 72; font.bold: true; color: "#164e63" }
                Rectangle {
                    width: 190; height: 104; radius: 18
                    color: win.state2 === "right" ? "#bbf7d0" : win.state2 === "wrong" ? "#fecaca" : "white"
                    border.color: "#06b6d4"; border.width: 4
                    Label {
                        anchors.centerIn: parent
                        text: win.state2 === "wrong" ? String(win.sum.answer) : (win.typed || "?")
                        font.pixelSize: 64; font.bold: true
                        color: win.typed || win.state2 === "wrong" ? "#164e63" : "#a5f3fc"
                    }
                }
            }
            Label {
                Layout.alignment: Qt.AlignHCenter
                font.pixelSize: 22; font.bold: true
                color: win.state2 === "right" ? "#15803d" : "#b91c1c"
                text: win.state2 === "right" ? "✅ Correct!" : win.state2 === "wrong" ? "Oops! " + win.sum.a + " " + win.sum.op + " " + win.sum.b + " = " + win.sum.answer : " "
            }
            // "Show me": dots for small sums, a tip for the rest.
            Row {
                visible: win.hint && win.level <= 1
                Layout.alignment: Qt.AlignHCenter
                spacing: 6
                Repeater {
                    model: win.sum.a
                    delegate: Label {
                        required property int index
                        text: win.sum.op === "−" && index >= win.sum.a - win.sum.b ? "❌" : "🔵"
                        font.pixelSize: 30
                    }
                }
                Label { visible: win.sum.op === "+"; text: "  ➕  "; font.pixelSize: 30 }
                Repeater {
                    model: win.sum.op === "+" ? win.sum.b : 0
                    delegate: Label { text: "🟠"; font.pixelSize: 30 }
                }
            }
            Label {
                visible: win.hint && win.level > 1
                Layout.alignment: Qt.AlignHCenter
                font.pixelSize: 18; color: "#155e75"; wrapMode: Text.WordWrap
                Layout.maximumWidth: 700; horizontalAlignment: Text.AlignHCenter
                text: win.sum.op === "+" ? "Tip: start at " + Math.max(win.sum.a, win.sum.b) + " and count on " + Math.min(win.sum.a, win.sum.b) + "."
                    : win.sum.op === "−" ? "Tip: start at " + win.sum.b + " and count up to " + win.sum.a + ". How many steps?"
                    : win.sum.op === "×" ? "Tip: " + win.sum.b + " groups of " + win.sum.a + ". Count up in " + win.sum.a + "s: " + Array.from({ length: Math.min(win.sum.b, 4) }, (_, i) => win.sum.a * (i + 1)).join(", ") + "..."
                    : "Tip: what times " + win.sum.b + " makes " + win.sum.a + "?"
            }
            Item { Layout.fillHeight: true }
            RowLayout {
                Layout.alignment: Qt.AlignHCenter
                spacing: 10
                Button { text: "👀 Show me"; font.pixelSize: 16; onClicked: win.hint = true }
            }
            GridLayout {
                Layout.alignment: Qt.AlignHCenter
                columns: 6
                columnSpacing: 8; rowSpacing: 8
                Repeater {
                    model: ["1", "2", "3", "4", "5", "del", "6", "7", "8", "9", "0", "ok"]
                    delegate: Button {
                        required property string modelData
                        Layout.preferredWidth: modelData === "ok" ? 150 : 80
                        Layout.preferredHeight: 64
                        font.pixelSize: 26
                        font.bold: true
                        highlighted: modelData === "ok"
                        text: modelData === "del" ? "⌫" : modelData === "ok" ? (win.state2 === "ask" ? "✔ Check" : win.n >= 9 ? "⭐ Stars" : "Next →") : modelData
                        onClicked: { win.press(modelData); keys.forceActiveFocus(); }
                    }
                }
            }
        }
    }
}

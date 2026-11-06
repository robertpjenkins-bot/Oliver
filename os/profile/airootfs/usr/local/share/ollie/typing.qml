// Typing Game (ollie-typing): words fall from the sky; type one and press
// Space or Enter to pop it before it lands. 3 hearts; every 10 words the
// next level is faster, with longer words. The best score is kept.
// Argument: config=FILE (where the best score is kept).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 900
    height: 680
    visible: true
    title: "Typing Game"
    color: "#CFE3EC"

    function arg(name) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(name + "=") === 0)
                return a[i].substring(name.length + 1);
        return "";
    }

    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property int best: 0
        property int bestLevel: 1
    }

    readonly property var words: [
        ["cat", "dog", "sun", "run", "fun", "hat", "box", "red", "big", "sky", "cup", "pig", "bus", "map", "egg", "win", "jam", "fox", "zip", "yes"],
        ["game", "jump", "frog", "cake", "star", "fish", "blue", "ship", "moon", "tree", "bike", "rock", "snow", "duck", "king", "lava", "gold", "door", "play", "cube"],
        ["apple", "pizza", "robot", "tiger", "candy", "ghost", "magic", "steam", "ninja", "dragon", "planet", "rocket", "turtle", "castle", "button", "laptop", "cookie", "banana", "jungle", "puzzle"],
        ["dinosaur", "keyboard", "computer", "treasure", "elephant", "minecraft", "spaceship", "chocolate", "volcano", "adventure", "pineapple", "butterfly", "controller", "lightning", "waterfall"]
    ]
    property var falling: []      // {id, text, x, y}
    property int nextId: 0
    property int score: 0
    property int popped: 0
    property int hearts: 3
    property int level: 1
    property string mode: "start"   // start, play, over
    property real startTime: 0
    property int typedChars: 0
    readonly property real speed: 0.6 + level * 0.35
    readonly property int spawnEvery: Math.max(700, 2300 - level * 180)

    function pick() {
        const set = words[Math.min(words.length - 1, Math.floor((level - 1) / 2))];
        return set[Math.floor(Math.random() * set.length)];
    }
    function start() {
        falling = []; score = 0; popped = 0; hearts = 3; level = 1; typedChars = 0;
        startTime = Date.now(); mode = "play"; input.text = ""; input.forceActiveFocus();
        spawn();
    }
    function spawn() {
        const t = pick();
        const w = t.length * 22 + 30;
        // Not on top of a word that has only just started falling.
        let x = 20;
        for (let tries = 0; tries < 12; tries++) {
            x = 20 + Math.random() * (sky.width - w - 40);
            if (!falling.some(f => f.y < 130 && Math.abs(f.x - x) < Math.max(w, f.text.length * 22 + 30) + 10)) break;
        }
        falling = falling.concat([{ id: nextId++, text: t, x: x, y: 60 }]);
    }
    function typed(t) {
        t = t.trim().toLowerCase();
        if (!t) return;
        const i = falling.findIndex(f => f.text === t);
        if (i >= 0) {
            const f = falling[i];
            pops.model.append({ px: f.x, py: f.y, txt: "+" + (t.length * 10 * level) });
            falling = falling.slice(0, i).concat(falling.slice(i + 1));
            score += t.length * 10 * level;
            typedChars += t.length + 1;
            popped++; trophy("count typing-word"); trophy("max typing-level " + level);
            if (popped % 10 === 0) { level++; }
        } else if (t === "ollie") {
            // A secret: Ollie's own name is worth 100.
            score += 100;
            pops.model.append({ px: sky.width / 2 - 100, py: sky.height / 2, txt: "Secret bonus +100" });
        } else {
            miss.restart();
        }
    }
    function trophy(w) { console.log("OLLIE-ACTION trophy " + w) }
    function gameOver() {
        mode = "over";
        if (score > saved.best) { saved.best = score; saved.bestLevel = level; newBest.visible = true; }
        else { newBest.visible = false; }
    }
    readonly property int wpm: mode === "start" || Date.now() - startTime < 1000 ? 0 : Math.round(typedChars / 5 / ((Date.now() - startTime) / 60000))

    Rectangle {
        id: sky
        anchors.fill: parent
        color: "#CFE3EC" // a flat sky, with grass along the bottom
        Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: parent.height * 0.12; color: "#6E9F4E" }

        // Falling words.
        Repeater {
            model: win.falling
            delegate: Rectangle {
                required property var modelData
                x: modelData.x
                y: modelData.y
                width: wordText.width + 30
                height: 44
                radius: 4
                readonly property bool matching: input.text.length > 0 && modelData.text.indexOf(input.text.trim().toLowerCase()) === 0
                color: matching ? "#F2D06B" : "white"
                border.color: "#22252B"
                border.width: 2
                Text {
                    id: wordText
                    anchors.centerIn: parent
                    text: modelData.text
                    font.pixelSize: 24
                    font.family: "Noto Sans Mono"
                    font.bold: true
                    color: "#22252B"
                }
            }
        }
        // "+points" where a word popped.
        Repeater {
            id: pops
            model: ListModel {}
            delegate: Text {
                required property real px
                required property real py
                required property string txt
                x: px; y: py
                text: txt
                color: "#B7791F"
                font.pixelSize: 28
                font.bold: true
                NumberAnimation on y { to: py - 80; duration: 700 }
                NumberAnimation on opacity { from: 1; to: 0; duration: 700 }
            }
        }
    }

    Timer {
        interval: 30
        running: win.mode === "play"
        repeat: true
        onTriggered: {
            const ground = sky.height - 110;
            let lost = 0;
            const next = [];
            for (const f of win.falling) {
                const g = { id: f.id, text: f.text, x: f.x, y: f.y + win.speed };
                if (g.y > ground) lost++; else next.push(g);
            }
            win.falling = next;
            if (lost > 0) {
                win.hearts -= lost;
                ouch.restart();
                if (win.hearts <= 0) win.gameOver();
            }
        }
    }
    Timer {
        interval: win.spawnEvery
        running: win.mode === "play"
        repeat: true
        onTriggered: win.spawn()
    }

    // Top bar.
    Rectangle { anchors.top: parent.top; anchors.left: parent.left; anchors.right: parent.right; height: 56; color: "#22252B" }
    RowLayout {
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 14
        Label { text: "Typing Game"; color: "white"; font.pixelSize: 22; font.bold: true }
        Item { Layout.fillWidth: true }
        Label { text: "Level " + win.level; color: "#C9CAD0"; font.pixelSize: 20; font.bold: true }
        Label { text: "   Score " + win.score; color: "white"; font.pixelSize: 20; font.bold: true }
        Label { text: "   " + "♥".repeat(Math.max(0, win.hearts)); color: "#E8665C"; font.pixelSize: 22 }
        Label { text: "♥".repeat(3 - Math.max(0, win.hearts)); color: "#55575E"; font.pixelSize: 22 }
    }

    // Where you type.
    Rectangle {
        id: bar
        anchors.bottom: parent.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        height: 90
        color: ouch.running ? "#C8473D" : "#22252B"
        RowLayout {
            anchors.fill: parent
            anchors.margins: 18
            TextField {
                id: input
                Layout.fillWidth: true
                font.pixelSize: 28
                placeholderText: win.mode === "play" ? "Type a word, then Space or Enter" : ""
                enabled: win.mode === "play"
                background: Rectangle { radius: 4; color: miss.running ? "#F3C9C4" : "white" }
                onTextChanged: if (text.endsWith(" ")) { win.typed(text); text = ""; }
                onAccepted: { win.typed(text); text = ""; }
            }
            Label { text: win.wpm + " words a minute"; color: "white"; font.pixelSize: 16; Layout.leftMargin: 10 }
        }
    }
    Timer { id: miss; interval: 300 }
    Timer { id: ouch; interval: 400 }

    // Start and game-over screens.
    Rectangle {
        anchors.centerIn: parent
        width: 520
        height: 360
        radius: 6
        color: "white"
        border.color: "#22252B"; border.width: 2
        visible: win.mode !== "play"
        ColumnLayout {
            anchors.centerIn: parent
            spacing: 14
            Label {
                Layout.alignment: Qt.AlignHCenter
                text: win.mode === "start" ? "Typing Game" : "Game over"
                font.pixelSize: 38
                font.bold: true
                color: "#22252B"
            }
            Label {
                Layout.alignment: Qt.AlignHCenter
                horizontalAlignment: Text.AlignHCenter
                text: win.mode === "start"
                    ? "Words fall from the sky.\nType each one and press Space before it lands."
                    : "Score: " + win.score + "   Level: " + win.level + "\nWords popped: " + win.popped + "   Speed: " + win.wpm + " words a minute"
                font.pixelSize: 18
                color: "#44474F"
            }
            Label { id: newBest; visible: false; Layout.alignment: Qt.AlignHCenter; text: "New best score"; font.pixelSize: 24; font.bold: true; color: "#B7791F" }
            Label {
                Layout.alignment: Qt.AlignHCenter
                text: "Best score: " + saved.best + " (level " + saved.bestLevel + ")"
                font.pixelSize: 16
                color: "#6B6F78"
            }
            Button {
                Layout.alignment: Qt.AlignHCenter
                text: win.mode === "start" ? "Play" : "Play again"
                font.pixelSize: 22
                highlighted: true
                onClicked: win.start()
            }
        }
    }
}

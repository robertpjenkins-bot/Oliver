// Ollie Pet (ollie-pet): a little cube pet that lives in its own window.
// Feed it, play with it, put it to bed; click it for a cuddle; play its
// mini-games (Catch apples, Hide and seek, Dance party). It gets hungry,
// bored and sleepy as time goes by, even while its window is shut, and it
// grows up: a baby, a kid, then a grown-up with a crown. It can walk along
// the bottom of the screen too (Pet on the desktop).
// Secret: call it Ollie and it is golden.
// Says on the console: OLLIE-ACTION say TEXT (Ollie's voice), desktop on|off.
// Arguments: config=FILE (where the pet is kept), mood=FILE (one line, for
// the Game Overlay), desktop=on|off.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 680
    height: 540
    visible: true
    title: saved.name ? saved.name + " (Ollie Pet)" : "Ollie Pet"
    color: "#7dd3fc"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }
    function trophy(w) { console.log("OLLIE-ACTION trophy " + w) }

    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property string name: ""
        property real full: 80
        property real fun: 80
        property real energy: 80
        property bool asleep: false
        property real last: 0
        property real born: 0
        property int stageSeen: 0
        property int bestCatch: 0
        property int bestDance: 0
    }

    // Time goes by (per hour): food -6, fun -5, energy -4 awake or +20 asleep.
    function passTime(hours) {
        hours = Math.min(hours, 72);
        saved.full = Math.max(0, saved.full - 6 * hours);
        saved.fun = Math.max(0, saved.fun - 5 * hours);
        saved.energy = saved.asleep ? Math.min(100, saved.energy + 20 * hours) : Math.max(0, saved.energy - 4 * hours);
        if (saved.asleep && saved.energy >= 100) saved.asleep = false;
    }
    property bool desktopOn: arg("desktop") === "on"
    Component.onCompleted: {
        const now = Date.now();
        if (saved.last > 0) passTime((now - saved.last) / 3600000);
        if (saved.born === 0) saved.born = now;
        saved.last = now;
        if (saved.name) { say(hello()); voice(hello().replace(/[^\w !?'.,]/g, "")); checkGrowth(); }
        writeMood();
    }
    Timer {
        interval: 10000
        running: true
        repeat: true
        onTriggered: {
            const now = Date.now();
            win.passTime((now - saved.last) / 3600000);
            saved.last = now;
            win.writeMood();
        }
    }

    readonly property string mood: saved.asleep ? "asleep" : saved.full < 25 ? "hungry" : saved.energy < 20 ? "sleepy" : saved.fun < 25 ? "bored" : "happy"
    property real nowMs: Date.now()
    Timer { interval: 60000; running: true; repeat: true; onTriggered: win.nowMs = Date.now() }
    readonly property int ageDays: saved.born > 0 ? Math.floor((nowMs - saved.born) / 86400000) : 0
    readonly property int stage: ageDays < 2 ? 0 : ageDays < 6 ? 1 : 2
    readonly property string stageName: ["Baby", "Kid", "Grown-up"][stage]
    onMoodChanged: writeMood()

    function checkGrowth() {
        trophy("max pet-stage " + stage);
        if (stage > saved.stageSeen) {
            saved.stageSeen = stage;
            say(stage === 1 ? "🎉 I grew up! I'm a kid now!" : "🎉 I'm all grown up! Look, a crown! 👑");
            voice(stage === 1 ? "Wow! I grew up! I'm a kid now!" : "I'm all grown up! I got a crown!");
            hearts(6);
        }
    }
    onStageChanged: checkGrowth()

    // The mood, for the Game Overlay and the desktop pet.
    function writeMood() {
        const f = arg("mood");
        if (!f || !saved.name) return;
        const words = { happy: "is happy", bored: "is bored", hungry: "is hungry", sleepy: "is sleepy", asleep: "is asleep" };
        const x = new XMLHttpRequest();
        x.open("PUT", "file://" + f);
        x.send(saved.name + " " + words[mood] + "\n");
    }

    function hello() {
        return mood === "hungry" ? "I'm hungry! 🍎" : mood === "sleepy" ? "I'm so sleepy..." : mood === "bored" ? "Let's play! ⚽" : mood === "asleep" ? "Zzz..." : "Hi! I missed you!";
    }
    property string saying: ""
    function say(t) { saying = t; bubbleTimer.restart(); }
    function voice(t) { if (!saved.asleep) act("say " + t); }
    Timer { id: bubbleTimer; interval: 3500; onTriggered: win.saying = "" }
    function hearts(n) { for (let i = 0; i < n; i++) heartModel.append({ hx: pet.x + 30 + Math.random() * 60, delay: i * 120 }); }

    function feed() {
        if (saved.asleep) { say("Shh... I'm sleeping"); return; }
        if (saved.full > 95) { say("I'm full! 😋"); return; }
        food.y = -60; food.x = pet.x + 40; food.visible = true; dropFood.start();
    }
    function play() {
        if (saved.asleep) { say("Shh... I'm sleeping"); return; }
        if (saved.energy < 10) { say("Too tired to play..."); return; }
        saved.fun = Math.min(100, saved.fun + 20);
        saved.energy = Math.max(0, saved.energy - 8);
        ball.x = pet.x + 120; ball.visible = true; bounce.start(); hop.start();
        say(["Wheee!", "Again! Again!", "This is fun!", "Catch!"][Math.floor(Math.random() * 4)]);
        hearts(2);
    }
    function bed() {
        saved.asleep = !saved.asleep;
        say(saved.asleep ? "Good night! 🌙" : saved.energy < 50 ? "Five more minutes..." : "Good morning! ☀️");
        if (!saved.asleep) voice("Good morning!");
    }
    function reward(fun, full, text) {
        trophy("count pet-game");
        saved.fun = Math.min(100, saved.fun + fun);
        saved.full = Math.min(100, saved.full + full);
        saved.energy = Math.max(0, saved.energy - 5);
        say(text); voice(text.replace(/[^\w !?'.,]/g, "")); hearts(5); hop.start();
    }

    // --- The garden ---
    Rectangle {
        id: scene
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: saved.asleep ? "#1e1b4b" : "#7dd3fc" }
            GradientStop { position: 0.68; color: saved.asleep ? "#312e81" : "#bae6fd" }
            GradientStop { position: 0.69; color: saved.asleep ? "#14532d" : "#4ade80" }
            GradientStop { position: 1; color: saved.asleep ? "#052e16" : "#16a34a" }
        }
        Label { x: 40; y: 70; text: saved.asleep ? "🌙" : "☀️"; font.pixelSize: 54 }
        Label { x: parent.width - 170; y: 80; text: saved.asleep ? "✨  ⭐" : "☁️"; font.pixelSize: 44; opacity: 0.9 }
        Label { id: tree; x: parent.width - 90; y: parent.height * 0.62; text: "🌳"; font.pixelSize: 70 }
        Label { x: 20; y: parent.height * 0.66; text: "🌷🌼"; font.pixelSize: 26 }
    }

    PetBody {
        id: pet
        x: 200
        y: scene.height * 0.69 - height + 8
        mood: win.mood
        stage: win.stage
        golden: saved.name.toLowerCase() === "ollie"
        visible: !games.playing || games.which === "catch" || games.which === "dance"
        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            enabled: !games.playing
            onClicked: {
                if (saved.asleep) { win.say("Zzz... 💤"); return; }
                saved.fun = Math.min(100, saved.fun + 3);
                win.hearts(3);
                hop.start();
                win.say(["I love you! 💖", "Hehe, that tickles!", "Yay!", "You're my best friend!", "Boop!"][Math.floor(Math.random() * 5)]);
            }
        }
    }

    // Walking about (not when asleep, sleepy or hungry, nor in a game).
    Timer {
        interval: 60
        running: (win.mood === "happy" || win.mood === "bored") && !games.playing
        repeat: true
        property int pause: 0
        onTriggered: {
            if (pause > 0) { pause--; pet.stepping = false; return; }
            pet.x += pet.dir * (win.mood === "happy" ? 2.2 : 1);
            pet.stepping = Math.floor(pet.x / 14) % 2 === 0;
            if (pet.x < 20) { pet.x = 20; pet.dir = 1; pause = 25; }
            if (pet.x > win.width - pet.width - 110) { pet.x = win.width - pet.width - 110; pet.dir = -1; pause = 25; }
            if (Math.random() < 0.006) { pet.dir = -pet.dir; pause = 30; }
            if (Math.random() < 0.004 && win.mood === "happy") hop.start();
        }
    }
    SequentialAnimation {
        id: hop
        NumberAnimation { target: pet; property: "hopY"; to: 50; duration: 220; easing.type: Easing.OutQuad }
        NumberAnimation { target: pet; property: "hopY"; to: 0; duration: 260; easing.type: Easing.InQuad }
    }

    Label { id: food; text: "🍎"; font.pixelSize: 44; visible: false }
    SequentialAnimation {
        id: dropFood
        NumberAnimation { target: food; property: "y"; to: pet.y + 20; duration: 700; easing.type: Easing.InQuad }
        ScriptAction { script: {
            food.visible = false;
            saved.full = Math.min(100, saved.full + 25); win.trophy("count pet-feed");
            win.hearts(2); hop.start();
            win.say(["Yum! 😋", "Delicious!", "Munch munch!", "Thank you!"][Math.floor(Math.random() * 4)]);
        } }
    }
    Label { id: ball; text: "⚽"; font.pixelSize: 40; visible: false; y: pet.y + 70 }
    SequentialAnimation {
        id: bounce
        NumberAnimation { target: ball; property: "y"; to: pet.y - 60; duration: 300; easing.type: Easing.OutQuad }
        NumberAnimation { target: ball; property: "y"; to: pet.y + 80; duration: 300; easing.type: Easing.InQuad }
        NumberAnimation { target: ball; property: "y"; to: pet.y + 10; duration: 250; easing.type: Easing.OutQuad }
        NumberAnimation { target: ball; property: "y"; to: pet.y + 80; duration: 250; easing.type: Easing.InQuad }
        ScriptAction { script: ball.visible = false }
    }

    Repeater {
        model: ListModel { id: heartModel }
        delegate: Label {
            required property real hx
            required property int delay
            x: hx; y: pet.y
            text: "💖"; font.pixelSize: 26; opacity: 0
            SequentialAnimation on y { PauseAnimation { duration: delay } NumberAnimation { to: pet.y - 120; duration: 900 } }
            SequentialAnimation on opacity { PauseAnimation { duration: delay } NumberAnimation { from: 1; to: 0; duration: 900 } }
        }
    }

    Rectangle {
        visible: win.saying !== ""
        x: Math.min(win.width - width - 10, Math.max(10, pet.x + pet.width - 20))
        y: pet.y - 70
        width: bubble.width + 26; height: 42; radius: 16
        color: "white"; border.color: "#0b4f4a"; border.width: 2
        Label { id: bubble; anchors.centerIn: parent; font.pixelSize: 17; color: "#0b4f4a"; text: win.saying }
    }

    // --- Mini-games ---
    Item {
        id: games
        anchors.fill: parent
        property bool playing: which !== ""
        property string which: ""
        property int score: 0
        property int time: 0
        focus: playing
        function stop() { which = ""; pet.x = 200; pet.y = Qt.binding(() => scene.height * 0.69 - pet.height + 8); }

        // Catch apples: move with ← → (or the mouse); 30 seconds.
        property var apples: []
        function startCatch() { which = "catch"; score = 0; time = 30; apples = []; forceActiveFocus(); win.say("Catch the apples! ← →"); }
        Timer {
            interval: 40; running: games.which === "catch"; repeat: true
            property int n: 0
            onTriggered: {
                n++;
                if (n % 22 === 0) games.apples = games.apples.concat([{ x: 30 + Math.random() * (win.width - 100), y: 70, gold: Math.random() < 0.1 }]);
                if (n % 25 === 0) { games.time--; if (games.time <= 0) {
                    games.stop();
                    if (games.score > saved.bestCatch) saved.bestCatch = games.score;
                    win.reward(Math.min(30, games.score * 2), Math.min(30, games.score * 2), "I caught " + games.score + " apples! 🍎");
                    return; } }
                const next = [];
                for (const a of games.apples) {
                    const b = { x: a.x, y: a.y + 5, gold: a.gold };
                    if (b.y > pet.y + 10 && b.y < pet.y + 90 && b.x > pet.x - 10 && b.x < pet.x + pet.width - 20) { games.score += b.gold ? 5 : 1; continue; }
                    if (b.y < win.height) next.push(b);
                }
                games.apples = next;
            }
        }
        Keys.onLeftPressed: if (which === "catch") { pet.x = Math.max(0, pet.x - 26); pet.dir = -1; pet.stepping = !pet.stepping }
            else if (which === "dance") dancePress("⬅️")
        Keys.onRightPressed: if (which === "catch") { pet.x = Math.min(win.width - pet.width, pet.x + 26); pet.dir = 1; pet.stepping = !pet.stepping }
            else if (which === "dance") dancePress("➡️")
        Keys.onUpPressed: if (which === "dance") dancePress("⬆️")
        Keys.onDownPressed: if (which === "dance") dancePress("⬇️")
        Keys.onEscapePressed: stop()
        MouseArea {
            anchors.fill: parent
            enabled: games.which === "catch"
            hoverEnabled: true
            onPositionChanged: m => { pet.dir = m.x > pet.x + 65 ? 1 : -1; pet.x = Math.max(0, Math.min(win.width - pet.width, m.x - 65)); }
        }
        Repeater {
            model: games.which === "catch" ? games.apples : []
            delegate: Label { required property var modelData; x: modelData.x; y: modelData.y; text: modelData.gold ? "🌟" : "🍎"; font.pixelSize: 40 }
        }

        // Hide and seek: the pet hides behind one of three things; find it in 5 rounds.
        property int hideAt: 0
        property int round: 0
        property bool found: false
        function startHide() { which = "hide"; score = 0; round = 0; nextHide(); }
        function nextHide() { round++; found = false; hideAt = Math.floor(Math.random() * 3); win.say("Round " + round + ": where am I hiding? 🙈"); }
        Row {
            visible: games.which === "hide"
            anchors.horizontalCenter: parent.horizontalCenter
            y: scene.height * 0.69 - 130
            spacing: 60
            Repeater {
                model: ["🌳", "📦", "🌼"]
                delegate: Item {
                    required property string modelData
                    required property int index
                    width: 140; height: 140
                    Label { anchors.centerIn: parent; text: modelData; font.pixelSize: 110; visible: !(games.found && games.hideAt === index) }
                    PetBody { anchors.centerIn: parent; visible: games.found && games.hideAt === index; mood: "happy"; stage: win.stage }
                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        enabled: !games.found
                        onClicked: {
                            games.found = true;
                            if (index === games.hideAt) { games.score++; win.say("You found me! 🎉"); win.voice("You found me!"); }
                            else win.say("Nope! I was behind the " + ["tree", "box", "flowers"][games.hideAt] + "!");
                            hideNext.start();
                        }
                    }
                }
            }
        }
        Timer { id: hideNext; interval: 1800; onTriggered: {
            if (games.round >= 5) { const s = games.score; games.stop(); win.reward(s * 6, 0, "You found me " + s + " times out of 5! 🙈"); }
            else games.nextHide(); } }

        // Dance party: watch the arrows, then press them in the same order.
        property var steps: []
        property int at: 0
        property bool showing: false
        property string shown: ""
        function startDance() { which = "dance"; score = 0; steps = []; forceActiveFocus(); danceMore(); }
        function danceMore() {
            steps = steps.concat([["⬅️", "⬆️", "➡️", "⬇️"][Math.floor(Math.random() * 4)]]);
            at = 0; showing = true; showStep.i = 0; showStep.start();
        }
        Timer {
            id: showStep; interval: 650; repeat: true
            property int i: 0
            onTriggered: {
                if (i < games.steps.length) { games.shown = games.steps[i]; hop.start(); i++; }
                else { stop(); games.shown = ""; games.showing = false; win.say("Your turn! 💃"); }
            }
        }
        function dancePress(k) {
            if (showing) return;
            if (k === steps[at]) {
                at++; hop.start(); shown = k;
                if (at === steps.length) { score = steps.length; win.say("Yes! " + score + " moves! 🕺"); danceNext.start(); }
            } else {
                const s = steps.length - 1;
                stop();
                if (s > saved.bestDance) saved.bestDance = s;
                win.reward(Math.min(30, s * 4), 0, "We danced " + s + " moves! 🎶");
            }
        }
        Timer { id: danceNext; interval: 900; onTriggered: games.danceMore() }
        Label {
            visible: games.which === "dance" && games.shown !== ""
            anchors.horizontalCenter: parent.horizontalCenter
            y: 110; text: games.shown; font.pixelSize: 90
        }
        Row {
            visible: games.which === "dance"
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.bottom: parent.bottom; anchors.bottomMargin: 70
            spacing: 10
            Repeater {
                model: ["⬅️", "⬆️", "⬇️", "➡️"]
                delegate: Button { required property string modelData; text: modelData; font.pixelSize: 26; onClicked: games.dancePress(modelData) }
            }
        }

        // The game's score and a way out.
        Rectangle {
            visible: games.playing
            anchors.top: parent.top; anchors.topMargin: 64
            anchors.horizontalCenter: parent.horizontalCenter
            width: gameInfo.width + 40; height: 40; radius: 14; color: "#ffffffdd"
            Label {
                id: gameInfo
                anchors.centerIn: parent
                font.pixelSize: 17; font.bold: true; color: "#0b4f4a"
                text: games.which === "catch" ? "🍎 " + games.score + "    ⏱️ " + games.time + "    (best " + saved.bestCatch + ")"
                    : games.which === "hide" ? "🙈 Round " + games.round + " of 5    Found: " + games.score
                    : "💃 Moves: " + games.steps.length + "    (best " + saved.bestDance + ")"
            }
        }
        Button {
            visible: games.playing
            anchors.right: parent.right; anchors.bottom: parent.bottom; anchors.margins: 14
            text: "✖ Stop"
            onClicked: games.stop()
        }
    }

    // --- Its meters and buttons ---
    Rectangle {
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 10
        height: 46
        radius: 14
        color: "#ffffffcc"
        RowLayout {
            anchors.fill: parent
            anchors.margins: 8
            spacing: 12
            Label { text: (saved.name || "Ollie Pet") + "  ·  " + win.stageName + (win.ageDays > 0 ? ", " + win.ageDays + (win.ageDays === 1 ? " day old" : " days old") : ""); font.bold: true; font.pixelSize: 16; color: "#0b4f4a" }
            Item { Layout.fillWidth: true }
            Repeater {
                model: [{ i: "🍎", v: saved.full }, { i: "⚽", v: saved.fun }, { i: "⚡", v: saved.energy }]
                delegate: RowLayout {
                    required property var modelData
                    spacing: 4
                    Label { text: modelData.i; font.pixelSize: 16 }
                    Rectangle {
                        width: 64; height: 12; radius: 6; color: "#e5e7eb"
                        Rectangle { width: parent.width * modelData.v / 100; height: parent.height; radius: 6
                            color: modelData.v < 25 ? "#ef4444" : modelData.v < 50 ? "#f59e0b" : "#22c55e" }
                    }
                }
            }
        }
    }
    RowLayout {
        visible: !games.playing
        anchors.bottom: parent.bottom
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottomMargin: 14
        spacing: 8
        Button { text: "🍎 Feed"; font.pixelSize: 16; onClicked: win.feed() }
        Button { text: "⚽ Play"; font.pixelSize: 16; onClicked: win.play() }
        Button { text: saved.asleep ? "☀️ Wake up" : "🌙 Bed"; font.pixelSize: 16; onClicked: win.bed() }
        Button {
            text: "🎮 Games"; font.pixelSize: 16
            onClicked: gameMenu.open()
            Menu {
                id: gameMenu
                MenuItem { text: "🍎 Catch apples"; onTriggered: saved.asleep ? win.say("Shh... I'm sleeping") : games.startCatch() }
                MenuItem { text: "🙈 Hide and seek"; onTriggered: saved.asleep ? win.say("Shh... I'm sleeping") : games.startHide() }
                MenuItem { text: "💃 Dance party"; onTriggered: saved.asleep ? win.say("Shh... I'm sleeping") : games.startDance() }
            }
        }
        Button {
            text: win.desktopOn ? "🖥️ Desktop: on" : "🖥️ Desktop: off"; font.pixelSize: 16
            ToolTip.visible: hovered
            ToolTip.text: "Let your pet walk along the bottom of the screen"
            onClicked: { win.desktopOn = !win.desktopOn; win.act("desktop " + (win.desktopOn ? "on" : "off")); }
        }
    }

    // First time: a name.
    Rectangle {
        anchors.fill: parent
        color: "#000000aa"
        visible: saved.name === ""
        MouseArea { anchors.fill: parent }
        Rectangle {
            anchors.centerIn: parent
            width: 420; height: 230; radius: 22; color: "white"
            ColumnLayout {
                anchors.centerIn: parent
                spacing: 12
                Label { text: "🥚 A new pet!"; font.pixelSize: 30; font.bold: true; color: "#0b4f4a"; Layout.alignment: Qt.AlignHCenter }
                Label { text: "What will you call it?"; font.pixelSize: 17; color: "#0f766e"; Layout.alignment: Qt.AlignHCenter }
                TextField { id: nameField; placeholderText: "Its name"; font.pixelSize: 20; Layout.preferredWidth: 260; Layout.alignment: Qt.AlignHCenter; maximumLength: 20
                    onAccepted: done.clicked() }
                Button { id: done; text: "Hello!"; font.pixelSize: 18; enabled: nameField.text.trim() !== ""; Layout.alignment: Qt.AlignHCenter; highlighted: true
                    onClicked: {
                        saved.name = nameField.text.trim(); saved.born = Date.now(); saved.stageSeen = 0;
                        if (saved.name.toLowerCase() === "ollie") {   // a secret
                            win.say("✨ A golden Ollie! You found a secret! ✨"); win.voice("Wow! I'm Ollie, the golden one! You found a secret!"); win.hearts(10);
                        } else { win.say("Hi! I'm " + saved.name + "! 💖"); win.voice("Hi! I'm " + saved.name + "!"); win.hearts(4); }
                        win.writeMood();
                    } }
            }
        }
    }
}

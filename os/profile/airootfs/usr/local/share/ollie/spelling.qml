// Spelling Bee (ollie-spelling): look, cover, spell. The word shows for a
// few seconds, then hides; spell it. 8 levels of 10 words each, from easy to
// tricky; stars for each level (10 right: 3 stars, 8: 2, 6: 1, and the next
// level opens). Look again, see it in a sentence (with a gap where the word
// goes), or peek at the first letter. Says on the console: OLLIE-ACTION
// trophy ... (ollie-trophy).
// Argument: config=FILE (the stars won).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 900
    height: 680
    visible: true
    title: "Spelling Bee"
    color: "#fefce8"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function trophy(w) { console.log("OLLIE-ACTION trophy " + w) }

    // [word, a sentence with it]
    readonly property var levels: [
        [["cat", "The cat is sleeping."], ["dog", "My dog likes to run."], ["sun", "The sun is hot."], ["red", "I have a red ball."], ["big", "That is a big tree."],
         ["hat", "She wears a hat."], ["cup", "Drink from the cup."], ["bed", "Go to bed."], ["pig", "The pig is pink."], ["fun", "Games are fun."]],
        [["fish", "The fish can swim."], ["jump", "Frogs jump high."], ["cake", "We ate birthday cake."], ["star", "Look at that star."], ["frog", "A green frog."],
         ["ship", "The ship sails on the sea."], ["moon", "The moon is bright."], ["tree", "Climb the tree."], ["blue", "The sky is blue."], ["game", "Let's play a game."]],
        [["apple", "An apple a day."], ["happy", "I feel happy."], ["robot", "The robot can talk."], ["pizza", "Pizza for dinner!"], ["tiger", "The tiger has stripes."],
         ["candy", "Too much candy."], ["water", "Drink some water."], ["green", "The grass is green."], ["sleep", "Time to sleep."], ["house", "This is my house."]],
        [["friend", "You are my best friend."], ["school", "We go to school."], ["animal", "A lion is an animal."], ["purple", "Purple grapes."], ["rocket", "The rocket flies to space."],
         ["garden", "Flowers in the garden."], ["monkey", "The monkey eats bananas."], ["dragon", "The dragon breathes fire."], ["winter", "Snow falls in winter."], ["castle", "The king lives in a castle."]],
        [["because", "I smiled because I won."], ["picture", "Draw a picture."], ["between", "Sit between us."], ["birthday", "Happy birthday!"], ["children", "The children play outside."],
         ["computer", "I play on the computer."], ["dinosaur", "A huge dinosaur."], ["elephant", "The elephant has a trunk."], ["treasure", "Pirates found the treasure."], ["mountain", "We climbed the mountain."]],
        [["beautiful", "A beautiful sunset."], ["different", "Every snowflake is different."], ["important", "This is important."], ["chocolate", "Hot chocolate."], ["adventure", "Let's go on an adventure."],
         ["favourite", "Blue is my favourite colour."], ["butterfly", "A butterfly landed on me."], ["vegetable", "A carrot is a vegetable."], ["remember", "Remember your coat."], ["question", "Can I ask a question?"]],
        [["necessary", "Sleep is necessary."], ["separate", "Keep them separate."], ["calendar", "Look at the calendar."], ["knowledge", "Books give us knowledge."], ["rhythm", "Clap to the rhythm."],
         ["science", "Science is fun."], ["surprise", "It's a surprise party!"], ["whisper", "Whisper a secret."], ["library", "Books from the library."], ["neighbour", "Our neighbour has a cat."]],
        [["extraordinary", "An extraordinary day."], ["encyclopedia", "Look it up in the encyclopedia."], ["temperature", "The temperature is cold."], ["mischievous", "A mischievous puppy."], ["conscience", "Listen to your conscience."],
         ["environment", "Take care of the environment."], ["imagination", "Use your imagination."], ["pronunciation", "Practise the pronunciation."], ["restaurant", "Dinner at a restaurant."], ["vacuum", "Vacuum the floor."]]
    ]
    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property string stars: "[]"
    }
    property var stars: { try { return JSON.parse(saved.stars); } catch (e) { return []; } }
    readonly property int unlocked: { let u = 0; while (u < levels.length - 1 && (stars[u] || 0) > 0) u++; return u; }

    property int level: -1          // -1: choosing a level
    property var order: []
    property int n: 0               // which word
    property int right: 0
    property string state2: "ask"   // ask, right, wrong, done
    property bool peeked: false
    property bool showing: false    // the word is on the screen now
    property bool clue: false       // the sentence is on the screen
    function look() { showing = true; lookTimer.interval = 1800 + 300 * word[0].length; lookTimer.restart(); }
    Timer { id: lookTimer; onTriggered: { win.showing = false; answer.forceActiveFocus(); } }
    // The sentence, with a gap where the word is.
    readonly property string gapped: word[1].replace(new RegExp(word[0], "i"), "_".repeat(word[0].length))
    readonly property var word: level >= 0 && n < order.length ? levels[level][order[n]] : ["", ""]

    function start(l) {
        level = l; n = 0; right = 0; peeked = false;
        const o = [...Array(10).keys()];
        for (let i = 9; i > 0; i--) { const j = Math.floor(Math.random() * (i + 1)); [o[i], o[j]] = [o[j], o[i]]; }
        order = o; state2 = "ask"; clue = false; Qt.callLater(() => { look(); answer.text = ""; answer.forceActiveFocus(); });
    }
    function check() {
        if (state2 !== "ask" || answer.text.trim() === "") return;
        if (answer.text.trim().toLowerCase() === word[0]) {
            right++; state2 = "right"; trophy("count spell-right");
        } else state2 = "wrong";
        showing = false; lookTimer.stop();
    }
    function next() {
        n++; peeked = false; clue = false; answer.text = "";
        if (n >= 10) {
            state2 = "done";
            const got = right === 10 ? 3 : right >= 8 ? 2 : right >= 6 ? 1 : 0;
            const s = stars.slice(); s[level] = Math.max(s[level] || 0, got); saved.stars = JSON.stringify(s); stars = s;
            if (got === 3) trophy("count spelling-3stars");
            trophy("max spelling-level " + (unlocked + 1));
            return;
        }
        state2 = "ask"; look(); answer.forceActiveFocus();
    }

    // --- Choosing a level ---
    ColumnLayout {
        anchors.centerIn: parent
        spacing: 18
        visible: win.level < 0 || win.state2 === "done"
        Label { text: "Spelling Bee"; font.pixelSize: 44; font.bold: true; color: "#a16207"; Layout.alignment: Qt.AlignHCenter }
        Label {
            visible: win.state2 === "done"
            text: "You spelled " + win.right + " of 10 right! " + "★".repeat(win.right === 10 ? 3 : win.right >= 8 ? 2 : win.right >= 6 ? 1 : 0)
            font.pixelSize: 24; color: "#854d0e"; Layout.alignment: Qt.AlignHCenter
        }
        Label { text: "Look at the word, then spell it. Pick a level:"; font.pixelSize: 18; color: "#854d0e"; Layout.alignment: Qt.AlignHCenter }
        GridLayout {
            columns: 4
            columnSpacing: 12; rowSpacing: 12
            Layout.alignment: Qt.AlignHCenter
            Repeater {
                model: win.levels.length
                delegate: Button {
                    required property int index
                    Layout.preferredWidth: 150; Layout.preferredHeight: 80
                    enabled: index <= win.unlocked
                    font.pixelSize: 17
                    text: "Level " + (index + 1) + "\n" + (index > win.unlocked ? "" : (win.stars[index] ? "★".repeat(win.stars[index]) : "☆"))
                    onClicked: win.start(index)
                }
            }
        }
    }

    // --- Spelling ---
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 30
        spacing: 18
        visible: win.level >= 0 && win.state2 !== "done"
        RowLayout {
            Label { text: "Level " + (win.level + 1); font.pixelSize: 24; font.bold: true; color: "#a16207" }
            Item { Layout.fillWidth: true }
            Label { text: "Word " + Math.min(10, win.n + 1) + " of 10" + win.right; font.pixelSize: 20; color: "#854d0e" }
            Button { text: "✖"; onClicked: win.level = -1 }
        }
        Item { Layout.fillHeight: true }
        // The word while it shows, then the sentence clue (or nothing).
        Label {
            Layout.alignment: Qt.AlignHCenter
            Layout.preferredHeight: 110
            verticalAlignment: Text.AlignVCenter
            text: win.showing ? win.word[0] : win.clue && win.state2 === "ask" ? win.gapped : win.state2 === "ask" ? "Now spell it." : ""
            font.pixelSize: win.showing ? 72 : 26; font.bold: win.showing
            font.letterSpacing: win.showing ? 4 : 0
            color: win.showing ? "#713f12" : "#854d0e"
        }
        RowLayout {
            Layout.alignment: Qt.AlignHCenter
            spacing: 10
            enabled: win.state2 === "ask" && !win.showing
            Button { text: "Look again"; font.pixelSize: 17; onClicked: win.look() }
            Button { text: "In a sentence"; font.pixelSize: 17; onClicked: win.clue = true }
            Button { text: "First letter"; font.pixelSize: 17; onClicked: win.peeked = true }
        }
        // A box for each letter.
        Row {
            Layout.alignment: Qt.AlignHCenter
            spacing: 6
            Repeater {
                model: win.word[0].length
                delegate: Rectangle {
                    required property int index
                    width: 46; height: 56; radius: 8
                    color: win.state2 === "right" ? "#bbf7d0" : win.state2 === "wrong" ? "#fecaca" : "white"
                    border.color: "#ca8a04"; border.width: 3
                    Text {
                        anchors.centerIn: parent
                        font.pixelSize: 30; font.bold: true; color: "#713f12"
                        text: win.state2 === "wrong" ? win.word[0][index].toUpperCase()
                            : index < answer.text.length ? answer.text[index].toUpperCase()
                            : (index === 0 && win.peeked) ? win.word[0][0].toUpperCase() : ""
                        opacity: index === 0 && win.peeked && answer.text.length === 0 && win.state2 === "ask" ? 0.4 : 1
                    }
                }
            }
        }
        TextField {
            id: answer
            Layout.alignment: Qt.AlignHCenter
            Layout.preferredWidth: 420
            font.pixelSize: 30
            horizontalAlignment: TextInput.AlignHCenter
            placeholderText: "Type the word here"
            enabled: win.state2 === "ask" && !win.showing
            onAccepted: win.check()
        }
        Label {
            Layout.alignment: Qt.AlignHCenter
            font.pixelSize: 22; font.bold: true
            color: win.state2 === "right" ? "#15803d" : "#b91c1c"
            text: win.state2 === "right" ? "Correct!" : win.state2 === "wrong" ? "Oops! It's spelled " + win.word[0].toUpperCase() : ""
        }
        Button {
            Layout.alignment: Qt.AlignHCenter
            text: win.state2 === "ask" ? "✔ Check" : win.n >= 9 ? "See my stars" : "Next word →"
            font.pixelSize: 20
            highlighted: true
            onClicked: win.state2 === "ask" ? win.check() : win.next()
        }
        Item { Layout.fillHeight: true }
    }
}

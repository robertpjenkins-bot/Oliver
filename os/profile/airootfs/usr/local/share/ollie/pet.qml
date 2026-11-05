// Ollie Pet (ollie-pet): a little cube pet that lives in its own window.
// Feed it, play with it, put it to bed; click it for a cuddle. It gets
// hungry, bored and sleepy as time goes by, even while its window is shut,
// and remembers its name and how it is doing.
// Argument: config=FILE (where the pet is kept).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 640
    height: 520
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

    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property string name: ""
        property real full: 80
        property real fun: 80
        property real energy: 80
        property bool asleep: false
        property real last: 0
        property int days: 0
        property real born: 0
    }

    // Time goes by (per hour): food -6, fun -5, energy -4 awake or +20 asleep.
    function passTime(hours) {
        hours = Math.min(hours, 72);
        saved.full = Math.max(0, saved.full - 6 * hours);
        saved.fun = Math.max(0, saved.fun - 5 * hours);
        saved.energy = saved.asleep ? Math.min(100, saved.energy + 20 * hours) : Math.max(0, saved.energy - 4 * hours);
        if (saved.asleep && saved.energy >= 100) saved.asleep = false;
    }
    Component.onCompleted: {
        const now = Date.now();
        if (saved.last > 0) passTime((now - saved.last) / 3600000);
        if (saved.born === 0) saved.born = now;
        saved.last = now;
        say(saved.name ? hello() : "");
    }
    Timer {
        interval: 10000
        running: true
        repeat: true
        onTriggered: {
            const now = Date.now();
            win.passTime((now - saved.last) / 3600000);
            saved.last = now;
        }
    }

    readonly property string mood: saved.asleep ? "asleep" : saved.full < 25 ? "hungry" : saved.energy < 20 ? "sleepy" : saved.fun < 25 ? "bored" : "happy"
    readonly property int ageDays: saved.born > 0 ? Math.floor((Date.now() - saved.born) / 86400000) : 0

    function hello() {
        return mood === "hungry" ? "I'm hungry! 🍎" : mood === "sleepy" ? "I'm so sleepy..." : mood === "bored" ? "Let's play! ⚽" : mood === "asleep" ? "Zzz..." : "Hi! I missed you!";
    }
    property string saying: ""
    function say(t) { saying = t; bubbleTimer.restart(); }
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
        Label { x: parent.width - 90; y: parent.height * 0.62; text: "🌳"; font.pixelSize: 70 }
        Label { x: 20; y: parent.height * 0.66; text: "🌷🌼"; font.pixelSize: 26 }
    }

    // --- The pet: the Ollie cube with a face ---
    Item {
        id: pet
        width: 130
        height: 130
        y: scene.height * 0.69 - height + 8 - hopY
        property real hopY: 0
        property int dir: 1
        x: 200
        property bool stepping: false

        Rectangle { // shadow
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height - 6 + pet.hopY
            width: 100 - pet.hopY / 2; height: 14; radius: 7
            color: "#00000030"
        }
        Rectangle { // feet
            x: 28; y: pet.height - 16; width: 26; height: 16; radius: 6; color: "#0f766e"
            transform: Translate { y: pet.stepping ? -4 : 0 }
        }
        Rectangle {
            x: pet.width - 54; y: pet.height - 16; width: 26; height: 16; radius: 6; color: "#0f766e"
            transform: Translate { y: pet.stepping ? 0 : -4 }
        }
        Rectangle { // body
            id: body
            x: 10; y: 6; width: 110; height: 108; radius: 22
            color: "#2dd4bf"
            border.color: "#0b4f4a"; border.width: 4
            Rectangle { // the cube's white top
                x: 4; y: 4; width: parent.width - 8; height: 30; radius: 18
                color: "white"
                Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 12; color: "white" }
            }
            Rectangle { // light side
                x: 4; y: 30; width: (parent.width - 8) / 2; height: parent.height - 34; radius: 14
                color: "#9eeee4"
            }
            // eyes
            Repeater {
                model: [30, 70]
                delegate: Item {
                    required property int modelData
                    x: modelData - 10; y: 44; width: 22; height: 24
                    Rectangle {
                        visible: win.mood !== "asleep" && !blink.closed
                        anchors.fill: parent; radius: 11; color: "white"; border.color: "#0b4f4a"; border.width: 2
                        Rectangle { x: 6 + pet.dir * 3; y: 7; width: 10; height: 12; radius: 5; color: "#111827" }
                    }
                    Rectangle {
                        visible: win.mood === "asleep" || blink.closed
                        y: 12; width: parent.width; height: 4; radius: 2; color: "#0b4f4a"
                    }
                }
            }
            // cheeks
            Rectangle { x: 14; y: 72; width: 14; height: 8; radius: 4; color: "#fda4af"; visible: win.mood === "happy" }
            Rectangle { x: 82; y: 72; width: 14; height: 8; radius: 4; color: "#fda4af"; visible: win.mood === "happy" }
            // mouth
            Canvas {
                id: mouth
                x: 38; y: 74; width: 34; height: 20
                property string m: win.mood
                onMChanged: requestPaint()
                onPaint: {
                    const c = getContext("2d");
                    c.reset();
                    c.strokeStyle = "#0b4f4a"; c.lineWidth = 4; c.lineCap = "round";
                    c.beginPath();
                    if (m === "happy") c.arc(17, 2, 12, 0.15 * Math.PI, 0.85 * Math.PI);
                    else if (m === "asleep") c.ellipse(12, 6, 8, 8);
                    else if (m === "hungry") { c.ellipse(9, 2, 14, 14); }
                    else c.arc(17, 20, 10, 1.2 * Math.PI, 1.8 * Math.PI);
                    c.stroke();
                }
            }
        }
        Label { // Zzz
            visible: win.mood === "asleep"
            x: pet.width - 20; y: -10
            text: "💤"; font.pixelSize: 34
            SequentialAnimation on y { running: win.mood === "asleep"; loops: Animation.Infinite
                NumberAnimation { to: -30; duration: 1200 } NumberAnimation { to: -10; duration: 1200 } }
        }
        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: {
                if (saved.asleep) { win.say("Zzz... 💤"); return; }
                saved.fun = Math.min(100, saved.fun + 3);
                win.hearts(3);
                hop.start();
                win.say(["I love you! 💖", "Hehe, that tickles!", "Yay!", "You're my best friend!", "Boop!"][Math.floor(Math.random() * 5)]);
            }
        }
    }
    Item {
        id: blink
        property bool closed: false
        Timer { interval: 3200; running: win.mood !== "asleep"; repeat: true; onTriggered: { blink.closed = true; openTimer.start(); } }
        Timer { id: openTimer; interval: 160; onTriggered: blink.closed = false }
    }

    // Walking about (not when asleep, sleepy or hungry: then it sits).
    Timer {
        interval: 60
        running: win.mood === "happy" || win.mood === "bored"
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

    // Food and ball.
    Label { id: food; text: "🍎"; font.pixelSize: 44; visible: false }
    SequentialAnimation {
        id: dropFood
        NumberAnimation { target: food; property: "y"; to: pet.y + 20; duration: 700; easing.type: Easing.InQuad }
        ScriptAction { script: {
            food.visible = false;
            saved.full = Math.min(100, saved.full + 25);
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

    // Hearts that float up.
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

    // Speech bubble.
    Rectangle {
        visible: win.saying !== ""
        x: Math.min(win.width - width - 10, Math.max(10, pet.x + pet.width - 20))
        y: pet.y - 60
        width: bubble.width + 26; height: 42; radius: 16
        color: "white"; border.color: "#0b4f4a"; border.width: 2
        Label { id: bubble; anchors.centerIn: parent; font.pixelSize: 17; color: "#0b4f4a"; text: win.saying }
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
            Label { text: (saved.name || "Ollie Pet") + (win.ageDays > 0 ? "  (" + win.ageDays + (win.ageDays === 1 ? " day old)" : " days old)") : ""); font.bold: true; font.pixelSize: 17; color: "#0b4f4a" }
            Item { Layout.fillWidth: true }
            Repeater {
                model: [{ i: "🍎", v: saved.full }, { i: "⚽", v: saved.fun }, { i: "⚡", v: saved.energy }]
                delegate: RowLayout {
                    required property var modelData
                    spacing: 4
                    Label { text: modelData.i; font.pixelSize: 16 }
                    Rectangle {
                        width: 70; height: 12; radius: 6; color: "#e5e7eb"
                        Rectangle { width: parent.width * modelData.v / 100; height: parent.height; radius: 6
                            color: modelData.v < 25 ? "#ef4444" : modelData.v < 50 ? "#f59e0b" : "#22c55e" }
                    }
                }
            }
        }
    }
    RowLayout {
        anchors.bottom: parent.bottom
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottomMargin: 14
        spacing: 12
        Button { text: "🍎 Feed"; font.pixelSize: 18; onClicked: win.feed() }
        Button { text: "⚽ Play"; font.pixelSize: 18; onClicked: win.play() }
        Button { text: saved.asleep ? "☀️ Wake up" : "🌙 Bed time"; font.pixelSize: 18; onClicked: win.bed() }
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
                    onClicked: { saved.name = nameField.text.trim(); saved.born = Date.now(); win.say("Hi! I'm " + saved.name + "! 💖"); win.hearts(4); } }
            }
        }
    }
}

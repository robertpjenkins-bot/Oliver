// Ollie Pet on the desktop (ollie-pet --desktop): the pet walks along the
// bottom of the screen, just above the taskbar, over the other windows
// (Plasma's layer shell). Clicks go through it to what is under it; the
// Ollie Pet window is where to feed and play. It reads how the pet is from
// its file every half minute (time passing the same way as there), and
// tells the Game Overlay its mood.
// Arguments: config=FILE (the pet), mood=FILE.
import QtQuick
import org.kde.layershell 1.0 as LayerShell

Window {
    id: win
    visible: true
    color: "transparent"
    title: "Ollie Pet on the desktop"
    flags: Qt.FramelessWindowHint | Qt.WindowTransparentForInput | Qt.WindowDoesNotAcceptFocus
    height: 190
    width: Screen.width
    LayerShell.Window.scope: "ollie-pet"
    LayerShell.Window.layer: LayerShell.Window.LayerTop
    LayerShell.Window.anchors: LayerShell.Window.AnchorBottom | LayerShell.Window.AnchorLeft | LayerShell.Window.AnchorRight
    LayerShell.Window.exclusionZone: 0

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    property var pet: ({ name: "", full: 80, fun: 80, energy: 80, asleep: false, last: 0, born: 0 })
    function load() {
        const x = new XMLHttpRequest();
        x.open("GET", "file://" + arg("config"), false);
        try { x.send(); } catch (e) { return; }
        const p = { name: "", full: 80, fun: 80, energy: 80, asleep: false, last: Date.now(), born: Date.now() };
        for (const line of x.responseText.split("\n")) {
            const m = line.match(/^(\w+)=(.*)$/);
            if (!m) continue;
            const v = m[2].replace(/^"(.*)"$/, "$1");
            if (m[1] === "name") p.name = v;
            else if (m[1] === "asleep") p.asleep = v === "true";
            else if (["full", "fun", "energy", "last", "born"].includes(m[1])) p[m[1]] = parseFloat(v);
        }
        // Time since the pet's window last looked after it.
        const h = Math.min(72, (Date.now() - p.last) / 3600000);
        p.full = Math.max(0, p.full - 6 * h);
        p.fun = Math.max(0, p.fun - 5 * h);
        p.energy = p.asleep ? Math.min(100, p.energy + 20 * h) : Math.max(0, p.energy - 4 * h);
        pet = p;
        const words = { happy: "is happy", bored: "is bored", hungry: "is hungry", sleepy: "is sleepy", asleep: "is asleep" };
        if (p.name && arg("mood")) {
            const w = new XMLHttpRequest();
            w.open("PUT", "file://" + arg("mood"));
            w.send(p.name + " " + words[mood] + "\n");
        }
    }
    Component.onCompleted: load()
    Timer { interval: 30000; running: true; repeat: true; onTriggered: win.load() }

    readonly property string mood: pet.asleep ? "asleep" : pet.full < 25 ? "hungry" : pet.energy < 20 ? "sleepy" : pet.fun < 25 ? "bored" : "happy"
    readonly property int ageDays: pet.born > 0 ? Math.floor((Date.now() - pet.born) / 86400000) : 0

    PetBody {
        id: body
        visible: win.pet.name !== ""
        x: 300
        y: win.height - height - 4
        mood: win.mood
        stage: win.ageDays < 2 ? 0 : win.ageDays < 6 ? 1 : 2
        scale: 0.8
    }
    Rectangle { // what it says now and then
        id: bubble
        visible: text.text !== ""
        x: Math.min(win.width - width - 10, body.x + body.width - 20)
        y: body.y - 30
        width: text.width + 22; height: 34; radius: 14
        color: "white"; border.color: "#0b4f4a"; border.width: 2
        Text { id: text; anchors.centerIn: parent; font.pixelSize: 15; color: "#0b4f4a"; text: "" }
    }
    Timer {
        interval: 45000; running: true; repeat: true
        onTriggered: {
            const lines = win.mood === "hungry" ? ["I'm hungry! 🍎", "Feed me please!"] : win.mood === "sleepy" ? ["So sleepy..."] :
                win.mood === "bored" ? ["Let's play! ⚽", "I'm bored..."] : win.mood === "asleep" ? ["Zzz..."] : ["Hi! 💖", "La la la 🎶", "I like it here!", "Wheee!"];
            text.text = lines[Math.floor(Math.random() * lines.length)];
            hush.restart();
        }
    }
    Timer { id: hush; interval: 4000; onTriggered: text.text = "" }

    Timer {
        interval: 60
        running: win.mood === "happy" || win.mood === "bored"
        repeat: true
        property int pause: 0
        onTriggered: {
            if (pause > 0) { pause--; body.stepping = false; return; }
            body.x += body.dir * (win.mood === "happy" ? 2 : 1);
            body.stepping = Math.floor(body.x / 14) % 2 === 0;
            if (body.x < 10) { body.x = 10; body.dir = 1; pause = 40; }
            if (body.x > win.width - body.width - 10) { body.x = win.width - body.width - 10; body.dir = -1; pause = 40; }
            if (Math.random() < 0.004) { body.dir = -body.dir; pause = 60; }
            if (Math.random() < 0.003 && win.mood === "happy") hop.start();
        }
    }
    SequentialAnimation {
        id: hop
        NumberAnimation { target: body; property: "hopY"; to: 40; duration: 220; easing.type: Easing.OutQuad }
        NumberAnimation { target: body; property: "hopY"; to: 0; duration: 260; easing.type: Easing.InQuad }
    }
}

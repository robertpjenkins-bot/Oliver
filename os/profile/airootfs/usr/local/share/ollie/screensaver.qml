// Ollie screensaver (ollie-screensaver): when nobody has used the computer
// for a while, the Ollie Pet plays on the screen: it walks, hops and chases
// its ball by day, and sleeps under the stars at night, with a big clock.
// Any key, a click, moving the mouse or a controller button ends it.
// Over everything (Plasma's layer shell, the overlay layer).
// Arguments: name= (the pet's), events=FILE (ollie-padd's controller buttons).
import QtQuick
import org.kde.layershell 1.0 as LayerShell

Window {
    id: win
    visible: true
    color: "black"
    title: "Ollie screensaver"
    LayerShell.Window.scope: "ollie-screensaver"
    LayerShell.Window.layer: LayerShell.Window.LayerOverlay
    LayerShell.Window.anchors: LayerShell.Window.AnchorTop | LayerShell.Window.AnchorBottom | LayerShell.Window.AnchorLeft | LayerShell.Window.AnchorRight
    LayerShell.Window.keyboardInteractivity: LayerShell.Window.KeyboardInteractivityExclusive
    LayerShell.Window.exclusionZone: -1
    width: Screen.width
    height: Screen.height

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    readonly property int hour: new Date().getHours()
    readonly property bool night: hour >= 20 || hour < 7
    property real startedAt: Date.now()
    function bye() { if (Date.now() - startedAt > 1500) Qt.quit(); }

    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: win.night ? "#020617" : "#38bdf8" }
            GradientStop { position: 0.7; color: win.night ? "#1e1b4b" : "#bae6fd" }
        }
    }
    Repeater { // stars at night, clouds by day
        model: win.night ? 70 : 6
        delegate: Text {
            required property int index
            x: (index * 467) % win.width
            y: win.night ? (index * 211) % (win.height * 0.6) : 60 + (index * 97) % (win.height * 0.35)
            text: win.night ? (index % 7 === 0 ? "⭐" : "✦") : "☁️"
            color: "white"
            font.pixelSize: win.night ? (index % 7 === 0 ? 26 : 10 + index % 6) : 70 + (index % 3) * 30
            opacity: win.night ? 0.5 + (index % 5) / 10 : 0.9
            SequentialAnimation on opacity { running: win.night; loops: Animation.Infinite
                PauseAnimation { duration: (index * 137) % 2000 }
                NumberAnimation { to: 0.2; duration: 1200 } NumberAnimation { to: 1; duration: 1200 } }
            NumberAnimation on x { running: !win.night; loops: Animation.Infinite; from: -300; to: win.width + 300; duration: 60000 + index * 9000 }
        }
    }
    Text { x: win.width - 260; y: 60; text: win.night ? "🌙" : "☀️"; font.pixelSize: 130 }
    // The hill.
    Rectangle {
        x: -win.width * 0.2; y: win.height * 0.72
        width: win.width * 1.4; height: win.height
        radius: win.width * 0.7
        color: win.night ? "#14532d" : "#4ade80"
    }
    Text { x: win.width * 0.08; y: win.height * 0.66; text: "🌳"; font.pixelSize: 120 }
    Text { x: win.width * 0.82; y: win.height * 0.7; text: "🌷🌼🌷"; font.pixelSize: 40 }

    // The clock.
    Column {
        anchors.horizontalCenter: parent.horizontalCenter
        y: win.height * 0.12
        spacing: 4
        Text { id: clock; anchors.horizontalCenter: parent.horizontalCenter; color: "white"; font.pixelSize: 120; font.bold: true; style: Text.Raised; styleColor: "#40000000" }
        Text { id: day; anchors.horizontalCenter: parent.horizontalCenter; color: "white"; font.pixelSize: 32; opacity: 0.9 }
        Timer { interval: 1000; running: true; repeat: true; triggeredOnStart: true
            onTriggered: { const d = new Date(); clock.text = d.toLocaleTimeString(Qt.locale(), Locale.ShortFormat); day.text = Qt.formatDate(d, "dddd d MMMM"); } }
    }

    // The ball.
    Text {
        id: ball
        visible: !win.night
        text: "⚽"; font.pixelSize: 54
        x: win.width * 0.6; y: win.height * 0.72 - 50
    }
    // The pet.
    PetBody {
        id: pet
        x: win.width * 0.3
        y: win.height * 0.72 - 120
        mood: win.night ? "asleep" : "happy"
        stage: 2
        golden: win.arg("name").toLowerCase() === "ollie"
        scale: 1.3
        hopY: 0
    }
    Text {
        visible: win.arg("name") !== "" && !win.night
        x: pet.x + 20; y: pet.y - 120
        text: win.arg("name"); color: "white"; font.pixelSize: 26; font.bold: true
        style: Text.Outline; styleColor: "#40000000"
    }
    // By day: walk to the ball, kick it, hop, and again.
    SequentialAnimation {
        running: !win.night
        loops: Animation.Infinite
        ScriptAction { script: { pet.dir = ball.x > pet.x ? 1 : -1; } }
        ParallelAnimation {
            NumberAnimation { target: pet; property: "x"; to: ball.x - (pet.dir > 0 ? 120 : -60); duration: 2600; easing.type: Easing.InOutQuad }
            SequentialAnimation { loops: 6
                PropertyAction { target: pet; property: "stepping"; value: true } PauseAnimation { duration: 220 }
                PropertyAction { target: pet; property: "stepping"; value: false } PauseAnimation { duration: 220 } }
        }
        ParallelAnimation { // the kick
            NumberAnimation { target: ball; property: "x"; to: Math.random() < 0.5 ? win.width * (0.15 + Math.random() * 0.25) : win.width * (0.6 + Math.random() * 0.25); duration: 1400; easing.type: Easing.OutQuad }
            SequentialAnimation {
                NumberAnimation { target: ball; property: "y"; to: win.height * 0.72 - 260; duration: 600; easing.type: Easing.OutQuad }
                NumberAnimation { target: ball; property: "y"; to: win.height * 0.72 - 50; duration: 800; easing.type: Easing.OutBounce }
            }
            SequentialAnimation { // a happy hop
                NumberAnimation { target: pet; property: "hopY"; to: 60; duration: 250; easing.type: Easing.OutQuad }
                NumberAnimation { target: pet; property: "hopY"; to: 0; duration: 300; easing.type: Easing.InQuad }
            }
        }
        PauseAnimation { duration: 900 }
    }
    Text {
        visible: win.night
        x: pet.x + 170; y: pet.y - 30
        text: "Zzz..."; color: "#c7d2fe"; font.pixelSize: 36
        SequentialAnimation on opacity { loops: Animation.Infinite; NumberAnimation { to: 0.2; duration: 1500 } NumberAnimation { to: 1; duration: 1500 } }
    }

    // Anything ends it.
    Item {
        anchors.fill: parent
        focus: true
        Keys.onPressed: win.bye()
    }
    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        property point first: Qt.point(-1, -1)
        onPositionChanged: (m) => {
            if (first.x < 0) { first = Qt.point(m.x, m.y); return; }
            if (Math.abs(m.x - first.x) + Math.abs(m.y - first.y) > 12) win.bye();
        }
        onPressed: win.bye()
        onWheel: win.bye()
    }
    property int lastSeq: -1
    Timer {
        interval: 200; running: true; repeat: true
        onTriggered: {
            const x = new XMLHttpRequest();
            x.open("GET", "file://" + win.arg("events"), false);
            let e;
            try { x.send(); e = JSON.parse(x.responseText).events; } catch (z) { return; }
            const last = e.length ? e[e.length - 1][0] : 0;
            if (win.lastSeq < 0) win.lastSeq = last; else if (last > win.lastSeq) win.bye();
        }
    }
}

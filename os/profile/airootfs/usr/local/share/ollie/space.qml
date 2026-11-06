// Space Explorer (ollie-space): a tour of the solar system: the Sun, the
// eight planets, the Moon and Pluto, each drawn big with its facts (how far
// from the Sun, how long its day and year are, its moons, how hot or cold,
// and something amazing). Click one in the strip, or fly to the next.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1120
    height: 740
    visible: true
    title: "Space Explorer"
    color: "#0B0E14" // space, with a mission screen's amber writing
    readonly property color amber: "#F2A33A"

    function trophy(w) { console.log("OLLIE-ACTION trophy " + w) }
    onAtChanged: trophy("add space-body " + bodies[at].name)

    readonly property var bodies: [
        { name: "The Sun", kind: "A star", size: 1.0, c1: "#fef08a", c2: "#f97316", glow: true,
          facts: [["☀️", "What it is", "The star in the middle of our solar system"], ["📏", "How big", "109 Earths would fit across it"],
                  ["🌡️", "How hot", "About 5,500 °C on the outside"], ["💡", "Light", "Its light takes about 8 minutes to reach Earth"]],
          wow: "More than a million Earths could fit inside the Sun!" },
        { name: "Mercury", kind: "Planet 1", size: 0.38, c1: "#d6d3d1", c2: "#78716c", craters: true,
          facts: [["🛣️", "From the Sun", "58 million km (the closest)"], ["🌅", "One day", "176 Earth days, sunrise to sunrise"],
                  ["🔄", "One year", "88 Earth days"], ["🌙", "Moons", "None"], ["🌡️", "Temperature", "From −180 °C at night to 430 °C in the day"]],
          wow: "Mercury is the smallest planet: only a bit bigger than our Moon." },
        { name: "Venus", kind: "Planet 2", size: 0.95, c1: "#fef3c7", c2: "#d97706",
          facts: [["🛣️", "From the Sun", "108 million km"], ["🌅", "One day", "117 Earth days, sunrise to sunrise"],
                  ["🔄", "One year", "225 Earth days"], ["🌙", "Moons", "None"], ["🌡️", "Temperature", "About 465 °C: the hottest planet"]],
          wow: "Venus spins backwards, so there the Sun rises in the west!" },
        { name: "Earth", kind: "Planet 3: home!", size: 1.0, c1: "#60a5fa", c2: "#1e3a8a", land: true,
          facts: [["🛣️", "From the Sun", "150 million km"], ["🌅", "One day", "24 hours"], ["🔄", "One year", "365 days"],
                  ["🌙", "Moons", "1: the Moon"], ["🌊", "Water", "Water covers about 71% of it"]],
          wow: "Earth is the only planet we know of with life on it. That's you!" },
        { name: "The Moon", kind: "Earth's moon", size: 0.27, c1: "#f5f5f4", c2: "#a8a29e", craters: true,
          facts: [["🛣️", "From Earth", "384,400 km"], ["🔄", "Around Earth", "About 27 days"],
                  ["👨‍🚀", "Visitors", "12 astronauts have walked on it"], ["🌡️", "Temperature", "From −170 °C to 120 °C"]],
          wow: "The Moon always shows Earth the same side." },
        { name: "Mars", kind: "Planet 4", size: 0.53, c1: "#fb923c", c2: "#9a3412", craters: true,
          facts: [["🛣️", "From the Sun", "228 million km"], ["🌅", "One day", "24 hours and 37 minutes"],
                  ["🔄", "One year", "687 Earth days"], ["🌙", "Moons", "2: Phobos and Deimos"], ["🌡️", "Temperature", "About −60 °C"]],
          wow: "Mars has the tallest volcano we know of: Olympus Mons, almost 3 times as tall as Mount Everest!" },
        { name: "Jupiter", kind: "Planet 5", size: 11.2, c1: "#fde68a", c2: "#b45309", bands: true, spot: true,
          facts: [["🛣️", "From the Sun", "778 million km"], ["🌅", "One day", "About 10 hours (the shortest)"],
                  ["🔄", "One year", "About 12 Earth years"], ["🌙", "Moons", "More than 90"], ["📏", "How big", "The biggest planet: 11 Earths across"]],
          wow: "Jupiter's Great Red Spot is a storm bigger than the whole Earth!" },
        { name: "Saturn", kind: "Planet 6", size: 9.4, c1: "#fef9c3", c2: "#ca8a04", bands: true, rings: true,
          facts: [["🛣️", "From the Sun", "1,430 million km"], ["🌅", "One day", "About 10 and a half hours"],
                  ["🔄", "One year", "About 29 Earth years"], ["🌙", "Moons", "More than 140"], ["💍", "Rings", "Made of ice and rock"]],
          wow: "Saturn is so light for its size that it would float in a giant bath!" },
        { name: "Uranus", kind: "Planet 7", size: 4.0, c1: "#a5f3fc", c2: "#0e7490", rings: true, thinRings: true,
          facts: [["🛣️", "From the Sun", "2,870 million km"], ["🌅", "One day", "About 17 hours"],
                  ["🔄", "One year", "84 Earth years"], ["🌙", "Moons", "28"], ["🌡️", "Temperature", "About −195 °C"]],
          wow: "Uranus spins lying on its side, like a rolling ball!" },
        { name: "Neptune", kind: "Planet 8", size: 3.9, c1: "#93c5fd", c2: "#1e40af", bands: true,
          facts: [["🛣️", "From the Sun", "4,500 million km"], ["🌅", "One day", "About 16 hours"],
                  ["🔄", "One year", "165 Earth years"], ["🌙", "Moons", "16"], ["💨", "Wind", "Up to 2,000 km an hour"]],
          wow: "Neptune has the fastest winds in the solar system!" },
        { name: "Pluto", kind: "A dwarf planet", size: 0.19, c1: "#fde4cf", c2: "#a16207", heart: true,
          facts: [["🛣️", "From the Sun", "About 5,900 million km"], ["🔄", "One year", "248 Earth years"],
                  ["🌙", "Moons", "5"], ["🌡️", "Temperature", "About −230 °C"]],
          wow: "Pluto has a big heart-shaped patch of ice on it!" }
    ]
    property int at: 3
    readonly property var b: bodies[at]
    // Stars.
    Repeater {
        model: 160
        delegate: Rectangle {
            required property int index
            x: (index * 7919) % win.width
            y: (index * 104729) % win.height
            width: index % 9 === 0 ? 3 : 2; height: width; radius: width / 2
            color: "white"; opacity: 0.25 + (index % 5) * 0.15
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 12

        // The strip.
        RowLayout {
            Label { text: "SPACE EXPLORER"; color: win.amber; font.family: "Noto Sans Mono"; font.pixelSize: 24; font.bold: true; font.letterSpacing: 2 }
            Item { Layout.fillWidth: true }
            Repeater {
                model: win.bodies
                delegate: Rectangle {
                    required property var modelData
                    required property int index
                    width: 58; height: 58; radius: 2
                    color: "transparent"
                    border.color: index === win.at ? win.amber : strip.containsMouse ? "#3A4150" : "transparent"; border.width: 2
                    Rectangle {
                        anchors.centerIn: parent
                        width: 14 + Math.min(30, Math.sqrt(modelData.size) * 12); height: width; radius: width / 2
                        gradient: Gradient { GradientStop { position: 0; color: modelData.c1 } GradientStop { position: 1; color: modelData.c2 } }
                    }
                    MouseArea { id: strip; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: win.at = index }
                    ToolTip.visible: strip.containsMouse; ToolTip.text: modelData.name
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 24

            // The planet, big.
            Item {
                Layout.preferredWidth: 520
                Layout.fillHeight: true
                Canvas {
                    id: planet
                    anchors.centerIn: parent
                    width: 500; height: 500
                    property int which: win.at
                    onWhichChanged: requestPaint()
                    rotation: win.b.name === "Uranus" ? 80 : 0
                    onPaint: {
                        const c = getContext("2d"), p = win.b, cx = width / 2, cy = height / 2;
                        const r = p.rings && !p.thinRings ? 150 : 200;
                        c.reset();
                        if (p.glow) { // the Sun's glow
                            const g = c.createRadialGradient(cx, cy, r * 0.8, cx, cy, r * 1.25);
                            g.addColorStop(0, "rgba(253,186,116,0.7)"); g.addColorStop(1, "rgba(253,186,116,0)");
                            c.fillStyle = g; c.fillRect(0, 0, width, height);
                        }
                        if (p.rings) { // the back half of the rings
                            c.strokeStyle = p.thinRings ? "rgba(165,243,252,0.6)" : "rgba(253,230,138,0.75)";
                            c.lineWidth = p.thinRings ? 4 : 26;
                            c.beginPath(); c.ellipse(cx - r * 1.65, cy - r * 0.38, r * 3.3, r * 0.76); c.stroke();
                        }
                        c.save();
                        c.beginPath(); c.arc(cx, cy, r, 0, 2 * Math.PI); c.clip();
                        const g = c.createRadialGradient(cx - r * 0.35, cy - r * 0.35, r * 0.1, cx, cy, r * 1.05);
                        g.addColorStop(0, p.c1); g.addColorStop(1, p.c2);
                        c.fillStyle = g; c.fillRect(cx - r, cy - r, 2 * r, 2 * r);
                        if (p.bands) {
                            for (let i = -5; i <= 5; i++) {
                                c.fillStyle = i % 2 ? "rgba(120,53,15,0.18)" : "rgba(255,255,255,0.12)";
                                c.fillRect(cx - r, cy + i * r / 5.5 - r / 22, 2 * r, r / 11 + (i % 3) * 4);
                            }
                        }
                        if (p.spot) { c.fillStyle = "rgba(185,28,28,0.75)"; c.beginPath(); c.ellipse(cx + r * 0.15, cy + r * 0.28, r * 0.38, r * 0.2); c.fill(); }
                        if (p.craters) {
                            for (let i = 0; i < 14; i++) {
                                const a = i * 2.4, d = r * (0.15 + (i * 37 % 70) / 100), s = r * (0.05 + (i % 4) * 0.03);
                                c.fillStyle = "rgba(0,0,0,0.15)"; c.beginPath(); c.arc(cx + Math.cos(a) * d, cy + Math.sin(a) * d, s, 0, 2 * Math.PI); c.fill();
                            }
                        }
                        if (p.land) { // Earth's land and clouds
                            c.fillStyle = "#16a34a";
                            [[-0.35, -0.2, 0.32, 0.4], [0.25, 0.1, 0.28, 0.45], [-0.1, 0.5, 0.25, 0.15], [0.35, -0.45, 0.2, 0.15]].forEach(l => {
                                c.beginPath(); c.ellipse(cx + l[0] * r - l[2] * r / 2, cy + l[1] * r - l[3] * r / 2, l[2] * r, l[3] * r); c.fill(); });
                            c.fillStyle = "rgba(255,255,255,0.55)";
                            [[-0.5, -0.55, 0.5, 0.08], [0.0, 0.3, 0.6, 0.07], [0.2, -0.15, 0.4, 0.06]].forEach(l => {
                                c.beginPath(); c.ellipse(cx + l[0] * r, cy + l[1] * r, l[2] * r, l[3] * r); c.fill(); });
                        }
                        if (p.heart) { // Pluto's heart
                            c.fillStyle = "rgba(255,255,255,0.8)";
                            const hx = cx + r * 0.15, hy = cy + r * 0.1, s = r * 0.5;
                            c.beginPath(); c.moveTo(hx, hy + s * 0.6);
                            c.bezierCurveTo(hx - s, hy, hx - s * 0.5, hy - s * 0.7, hx, hy - s * 0.25);
                            c.bezierCurveTo(hx + s * 0.5, hy - s * 0.7, hx + s, hy, hx, hy + s * 0.6); c.fill();
                        }
                        // Shade the night side.
                        if (!p.glow) {
                            const sh = c.createLinearGradient(cx - r, cy, cx + r, cy);
                            sh.addColorStop(0.55, "rgba(0,0,0,0)"); sh.addColorStop(1, "rgba(0,0,0,0.55)");
                            c.fillStyle = sh; c.fillRect(cx - r, cy - r, 2 * r, 2 * r);
                        }
                        c.restore();
                        if (p.rings) { // the front half of the rings
                            c.strokeStyle = p.thinRings ? "rgba(165,243,252,0.8)" : "rgba(254,243,199,0.9)";
                            c.lineWidth = p.thinRings ? 4 : 26;
                            c.beginPath(); c.ellipse(cx - r * 1.65, cy - r * 0.38, r * 3.3, r * 0.76);
                            c.save(); c.beginPath(); c.rect(0, cy, width, height); c.clip();
                            c.beginPath(); c.ellipse(cx - r * 1.65, cy - r * 0.38, r * 3.3, r * 0.76); c.stroke(); c.restore();
                        }
                    }
                    RotationAnimation on rotation { running: false }
                }
            }

            // Its facts.
            ColumnLayout {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 10
                Label { text: win.b.name; color: "white"; font.pixelSize: 44; font.bold: true }
                Label { text: win.b.kind.toUpperCase(); color: win.amber; font.family: "Noto Sans Mono"; font.pixelSize: 16; font.letterSpacing: 1 }
                Repeater {
                    model: win.b.facts
                    delegate: Rectangle {
                        required property var modelData
                        Layout.fillWidth: true
                        implicitHeight: 54
                        radius: 2
                        color: "transparent"
                        border.color: "#2A3140"
                        RowLayout {
                            anchors.fill: parent
                            anchors.margins: 10
                            spacing: 12
                            Label { text: modelData[1].toUpperCase(); color: "#8C93A3"; font.family: "Noto Sans Mono"; font.pixelSize: 13; Layout.preferredWidth: 130 }
                            Label { text: modelData[2]; color: "white"; font.pixelSize: 16; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                        }
                    }
                }
                Rectangle {
                    Layout.fillWidth: true
                    implicitHeight: wowText.implicitHeight + 24
                    radius: 2
                    color: "transparent"
                    border.color: win.amber; border.width: 1
                    Label { id: wowText; anchors.fill: parent; anchors.margins: 12; text: win.b.wow; color: "white"; font.pixelSize: 17; wrapMode: Text.WordWrap }
                }
                Item { Layout.fillHeight: true }
                RowLayout {
                    Button { text: "← Back"; enabled: win.at > 0; onClicked: win.at-- }
                    Item { Layout.fillWidth: true }
                    Button { text: "Fly to the next one"; highlighted: true; enabled: win.at < win.bodies.length - 1; onClicked: win.at++ }
                }
            }
        }
    }
}

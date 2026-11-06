// World Explorer (ollie-world): a spinning globe. Drag it to turn it; click
// a country to see its flag, continent, capital, how many people live
// there and (for many) a fun fact.
// "Find it!" is a game: find the country named, as fast as you can.
// The map is Natural Earth's (public domain), in world.json
// (os/art/make-world.py). Says on the console: OLLIE-ACTION trophy ...
// Argument: config=FILE (the best Find it! score).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: win
    width: 1120
    height: 740
    visible: true
    title: "World Explorer"
    color: "#020617"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function trophy(w) { console.log("OLLIE-ACTION trophy " + w) }
    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property int best: 0
    }

    property var countries: []
    Component.onCompleted: {
        const x = new XMLHttpRequest();
        x.open("GET", "file:///usr/local/share/ollie/world.json", false);
        try { x.send(); countries = JSON.parse(x.responseText); } catch (e) { countries = []; }
        globe.requestPaint();
    }
    property real lon0: 10      // the globe's middle, in degrees
    property real lat0: 20
    property int picked: -1
    property real idleUntil: 0
    readonly property var colours: ({ "Africa": "#f59e0b", "Europe": "#a78bfa", "Asia": "#f87171", "North America": "#34d399",
        "South America": "#22d3ee", "Oceania": "#f472b6", "Antarctica": "#e2e8f0" })

    function flag(code) {
        if (!code || code.length !== 2) return "🏳️";
        return String.fromCodePoint(0x1F1E6 + code.charCodeAt(0) - 65, 0x1F1E6 + code.charCodeAt(1) - 65);
    }
    function people(p) {
        if (p >= 1e9) return "about " + (p / 1e9).toFixed(1) + " billion";
        if (p >= 1e6) return "about " + Math.round(p / 1e6) + " million";
        if (p >= 1e3) return "about " + Math.round(p / 1e3) + " thousand";
        return p > 0 ? "about " + p : "very few";
    }

    // Drawing the globe (orthographic: as seen from space).
    readonly property real rad: Math.PI / 180
    function project(lon, lat, R) {
        const l = (lon - lon0) * rad, p = lat * rad, p0 = lat0 * rad;
        const cosc = Math.sin(p0) * Math.sin(p) + Math.cos(p0) * Math.cos(p) * Math.cos(l);
        let x = R * Math.cos(p) * Math.sin(l), y = R * (Math.cos(p0) * Math.sin(p) - Math.sin(p0) * Math.cos(p) * Math.cos(l));
        if (cosc < 0) { const k = R / (Math.hypot(x, y) || 1); x *= k; y *= k; }  // behind: on the edge
        return [x, -y, cosc >= 0];
    }
    function unproject(sx, sy, R) {
        const x = sx / R, y = -sy / R, rho = Math.hypot(x, y);
        if (rho > 1) return null;
        const c = Math.asin(rho), p0 = lat0 * rad;
        if (rho === 0) return [lon0, lat0];
        const lat = Math.asin(Math.cos(c) * Math.sin(p0) + y * Math.sin(c) * Math.cos(p0) / rho);
        const lon = lon0 * rad + Math.atan2(x * Math.sin(c), rho * Math.cos(c) * Math.cos(p0) - y * Math.sin(c) * Math.sin(p0));
        let d = lon / rad;
        while (d > 180) d -= 360;
        while (d < -180) d += 360;
        return [d, lat / rad];
    }
    function inside(ring, x, y) {
        let hit = false;
        for (let i = 0, j = ring.length - 2; i < ring.length; j = i, i += 2) {
            const xi = ring[i], yi = ring[i + 1], xj = ring[j], yj = ring[j + 1];
            if ((yi > y) !== (yj > y) && x < (xj - xi) * (y - yi) / (yj - yi) + xi) hit = !hit;
        }
        return hit;
    }
    function countryAt(lon, lat) {
        for (let i = 0; i < countries.length; i++)
            for (const r of countries[i].r) if (inside(r, lon, lat)) return i;
        return -1;
    }

    // Find it! (the game)
    property bool quiz: false
    property int target: -1
    property int score: 0
    property int timeLeft: 0
    property string quizNote: ""
    function nextTarget() {
        const known = [];
        for (let i = 0; i < countries.length; i++) if (countries[i].cap && countries[i].p > 3000000) known.push(i);
        target = known[Math.floor(Math.random() * known.length)];
    }
    Timer { interval: 1000; running: win.quiz; repeat: true
        onTriggered: { win.timeLeft--; if (win.timeLeft <= 0) { win.quiz = false;
            if (win.score > saved.best) saved.best = win.score;
            win.trophy("max world-find " + win.score);
            win.quizNote = "Time's up! You found " + win.score + "."; } } }

    Timer {
        interval: 40; running: true; repeat: true
        onTriggered: { if (Date.now() > win.idleUntil && !win.quiz) { win.lon0 = (win.lon0 + 0.25) % 360; globe.requestPaint(); } }
    }

    // Stars.
    Repeater {
        model: 140
        delegate: Rectangle { required property int index; x: (index * 7919) % win.width; y: (index * 104729) % win.height
            width: 2; height: 2; radius: 1; color: "white"; opacity: 0.2 + (index % 5) * 0.15 }
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 20

        Item {
            Layout.preferredWidth: 680
            Layout.fillHeight: true
            Canvas {
                id: globe
                anchors.centerIn: parent
                width: Math.min(parent.width, parent.height); height: width
                onPaint: {
                    const c = getContext("2d"), R = width / 2 - 10, cx = width / 2, cy = height / 2;
                    c.reset();
                    const sea = c.createRadialGradient(cx - R * 0.3, cy - R * 0.3, R * 0.1, cx, cy, R);
                    sea.addColorStop(0, "#3b82f6"); sea.addColorStop(1, "#1e3a8a");
                    c.fillStyle = sea; c.beginPath(); c.arc(cx, cy, R, 0, 2 * Math.PI); c.fill();
                    c.lineWidth = 0.8; c.strokeStyle = "rgba(15,23,42,0.6)";
                    for (let i = 0; i < win.countries.length; i++) {
                        const k = win.countries[i];
                        c.fillStyle = i === win.picked ? "#fde047" : (win.colours[k.c] || "#94a3b8");
                        for (const r of k.r) {
                            let seen = false;
                            c.beginPath();
                            for (let j = 0; j < r.length; j += 2) {
                                const p = win.project(r[j], r[j + 1], R);
                                if (p[2]) seen = true;
                                if (j === 0) c.moveTo(cx + p[0], cy + p[1]); else c.lineTo(cx + p[0], cy + p[1]);
                            }
                            if (!seen) continue;
                            c.closePath(); c.fill(); c.stroke();
                        }
                    }
                    const shade = c.createRadialGradient(cx - R * 0.35, cy - R * 0.35, R * 0.2, cx, cy, R);
                    shade.addColorStop(0, "rgba(255,255,255,0.12)"); shade.addColorStop(0.7, "rgba(0,0,0,0)"); shade.addColorStop(1, "rgba(0,0,0,0.45)");
                    c.fillStyle = shade; c.beginPath(); c.arc(cx, cy, R, 0, 2 * Math.PI); c.fill();
                }
                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    property real px: 0
                    property real py: 0
                    property bool moved: false
                    onPressed: m => { px = m.x; py = m.y; moved = false; win.idleUntil = Date.now() + 8000; }
                    onPositionChanged: m => {
                        const R = globe.width / 2 - 10;
                        if (Math.abs(m.x - px) + Math.abs(m.y - py) > 3) moved = true;
                        win.lon0 -= (m.x - px) / R / win.rad * 0.9;
                        win.lat0 = Math.max(-80, Math.min(80, win.lat0 + (m.y - py) / R / win.rad * 0.9));
                        px = m.x; py = m.y; globe.requestPaint();
                    }
                    onReleased: m => {
                        win.idleUntil = Date.now() + 8000;
                        if (moved) return;
                        const ll = win.unproject(m.x - globe.width / 2, m.y - globe.height / 2, globe.width / 2 - 10);
                        if (!ll) return;
                        const i = win.countryAt(ll[0], ll[1]);
                        if (i < 0) { win.picked = -1; globe.requestPaint(); return; }
                        win.picked = i; globe.requestPaint(); win.trophy("add world-country " + win.countries[i].n);
                        if (win.quiz) {
                            if (i === win.target) { win.score++; win.quizNote = "✅ Yes! That's " + win.countries[i].n + "!"; win.nextTarget(); }
                            else win.quizNote = "❌ That's " + win.countries[i].n + ". Try again!";
                        }
                    }
                }
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 12
            Label { text: "🌍 World Explorer"; color: "white"; font.pixelSize: 30; font.bold: true }
            Label { text: "Drag the globe to spin it. Click a country!"; color: "#93c5fd"; font.pixelSize: 15; visible: !win.quiz }

            // Find it!
            Rectangle {
                visible: win.quiz
                Layout.fillWidth: true
                implicitHeight: quizCol.implicitHeight + 24
                radius: 16; color: "#7c2d12"
                ColumnLayout {
                    id: quizCol
                    anchors.fill: parent; anchors.margins: 12
                    Label { text: "🎯 Find: " + (win.target >= 0 ? win.countries[win.target].n : ""); color: "white"; font.pixelSize: 24; font.bold: true; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                    Label { text: win.target >= 0 ? "Hint: it's in " + win.countries[win.target].c : ""; color: "#fed7aa"; font.pixelSize: 15 }
                    Label { text: "Found: " + win.score + "     ⏱️ " + win.timeLeft + " s"; color: "white"; font.pixelSize: 18 }
                    RowLayout {
                        Button { text: "Skip"; onClicked: win.nextTarget() }
                        Button { text: "Stop"; onClicked: { win.quiz = false; win.quizNote = ""; } }
                    }
                }
            }
            Label { text: win.quizNote; color: "#fde68a"; font.pixelSize: 17; wrapMode: Text.WordWrap; Layout.fillWidth: true }

            // The country picked.
            Rectangle {
                Layout.fillWidth: true
                implicitHeight: info.implicitHeight + 28
                radius: 18
                color: "#0f172acc"
                border.color: "#1e40af"
                visible: win.picked >= 0
                readonly property var k: win.picked >= 0 ? win.countries[win.picked] : null
                ColumnLayout {
                    id: info
                    anchors.fill: parent
                    anchors.margins: 14
                    spacing: 8
                    RowLayout {
                        Label { text: parent.parent.parent.k ? win.flag(parent.parent.parent.k.f) : ""; font.pixelSize: 54 }
                        Label { text: parent.parent.parent.k ? parent.parent.parent.k.n : ""; color: "white"; font.pixelSize: 30; font.bold: true; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                    }
                    Repeater {
                        model: {
                            const k = win.picked >= 0 ? win.countries[win.picked] : null;
                            if (!k) return [];
                            const rows = [[k.e, "Continent", k.c]];
                            if (k.cap) rows.push(["🏛️", "Capital", k.cap]);
                            rows.push(["👨‍👩‍👧", "People", win.people(k.p)]);
                            return rows;
                        }
                        delegate: RowLayout {
                            required property var modelData
                            Label { text: modelData[0]; font.pixelSize: 22 }
                            Label { text: modelData[1]; color: "#93c5fd"; font.pixelSize: 15; font.bold: true; Layout.preferredWidth: 90 }
                            Label { text: modelData[2]; color: "white"; font.pixelSize: 17; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                        }
                    }
                    Label {
                        visible: text !== ""
                        text: win.picked >= 0 && win.countries[win.picked].fun ? "🤩 " + win.countries[win.picked].fun : ""
                        color: "#fde68a"; font.pixelSize: 16; wrapMode: Text.WordWrap; Layout.fillWidth: true
                    }
                }
            }
            Item { Layout.fillHeight: true }
            Button {
                text: "🎯 Find it! (60 seconds)" + (saved.best ? "   Best: " + saved.best : "")
                font.pixelSize: 18
                highlighted: true
                visible: !win.quiz
                Layout.fillWidth: true
                onClicked: { win.quiz = true; win.score = 0; win.timeLeft = 60; win.quizNote = ""; win.picked = -1; win.nextTarget(); }
            }
        }
    }
}

// Ollie Game Hub (ollie-gamehub): every game on one big screen, for the TV
// and the controller. Rows: all games, Steam, retro games, Ollie OS's games,
// and getting more. Left and right pick a game, up and down a row, A plays,
// B closes; X shows Play Time, Y the Trophies, Start the Quick Menu. The
// keyboard works too (arrows, Enter, Escape).
// The controller's buttons come from ollie-padd ($XDG_RUNTIME_DIR/ollie-pad-events).
// Says on the console: OLLIE-ACTION play KEY (games.py), run PROGRAM, quit.
// Arguments: list=FILE (games.py list), events=FILE, name= (yours).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1600
    height: 900
    visible: true
    visibility: Window.FullScreen
    title: "Ollie Game Hub"
    color: "#020617"

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }
    function read(f) {
        const x = new XMLHttpRequest();
        x.open("GET", "file://" + f, false);
        try { x.send(); return x.responseText; } catch (e) { return ""; }
    }

    readonly property var ollieGames: [
        { name: "Typing Game", emoji: "⌨️", icon: "typing", run: "ollie-typing" },
        { name: "Ollie Pet", emoji: "🐾", icon: "pet", run: "ollie-pet" },
        { name: "Couch Co-op", emoji: "🎮", icon: "coop", run: "ollie-coop" },
        { name: "Coding Quest", emoji: "🤖", icon: "codequest", run: "ollie-codequest" },
        { name: "Math Quest", emoji: "🧮", icon: "mathquest", run: "ollie-mathquest" },
        { name: "Spelling Bee", emoji: "🐝", icon: "spelling", run: "ollie-spelling" },
        { name: "World Explorer", emoji: "🌍", icon: "world", run: "ollie-world" },
        { name: "Space Explorer", emoji: "🪐", icon: "space", run: "ollie-space" }
    ]
    readonly property var more: [
        { name: "App Center", emoji: "🛍️", icon: "appcenter", run: "ollie-appcenter", note: "Free games, one click" },
        { name: "Epic & GOG games", emoji: "🎁", icon: "", run: "heroic", note: "Free games every week (Heroic)" },
        { name: "Steam Store", emoji: "🛒", icon: "", run: "steam-store", note: "Buy and get games" },
        { name: "Minecraft", emoji: "⛏️", icon: "", run: "ollie-minecraft", note: "Play Minecraft" },
        { name: "Play Time", emoji: "⏱️", icon: "playtime", run: "ollie-playtime", note: "How long you played" },
        { name: "Trophies", emoji: "🏆", icon: "trophies", run: "ollie-trophies", note: "Your trophies" }
    ]
    property var games: []
    Component.onCompleted: {
        try { games = JSON.parse(read(arg("list"))).games.filter(g => !(g.key || "").includes("/ollie-")); } catch (e) { games = []; }
        try { const e = JSON.parse(read(arg("events"))).events; lastSeq = e.length ? e[e.length - 1][0] : 0; } catch (e) {}
        say.text = "Hi" + (arg("name") ? " " + arg("name") : "") + "! What do you want to play?";
    }
    // [title, items]; each item: {name, emoji, cover, key | run, sub}
    readonly property var rows: {
        const r = [], all = games.map(g => ({ name: g.name, emoji: g.emoji, cover: g.cover, key: g.key, sub: g.system }));
        if (all.length) r.push(["🌟 All my games", all]);
        const steam = all.filter(g => g.sub === "Steam");
        if (steam.length) r.push(["🚂 Steam", steam]);
        const retro = all.filter(g => g.key.indexOf("file:") === 0);
        if (retro.length) r.push(["👾 Retro and my game files", retro]);
        r.push(["🧊 Ollie OS games", ollieGames.map(g => ({ name: g.name, emoji: g.emoji, cover: "file:///usr/share/icons/hicolor/256x256/apps/ollie-" + g.icon + ".png", run: g.run, sub: "Ollie OS" }))]);
        r.push(["➕ Get more and more", more.map(g => ({ name: g.name, emoji: g.emoji, cover: g.icon ? "file:///usr/share/icons/hicolor/256x256/apps/ollie-" + g.icon + ".png" : "", run: g.run, sub: g.note }))]);
        return r;
    }
    property int row: 0
    property var col: []          // the chosen tile in each row
    function colOf(r) { return col[r] || 0 }
    function setCol(r, c) { const a = col.slice(); a[r] = c; col = a; }
    readonly property var chosen: rows.length ? rows[row][1][Math.min(colOf(row), rows[row][1].length - 1)] : null

    function go(w) {
        if (w === "left") setCol(row, Math.max(0, colOf(row) - 1));
        else if (w === "right") setCol(row, Math.min(rows[row][1].length - 1, colOf(row) + 1));
        else if (w === "up") row = Math.max(0, row - 1);
        else if (w === "down") row = Math.min(rows.length - 1, row + 1);
        else if (w === "lb") row = Math.max(0, row - 1);
        else if (w === "rb") row = Math.min(rows.length - 1, row + 1);
        else if (w === "ok") play(chosen);
        else if (w === "back") act("quit");
        else if (w === "x") act("run ollie-playtime");
        else if (w === "y") act("run ollie-trophies");
        else if (w === "start") act("run ollie-quickmenu");
    }
    function play(g) {
        if (!g) return;
        if (g.key) act("play " + g.key); else act("run " + g.run);
    }
    property int lastSeq: 0
    Timer {
        interval: 60; running: true; repeat: true
        onTriggered: {
            let e;
            try { e = JSON.parse(win.read(win.arg("events"))).events; } catch (x) { return; }
            for (const [seq, w] of e) if (seq > win.lastSeq) { win.lastSeq = seq; win.go(w); }
        }
    }
    Item {
        anchors.fill: parent
        focus: true
        Keys.onPressed: (k) => {
            const m = { [Qt.Key_Left]: "left", [Qt.Key_Right]: "right", [Qt.Key_Up]: "up", [Qt.Key_Down]: "down",
                        [Qt.Key_Return]: "ok", [Qt.Key_Enter]: "ok", [Qt.Key_Space]: "ok", [Qt.Key_Escape]: "back",
                        [Qt.Key_PageUp]: "lb", [Qt.Key_PageDown]: "rb" };
            if (m[k.key]) { win.go(m[k.key]); k.accepted = true; }
        }
    }

    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: "#0b3b37" }
            GradientStop { position: 0.55; color: "#0b1220" }
            GradientStop { position: 1; color: "#1e1b4b" }
        }
    }
    // The chosen game's cover, big and soft behind everything.
    Image {
        anchors.fill: parent
        source: win.chosen && win.chosen.cover ? (win.chosen.cover.indexOf("file:") === 0 ? win.chosen.cover : "file://" + win.chosen.cover) : ""
        fillMode: Image.PreserveAspectCrop
        opacity: 0.12
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 40
        spacing: 10
        RowLayout {
            spacing: 16
            Image { source: "file:///usr/share/icons/hicolor/scalable/apps/ollie-os.svg"; sourceSize.width: 56; sourceSize.height: 56 }
            Label { text: "Game Hub"; color: "white"; font.pixelSize: 40; font.bold: true }
            Label { id: say; color: "#99f6e4"; font.pixelSize: 22; leftPadding: 20 }
            Item { Layout.fillWidth: true }
            Label { id: clock; color: "white"; font.pixelSize: 30; font.bold: true
                Timer { interval: 1000; running: true; repeat: true; triggeredOnStart: true; onTriggered: clock.text = Qt.formatTime(new Date(), "h:mm AP") } }
        }

        // The chosen game, big.
        ColumnLayout {
            Layout.topMargin: 10
            spacing: 2
            Label { text: win.chosen ? win.chosen.name : ""; color: "white"; font.pixelSize: 52; font.bold: true; elide: Text.ElideRight; Layout.maximumWidth: win.width - 100 }
            Label { text: win.chosen ? win.chosen.sub : ""; color: "#94a3b8"; font.pixelSize: 22 }
        }

        ListView {
            id: rowsView
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            model: win.rows
            currentIndex: win.row
            highlightRangeMode: ListView.ApplyRange
            preferredHighlightBegin: 0
            preferredHighlightEnd: 330
            highlightMoveDuration: 250
            spacing: 6
            delegate: ColumnLayout {
                id: rowItem
                required property var modelData
                required property int index
                width: rowsView.width
                height: 330
                spacing: 6
                Label { text: rowItem.modelData[0]; color: rowItem.index === win.row ? "white" : "#64748b"; font.pixelSize: 24; font.bold: true }
                ListView {
                    id: tiles
                    Layout.fillWidth: true
                    Layout.preferredHeight: 286
                    orientation: ListView.Horizontal
                    spacing: 6
                    leftMargin: 14
                    clip: false
                    model: rowItem.modelData[1]
                    currentIndex: win.colOf(rowItem.index)
                    highlightRangeMode: ListView.ApplyRange
                    preferredHighlightBegin: 40
                    preferredHighlightEnd: tiles.width - 260
                    highlightMoveDuration: 200
                    delegate: Item {
                        id: tile
                        required property var modelData
                        required property int index
                        readonly property bool on: rowItem.index === win.row && index === win.colOf(rowItem.index)
                        width: 190; height: 286
                        Rectangle {
                            anchors.centerIn: parent
                            width: 170; height: 240
                            scale: tile.on ? 1.12 : 0.95
                            Behavior on scale { NumberAnimation { duration: 150 } }
                            radius: 18
                            color: "#1e293b"
                            border.color: tile.on ? "#facc15" : "#334155"
                            border.width: tile.on ? 5 : 1
                            clip: true
                            Image {
                                id: cover
                                anchors.fill: parent
                                anchors.margins: tile.modelData.cover && tile.modelData.cover.indexOf("/icons/") > 0 ? 28 : 0
                                source: tile.modelData.cover ? (tile.modelData.cover.indexOf("file:") === 0 ? tile.modelData.cover : "file://" + tile.modelData.cover) : ""
                                fillMode: Image.PreserveAspectCrop
                                asynchronous: true
                            }
                            Label {
                                visible: cover.status !== Image.Ready
                                anchors.centerIn: parent
                                text: tile.modelData.emoji || "🎮"
                                font.pixelSize: 72
                            }
                            Rectangle {
                                visible: cover.status !== Image.Ready || tile.modelData.cover.indexOf("/icons/") > 0
                                anchors.bottom: parent.bottom
                                width: parent.width; height: 54
                                color: "#cc0f172a"
                                Label {
                                    anchors.fill: parent; anchors.margins: 6
                                    text: tile.modelData.name; color: "white"; font.pixelSize: 15; font.bold: true
                                    wrapMode: Text.WordWrap; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                                    elide: Text.ElideRight
                                }
                            }
                            MouseArea {
                                anchors.fill: parent
                                cursorShape: Qt.PointingHandCursor
                                onClicked: { win.row = rowItem.index; win.setCol(rowItem.index, tile.index); }
                                onDoubleClicked: win.play(tile.modelData)
                            }
                        }
                    }
                }
            }
        }

        // What the buttons do.
        RowLayout {
            spacing: 26
            Repeater {
                model: [["Ⓐ", "#22c55e", "Play"], ["Ⓑ", "#ef4444", "Close"], ["Ⓧ", "#3b82f6", "Play Time"], ["Ⓨ", "#eab308", "Trophies"], ["☰", "#94a3b8", "Quick Menu"]]
                delegate: RowLayout {
                    required property var modelData
                    spacing: 8
                    Label { text: modelData[0]; color: modelData[1]; font.pixelSize: 30; font.bold: true }
                    Label { text: modelData[2]; color: "#cbd5e1"; font.pixelSize: 19 }
                }
            }
            Item { Layout.fillWidth: true }
            Label { text: win.games.length + " games"; color: "#64748b"; font.pixelSize: 17 }
        }
    }
}

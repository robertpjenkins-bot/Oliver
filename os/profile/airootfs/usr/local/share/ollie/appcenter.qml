// Ollie App Center (ollie-appcenter): free games and apps picked for Ollie
// OS, in groups (Ollie apps: Ollie OS's own extras; game stores: Epic and
// GOG with Heroic; racing, building, adventure, strategy, puzzles, making
// things, learning). One click gets one (from Flathub, or for Ollie apps
// from the computer itself), then it's ready to play. Says on the console: OLLIE-ACTION install ID [ID...] (the first that
// Flathub has), play ID, remove ID.
// Arguments: installed=FILE (the app IDs installed, one a line), status=FILE
// (the helper's lines while getting one: "ID installing|done|failed").
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1100
    height: 760
    visible: true
    title: "Ollie App Center"
    color: "#0f172a"

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

    // [name, emoji, what it is, group, ages, Flathub IDs (the first one Flathub has is used)];
    // Ollie OS's own extra apps have the ID ollie-NAME (already on the computer).
    readonly property var apps: [
        ["Sticker Maker", "🌟", "Make stickers from pictures, drawings and shapes, and stick them on your desktop.", "Ollie apps", "Everyone", ["ollie-sticker"]],
        ["Ollie Animator", "🎞️", "Draw a cartoon frame by frame and save it as a GIF.", "Ollie apps", "Everyone", ["ollie-animator"]],
        ["Sound Board", "🥁", "Twelve big buttons with sounds, and record your own.", "Ollie apps", "Everyone", ["ollie-soundboard"]],
        ["Spelling Bee", "🐝", "Look at a word, then spell it. Eight levels, from easy to tricky.", "Ollie apps", "Everyone", ["ollie-spelling"]],
        ["World Explorer", "🌍", "Spin the globe: flags, capitals and facts for 176 countries.", "Ollie apps", "Everyone", ["ollie-world"]],
        ["Space Explorer", "🪐", "Visit the Sun, the planets, the Moon and Pluto.", "Ollie apps", "Everyone", ["ollie-space"]],
        ["Epic & GOG games", "🎁", "Heroic: play your Epic Games Store and GOG games, and get the free game Epic gives away every week. Sign in with a grown-up.", "Game stores", "Everyone", ["com.heroicgameslauncher.hgl"]],
        ["SuperTuxKart", "🏎️", "Kart racing with Tux and friends. Up to 4 players on one screen!", "Racing", "Everyone", ["net.supertuxkart.SuperTuxKart"]],
        ["Extreme Tux Racer", "🐧", "Slide down snowy mountains on your tummy, grabbing fish.", "Racing", "Everyone", ["net.sourceforge.ExtremeTuxRacer"]],
        ["Pixel Wheels", "🚗", "Top-down retro racing with power-ups.", "Racing", "Everyone", ["com.agateau.PixelWheels"]],
        ["Luanti", "⛏️", "A block world to build and explore, like a free Minecraft.", "Building", "Everyone", ["org.luanti.luanti", "net.minetest.Minetest"]],
        ["OpenTTD", "🚂", "Build trains, roads, ships and planes and run a transport company.", "Building", "Everyone", ["org.openttd.OpenTTD"]],
        ["Simutrans", "🏙️", "Connect towns with trains and buses and watch them grow.", "Building", "Everyone", ["com.simutrans.Simutrans"]],
        ["SuperTux", "🏃", "Jump and run through icy levels in this classic platformer.", "Adventure", "Everyone", ["org.supertuxproject.SuperTux"]],
        ["Frogatto", "🐸", "A frog's adventure: a colourful platformer with puzzles.", "Adventure", "Everyone", ["com.frogatto.Frogatto"]],
        ["Neverball", "⚽", "Tilt the floor to roll the ball and collect the coins.", "Adventure", "Everyone", ["org.neverball.Neverball"]],
        ["0 A.D.", "🏛️", "Build an ancient city and lead its people.", "Strategy", "10+", ["com.play0ad.zeroad"]],
        ["Mindustry", "🏭", "Build factories and conveyor belts to defend your base.", "Strategy", "Everyone", ["com.github.Anuken.Mindustry"]],
        ["Unciv", "🗺️", "Grow a small village into a great civilisation, turn by turn.", "Strategy", "Everyone", ["io.github.yairm210.unciv"]],
        ["Battle for Wesnoth", "🐉", "Fantasy battles on a map of hexagons.", "Strategy", "10+", ["org.wesnoth.Wesnoth"]],
        ["Frozen Bubble", "🫧", "Shoot bubbles to pop three of the same colour.", "Puzzles", "Everyone", ["org.frozen_bubble.frozen-bubble"]],
        ["Hedgewars", "🦔", "Silly turn-based battles between teams of hedgehogs.", "Puzzles", "8+", ["org.hedgewars.Hedgewars"]],
        ["Chess", "♟️", "Play chess against the computer or a friend.", "Puzzles", "Everyone", ["org.gnome.Chess"]],
        ["Sudoku", "🔢", "Number puzzles, from easy to very tricky.", "Puzzles", "Everyone", ["org.gnome.Sudoku"]],
        ["Mines", "💣", "Find the hidden mines using the number clues.", "Puzzles", "Everyone", ["org.gnome.Mines"]],
        ["Tux Paint", "🖍️", "A drawing program made for kids, with stamps and silly sounds.", "Making", "Everyone", ["org.tuxpaint.Tuxpaint"]],
        ["Krita", "🎨", "Paint and draw like the pros.", "Making", "Everyone", ["org.kde.krita"]],
        ["LMMS", "🎼", "Make music with instruments, beats and loops.", "Making", "Everyone", ["io.lmms.LMMS"]],
        ["Blender", "🧊", "Make 3D models and animations.", "Making", "10+", ["org.blender.Blender"]],
        ["GCompris", "🧩", "More than 100 learning games: reading, maths, science and more.", "Learning", "Everyone", ["org.kde.gcompris"]],
        ["Stellarium", "🔭", "A planetarium: see the real night sky, stars and planets.", "Learning", "Everyone", ["org.stellarium.Stellarium"]],
        ["KTouch", "⌨️", "Learn to type with all ten fingers.", "Learning", "Everyone", ["org.kde.ktouch"]],
        ["Marble", "🌐", "A virtual globe with maps of the whole world.", "Learning", "Everyone", ["org.kde.marble"]]
    ]
    readonly property var groups: [["All", "🌟"], ["Ollie apps", "🧊"], ["Game stores", "🎁"], ["Racing", "🏁"], ["Building", "🧱"], ["Adventure", "🗺️"], ["Strategy", "🏰"], ["Puzzles", "🧩"], ["Making", "🎨"], ["Learning", "🧠"]]
    property string group: "All"
    property string search: ""
    property var installed: []        // app IDs
    property var busy: ({})           // ID -> "installing" | "failed"

    function idOf(a) { for (const id of a[5]) if (installed.includes(id)) return id; return ""; }
    Component.onCompleted: installed = read(arg("installed")).split("\n").filter(l => l !== "")
    Timer {
        interval: 700; running: true; repeat: true
        onTriggered: {
            const b = {}, got = win.installed.slice();
            for (const line of win.read(win.arg("status")).split("\n")) {
                const p = line.split(" ");
                if (p.length < 2) continue;
                if (p[1] === "done") { delete b[p[0]]; if (!got.includes(p[0])) got.push(p[0]); }
                else if (p[1] === "removed") { delete b[p[0]]; const i = got.indexOf(p[0]); if (i >= 0) got.splice(i, 1); }
                else b[p[0]] = p[1];
            }
            if (JSON.stringify(b) !== JSON.stringify(win.busy)) win.busy = b;
            if (got.length !== win.installed.length) win.installed = got;
        }
    }
    function stateOf(a) {
        if (idOf(a)) return "installed";
        for (const id of a[5]) if (busy[id]) return busy[id];
        return "";
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 22
        spacing: 14

        RowLayout {
            spacing: 14
            Label { text: "🛍️ Ollie App Center"; color: "white"; font.pixelSize: 32; font.bold: true }
            Label { text: "Free games and apps, picked for you"; color: "#94a3b8"; font.pixelSize: 16; Layout.alignment: Qt.AlignBottom; bottomPadding: 6 }
            Item { Layout.fillWidth: true }
            TextField { placeholderText: "🔍 Find a game"; Layout.preferredWidth: 240; font.pixelSize: 16; onTextChanged: win.search = text.trim().toLowerCase() }
        }
        Flow {
            Layout.fillWidth: true
            spacing: 8
            Repeater {
                model: win.groups
                delegate: Button {
                    required property var modelData
                    text: modelData[1] + " " + modelData[0]
                    font.pixelSize: 15
                    highlighted: win.group === modelData[0]
                    onClicked: win.group = modelData[0]
                }
            }
        }
        GridView {
            id: grid
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            cellWidth: Math.floor(width / 3)
            cellHeight: 200
            model: win.apps.filter(a => (win.group === "All" || a[3] === win.group)
                && (win.search === "" || (a[0] + " " + a[2]).toLowerCase().includes(win.search)))
            delegate: Item {
                id: card
                required property var modelData
                readonly property string st: win.stateOf(modelData)
                width: grid.cellWidth
                height: grid.cellHeight
                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 7
                    radius: 20
                    color: "#16223b"
                    border.color: card.st === "installed" ? "#14b8a6" : "#22314f"
                    border.width: card.st === "installed" ? 2 : 1
                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 16
                        spacing: 6
                        RowLayout {
                            spacing: 12
                            Rectangle {
                                width: 64; height: 64; radius: 16
                                gradient: Gradient {
                                    GradientStop { position: 0; color: "#334155" }
                                    GradientStop { position: 1; color: "#1e293b" }
                                }
                                Label { anchors.centerIn: parent; visible: !ownIcon.visible; text: card.modelData[1]; font.pixelSize: 38 }
                                Image {
                                    id: ownIcon
                                    anchors.fill: parent
                                    visible: status === Image.Ready
                                    source: card.modelData[5][0].indexOf("ollie-") === 0 ? "file:///usr/share/icons/hicolor/128x128/apps/" + card.modelData[5][0] + ".png" : ""
                                    sourceSize: Qt.size(128, 128)
                                }
                            }
                            ColumnLayout {
                                spacing: 2
                                Label { text: card.modelData[0]; color: "white"; font.pixelSize: 19; font.bold: true }
                                Label { text: card.modelData[3] + "  ·  " + (card.modelData[4] === "Everyone" ? "👍 For everyone" : "Ages " + card.modelData[4]); color: "#94a3b8"; font.pixelSize: 13 }
                            }
                        }
                        Label { text: card.modelData[2]; color: "#cbd5e1"; font.pixelSize: 14; wrapMode: Text.WordWrap; Layout.fillWidth: true; Layout.fillHeight: true }
                        RowLayout {
                            Button {
                                Layout.fillWidth: true
                                font.pixelSize: 16
                                font.bold: true
                                highlighted: card.st === "" || card.st === "installed" || card.st === "failed"
                                enabled: card.st !== "installing"
                                text: card.st === "installed" ? "▶ Open" : card.st === "installing" ? "⏳ Getting it..." : card.st === "failed" ? "🔁 Try again" : card.modelData[5][0].indexOf("ollie-") === 0 ? "⬇️ Install" : "⬇️ Get it (free)"
                                onClicked: {
                                    if (card.st === "installed") { win.act("play " + win.idOf(card.modelData)); return; }
                                    const b = Object.assign({}, win.busy); b[card.modelData[5][0]] = "installing"; win.busy = b;
                                    win.act("install " + card.modelData[5].join(" "));
                                }
                            }
                            Button {
                                visible: card.st === "installed"
                                text: "🗑"
                                ToolTip.visible: hovered; ToolTip.text: "Remove it"
                                onClicked: win.act("remove " + win.idOf(card.modelData))
                            }
                        }
                    }
                }
            }
        }
        Label {
            Layout.alignment: Qt.AlignHCenter
            text: "Ollie apps install at once. Other games come from Flathub, the free app store: getting one needs the internet and takes a few minutes."
            color: "#64748b"; font.pixelSize: 13
        }
    }
}

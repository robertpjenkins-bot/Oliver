// Games (ollie-games): every game in one place, as big tiles, and a place to
// drop game files (ROMs, ISOs, Windows games). Says on its console what to
// do: OLLIE-ACTION play KEY, add FILE-URL (then refresh), refresh, folder.
// Argument: list=FILE (games.py's list: {"games": [...], "folder": ...}).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1100
    height: 760
    visible: true
    title: "Games"
    color: "#0b1220"

    function arg(name) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(name + "=") === 0)
                return a[i].substring(name.length + 1);
        return "";
    }
    function act(what) { console.log("OLLIE-ACTION " + what) }

    property var games: []
    property string filter: "all"
    property string search: ""
    readonly property var shown: games.filter(g =>
        (filter === "all" || (filter === "steam" && g.kind === "steam") || (filter === "apps" && g.kind === "app")
            || (filter === "files" && g.kind === "file"))
        && (search === "" || g.name.toLowerCase().indexOf(search.toLowerCase()) >= 0))

    Component.onCompleted: {
        const x = new XMLHttpRequest();
        x.open("GET", "file://" + arg("list"), false);
        x.send();
        try { games = JSON.parse(x.responseText).games; } catch (e) { games = []; }
    }

    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: "#1e1b4b" }
            GradientStop { position: 1; color: "#0b1220" }
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 14

        RowLayout {
            spacing: 12
            Label { text: "🕹️"; font.pixelSize: 40 }
            Label { text: "Games"; color: "white"; font.pixelSize: 34; font.bold: true }
            Label { text: win.games.length + (win.games.length === 1 ? " game" : " games"); color: "#a5b4fc"; font.pixelSize: 16; Layout.leftMargin: 6 }
            Item { Layout.fillWidth: true }
            TextField {
                placeholderText: "🔍 Find a game"
                Layout.preferredWidth: 260
                onTextChanged: win.search = text
            }
        }

        RowLayout {
            spacing: 8
            Repeater {
                model: [{ k: "all", t: "All" }, { k: "steam", t: "🎮 Steam" }, { k: "apps", t: "⛏️ Apps" }, { k: "files", t: "💿 My game files" }]
                delegate: Button {
                    required property var modelData
                    text: modelData.t
                    highlighted: win.filter === modelData.k
                    onClicked: win.filter = modelData.k
                }
            }
            Item { Layout.fillWidth: true }
            Button { text: "📁 My Games folder"; onClicked: win.act("folder") }
            Button { text: "🔄 Refresh"; onClicked: { win.act("refresh"); Qt.quit(); } }
        }

        // The drop place.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 64
            radius: 14
            color: drop.containsDrag ? "#4338ca" : "#1f2a4a"
            border.color: "#818cf8"
            border.width: 2
            Label {
                anchors.centerIn: parent
                text: drop.containsDrag ? "Let go to add them!"
                    : "⬇️  Drag game files here: ROMs (NES, SNES, Game Boy, N64, DS...), ISOs (PlayStation, PSP, GameCube), Windows games (.exe)"
                color: "white"
                font.pixelSize: 15
                elide: Text.ElideRight
                width: parent.width - 30
                horizontalAlignment: Text.AlignHCenter
            }
        }

        ScrollView {
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            contentWidth: availableWidth

            Flow {
                width: parent.width
                spacing: 14

                Repeater {
                    model: win.shown
                    delegate: Rectangle {
                        required property var modelData
                        width: 160
                        height: 260
                        radius: 14
                        color: tile.containsMouse ? "#3730a3" : "#1e293b"
                        border.color: tile.containsMouse ? "#c7d2fe" : "transparent"
                        border.width: 2
                        scale: tile.pressed ? 0.96 : 1

                        Rectangle {
                            id: art
                            anchors.top: parent.top
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.margins: 8
                            height: 180
                            radius: 10
                            color: modelData.kind === "steam" ? "#1b2838" : modelData.kind === "app" ? "#14532d" : "#4c1d95"
                            clip: true
                            Image {
                                anchors.fill: parent
                                source: modelData.cover ? "file://" + modelData.cover : ""
                                fillMode: modelData.kind === "app" ? Image.PreserveAspectFit : Image.PreserveAspectCrop
                                anchors.margins: modelData.kind === "app" ? 30 : 0
                                visible: status === Image.Ready
                                asynchronous: true
                            }
                            Label {
                                anchors.centerIn: parent
                                text: modelData.emoji
                                font.pixelSize: 64
                                visible: !modelData.cover
                            }
                        }
                        Label {
                            anchors.top: art.bottom
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.margins: 8
                            text: modelData.name
                            color: "white"
                            font.pixelSize: 14
                            font.bold: true
                            wrapMode: Text.WordWrap
                            maximumLineCount: 2
                            elide: Text.ElideRight
                        }
                        Label {
                            anchors.bottom: parent.bottom
                            anchors.left: parent.left
                            anchors.margins: 8
                            text: modelData.system
                            color: "#a5b4fc"
                            font.pixelSize: 12
                        }
                        MouseArea {
                            id: tile
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: win.act("play " + modelData.key)
                        }
                        ToolTip.visible: tile.containsMouse
                        ToolTip.delay: 600
                        ToolTip.text: "Play " + modelData.name
                    }
                }
            }
        }

        Label {
            visible: win.shown.length === 0
            Layout.alignment: Qt.AlignHCenter
            text: win.games.length === 0 ? "No games yet! Get some on Steam, or drag your game files here." : "No game with that name."
            color: "#c7d2fe"
            font.pixelSize: 18
        }
    }

    // Dropping anywhere on the window adds the files.
    DropArea {
        id: drop
        anchors.fill: parent
        keys: ["text/uri-list"]
        onDropped: d => {
            if (!d.hasUrls)
                return;
            for (let i = 0; i < d.urls.length; i++)
                win.act("add " + d.urls[i]);
            d.acceptProposedAction();
            win.act("refresh");
            Qt.quit();
        }
    }
}

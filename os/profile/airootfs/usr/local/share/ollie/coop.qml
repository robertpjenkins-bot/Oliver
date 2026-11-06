// Couch Co-op (ollie-coop): plug in 2 to 4 controllers; each gets a player
// card (Player 1 blue, 2 red, 3 green, 4 yellow) that lights up when its
// buttons are pressed, so everyone knows which controller is theirs. Then
// start playing together: Steam's big screen, or free SuperTuxKart.
// Says on the console: OLLIE-ACTION steam | supertuxkart.
// Argument: pads=FILE (pads.py's live file).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 980
    height: 640
    visible: true
    title: "Couch Co-op"
    color: "#1A1B1F" // a stage for the players

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }
    property var players: []
    property int lastCount: 0
    readonly property var colours: ["#336B9E", "#C8473D", "#4F8A45", "#D9A021"]
    readonly property var names: ["Player 1", "Player 2", "Player 3", "Player 4"]

    Timer {
        interval: 100; running: true; repeat: true
        onTriggered: {
            const x = new XMLHttpRequest();
            x.open("GET", "file://" + win.arg("pads"), false);
            try { x.send(); win.players = JSON.parse(x.responseText).players; } catch (e) {}
            if (win.players.length > win.lastCount && win.players.length >= 2) win.act("trophy max coop-players " + win.players.length);
            win.lastCount = win.players.length;
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 22
        spacing: 16

        RowLayout {
            Image { source: "file:///usr/share/icons/hicolor/64x64/apps/ollie-coop.png"; sourceSize: Qt.size(48, 48) }
            ColumnLayout {
                spacing: 0
                Label { text: "Couch Co-op"; color: "white"; font.pixelSize: 32; font.bold: true }
                Label {
                    color: "#9A9CA3"; font.pixelSize: 15
                    text: win.players.length === 0 ? "Plug in controllers (or pair them with Bluetooth): up to 4 players."
                        : "Press a button: your card lights up. That's you!"
                }
            }
        }

        GridLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            columns: 4
            columnSpacing: 14
            Repeater {
                model: 4
                delegate: Rectangle {
                    required property int index
                    readonly property var p: index < win.players.length ? win.players[index] : null
                    readonly property bool pressing: p !== null && p.held.length > 0
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: 6
                    color: p === null ? "#232429" : pressing ? win.colours[index] : Qt.darker(win.colours[index], 2.2)
                    border.color: p === null ? "#33353B" : win.colours[index]
                    border.width: 4
                    scale: pressing ? 1.04 : 1
                    Behavior on scale { NumberAnimation { duration: 80 } }
                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 16
                        spacing: 10
                        Label { text: win.names[index]; color: "white"; font.pixelSize: 24; font.bold: true; Layout.alignment: Qt.AlignHCenter }
                        Label {
                            text: p === null ? "" : pressing ? "" : ""
                            font.pixelSize: 90; Layout.alignment: Qt.AlignHCenter
                        }
                        Label {
                            text: p === null ? "No controller yet" : p.name
                            color: "#E4E4E7"; font.pixelSize: 14; wrapMode: Text.WordWrap; horizontalAlignment: Text.AlignHCenter
                            Layout.fillWidth: true
                        }
                        Label {
                            visible: pressing
                            text: p ? p.held.join(" + ") : ""
                            color: "white"; font.pixelSize: 22; font.bold: true; Layout.alignment: Qt.AlignHCenter
                        }
                        Item { Layout.fillHeight: true }
                        Label {
                            visible: p !== null
                            text: p ? p.presses + " presses" : ""
                            color: "#C9CAD0"; font.pixelSize: 13; Layout.alignment: Qt.AlignHCenter
                        }
                    }
                }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 120
            radius: 6
            color: "#232429"
            RowLayout {
                anchors.fill: parent
                anchors.margins: 16
                spacing: 16
                ColumnLayout {
                    Layout.fillWidth: true
                    Label { text: "Play together!"; color: "white"; font.pixelSize: 20; font.bold: true }
                    Label {
                        text: "Steam's big screen shows games for playing on one screen (look for \"Shared/Split Screen\").\nSuperTuxKart is a free racing game for up to 4 players on one screen."
                        color: "#9A9CA3"; font.pixelSize: 14; wrapMode: Text.WordWrap; Layout.fillWidth: true
                    }
                }
                Button { text: "Steam big screen"; font.pixelSize: 16; onClicked: win.act("steam") }
                Button { text: "SuperTuxKart"; font.pixelSize: 16; onClicked: win.act("supertuxkart") }
            }
        }
    }
}

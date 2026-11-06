// Trophies (ollie-trophies): your profile, and every trophy in Ollie OS's
// apps: the ones you've earned in colour, the rest grey with how far along
// you are.
// Arguments: data=FILE (ollie-trophy json), name= (yours), pet= (your Ollie Pet's name).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 980
    height: 720
    visible: true
    title: "Trophies"
    color: "#F4EFE6" // a wall of medals
    // A medal: a ribbon and a disc (gold, or grey when not won yet), with a letter.
    component Medal: Item {
        property bool won: false
        property string letter: ""
        width: 56; height: 64
        Rectangle { x: 14; width: 12; height: 26; color: won ? "#C8473D" : "#C9C3B8"; rotation: -12 }
        Rectangle { x: 30; width: 12; height: 26; color: won ? "#336B9E" : "#C9C3B8"; rotation: 12 }
        Rectangle {
            y: 16; anchors.horizontalCenter: parent.horizontalCenter
            width: 46; height: 46; radius: 23
            color: won ? "#D9A021" : "#DDD7CC"; border.color: won ? "#A67712" : "#C9C3B8"; border.width: 3
            Label { anchors.centerIn: parent; text: parent.parent.letter; color: won ? "white" : "#A8A193"; font.pixelSize: 20; font.bold: true }
        }
    }

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    property var trophies: []
    readonly property int got: trophies.filter(t => t.earned !== "").length
    property string show: "all"   // all, got, todo
    Component.onCompleted: {
        const x = new XMLHttpRequest();
        x.open("GET", "file://" + arg("data"), false);
        try { x.send(); trophies = JSON.parse(x.responseText); } catch (e) { trophies = []; }
        // Earned ones first, then the nearest to being earned.
        trophies = trophies.slice().sort((a, b) => (b.earned !== "") - (a.earned !== "") || (b.now / b.target) - (a.now / a.target));
    }
    readonly property string rank: got >= 30 ? "Ollie OS Legend" : got >= 20 ? "Champion" : got >= 12 ? "Gold" : got >= 6 ? "Silver" : got >= 1 ? "Bronze" : "Just starting"

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 24
        spacing: 16

        // Profile.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 130
            radius: 8
            color: "#22252B"
            RowLayout {
                anchors.fill: parent
                anchors.margins: 22
                spacing: 20
                Image { source: "file:///usr/share/icons/hicolor/128x128/apps/ollie-trophies.png"; sourceSize: Qt.size(80, 80) }
                ColumnLayout {
                    spacing: 2
                    Label { text: win.arg("name") || "You"; color: "white"; font.pixelSize: 32; font.bold: true }
                    Label { text: win.rank + (win.arg("pet") ? "  ·  " + win.arg("pet") : ""); color: "#D9A021"; font.pixelSize: 18 }
                }
                Item { Layout.fillWidth: true }
                ColumnLayout {
                    Label { text: win.got + " / " + win.trophies.length; color: "white"; font.pixelSize: 40; font.bold: true; Layout.alignment: Qt.AlignRight }
                    Label { text: "trophies"; color: "#C9CAD0"; font.pixelSize: 16; Layout.alignment: Qt.AlignRight }
                }
            }
        }

        RowLayout {
            spacing: 8
            Repeater {
                model: [["all", "All"], ["got", "Got them"], ["todo", "Still to get"]]
                delegate: Button {
                    required property var modelData
                    text: modelData[1]
                    highlighted: win.show === modelData[0]
                    onClicked: win.show = modelData[0]
                }
            }
            Item { Layout.fillWidth: true }
            Label { text: "Play Ollie OS's apps to earn more."; color: "#6B6F78"; font.pixelSize: 15 }
        }

        GridView {
            id: grid
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            cellWidth: Math.floor(width / 4)
            cellHeight: 200
            model: win.trophies.filter(t => win.show === "all" || (win.show === "got") === (t.earned !== ""))
            delegate: Item {
                required property var modelData
                readonly property bool have: modelData.earned !== ""
                width: grid.cellWidth
                height: grid.cellHeight
                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 7
                    radius: 6
                    color: "white"
                    border.color: parent.have ? "#D9A021" : "#DDD5C7"
                    border.width: parent.have ? 2 : 1
                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 12
                        spacing: 4
                        Medal { won: parent.parent.parent.have; letter: modelData.name[0]; Layout.alignment: Qt.AlignHCenter }
                        Label { text: modelData.name; color: "#22252B"; font.pixelSize: 16; font.bold: true; Layout.alignment: Qt.AlignHCenter }
                        Label {
                            text: modelData.how; color: "#6B6F78"; font.pixelSize: 12; wrapMode: Text.WordWrap
                            horizontalAlignment: Text.AlignHCenter; Layout.fillWidth: true
                        }
                        Item { Layout.fillHeight: true }
                        ProgressBar {
                            visible: !parent.parent.parent.have && modelData.target > 1
                            Layout.fillWidth: true
                            from: 0; to: modelData.target; value: Math.min(modelData.now, modelData.target)
                        }
                        Label {
                            Layout.alignment: Qt.AlignHCenter
                            font.pixelSize: 12
                            color: parent.parent.parent.have ? "#A67712" : "#8A8D94"
                            text: parent.parent.parent.have ? "Won " + modelData.earned
                                : modelData.target > 1 ? Math.min(modelData.now, modelData.target) + " / " + modelData.target : "Not yet"
                        }
                    }
                }
            }
        }
    }
}

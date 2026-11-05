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
    color: "#1e1b4b"

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
    readonly property string rank: got >= 30 ? "🌟 Ollie OS Legend" : got >= 20 ? "👑 Champion" : got >= 12 ? "🥇 Gold" : got >= 6 ? "🥈 Silver" : got >= 1 ? "🥉 Bronze" : "🥚 Just starting"

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 24
        spacing: 16

        // Profile.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 130
            radius: 24
            gradient: Gradient {
                orientation: Gradient.Horizontal
                GradientStop { position: 0; color: "#7c3aed" }
                GradientStop { position: 1; color: "#db2777" }
            }
            RowLayout {
                anchors.fill: parent
                anchors.margins: 22
                spacing: 20
                Rectangle {
                    width: 86; height: 86; radius: 43; color: "white"
                    Label { anchors.centerIn: parent; text: "🏆"; font.pixelSize: 50 }
                }
                ColumnLayout {
                    spacing: 2
                    Label { text: win.arg("name") || "You"; color: "white"; font.pixelSize: 32; font.bold: true }
                    Label { text: win.rank + (win.arg("pet") ? "   ·   🐾 " + win.arg("pet") : ""); color: "#fce7f3"; font.pixelSize: 18 }
                }
                Item { Layout.fillWidth: true }
                ColumnLayout {
                    Label { text: win.got + " / " + win.trophies.length; color: "white"; font.pixelSize: 40; font.bold: true; Layout.alignment: Qt.AlignRight }
                    Label { text: "trophies"; color: "#fce7f3"; font.pixelSize: 16; Layout.alignment: Qt.AlignRight }
                }
            }
        }

        RowLayout {
            spacing: 8
            Repeater {
                model: [["all", "All"], ["got", "🏆 Got them"], ["todo", "🎯 Still to get"]]
                delegate: Button {
                    required property var modelData
                    text: modelData[1]
                    highlighted: win.show === modelData[0]
                    onClicked: win.show = modelData[0]
                }
            }
            Item { Layout.fillWidth: true }
            Label { text: "Play Ollie OS's apps to earn more!"; color: "#c7d2fe"; font.pixelSize: 15 }
        }

        GridView {
            id: grid
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            cellWidth: Math.floor(width / 4)
            cellHeight: 190
            model: win.trophies.filter(t => win.show === "all" || (win.show === "got") === (t.earned !== ""))
            delegate: Item {
                required property var modelData
                readonly property bool have: modelData.earned !== ""
                width: grid.cellWidth
                height: grid.cellHeight
                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 7
                    radius: 18
                    color: parent.have ? "#312e81" : "#26234d"
                    border.color: parent.have ? "#facc15" : "#3b3770"
                    border.width: parent.have ? 3 : 1
                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 12
                        spacing: 4
                        Label {
                            text: modelData.emoji; font.pixelSize: 46; Layout.alignment: Qt.AlignHCenter
                            opacity: parent.parent.parent.have ? 1 : 0.25
                        }
                        Label { text: modelData.name; color: "white"; font.pixelSize: 16; font.bold: true; Layout.alignment: Qt.AlignHCenter }
                        Label {
                            text: modelData.how; color: "#c7d2fe"; font.pixelSize: 12; wrapMode: Text.WordWrap
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
                            color: parent.parent.parent.have ? "#fde047" : "#a5b4fc"
                            text: parent.parent.parent.have ? "🏆 Got it! " + modelData.earned
                                : modelData.target > 1 ? Math.min(modelData.now, modelData.target) + " / " + modelData.target : "🔒 Not yet"
                        }
                    }
                }
            }
        }
    }
}

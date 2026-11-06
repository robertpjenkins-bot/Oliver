import QtQuick
import QtQuick.Layouts
import org.kde.plasma.plasmoid
import org.kde.plasma.components as PlasmaComponents
import org.kde.plasma.plasma5support as P5Support
import "facts.js" as Facts

// Fun Fact of the Day: a card on the desktop with something new to know
// every day; "Another one" shows the next. Reading facts counts towards a
// trophy (ollie-trophy).
PlasmoidItem {
    id: root

    preferredRepresentation: fullRepresentation
    function dayNumber() { const d = new Date(); return d.getFullYear() * 366 + Math.floor((d - new Date(d.getFullYear(), 0, 0)) / 86400000); }
    property int today: dayNumber()
    property int extra: 0
    readonly property var fact: Facts.list[(today + extra) % Facts.list.length]

    P5Support.DataSource {
        id: run
        engine: "executable"
        connectedSources: []
        onNewData: (source) => disconnectSource(source)
    }
    function shell(cmd) { run.connectSource(cmd) }

    Timer { // a new day: back to that day's fact
        interval: 30 * 60 * 1000; running: true; repeat: true
        onTriggered: { const d = root.dayNumber(); if (d !== root.today) { root.today = d; root.extra = 0; } }
    }

    fullRepresentation: Rectangle {
        Layout.minimumWidth: 300
        Layout.preferredWidth: 400
        Layout.minimumHeight: 170
        Layout.preferredHeight: 210
        radius: 22
        gradient: Gradient {
            GradientStop { position: 0; color: "#e60f766e" }
            GradientStop { position: 1; color: "#e6312e81" }
        }
        border.color: "#5eead4"
        border.width: 2

        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 16
            spacing: 8
            RowLayout {
                Text { text: "💡"; font.pixelSize: 22 }
                Text { text: "Fun fact of the day"; color: "#ccfbf1"; font.pixelSize: 15; font.bold: true; Layout.fillWidth: true }
                Text { text: root.fact[0]; font.pixelSize: 34 }
            }
            Text {
                text: root.fact[1]
                color: "white"
                font.pixelSize: 18
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
                Layout.fillHeight: true
            }
            RowLayout {
                Item { Layout.fillWidth: true }
                PlasmaComponents.Button {
                    text: "Another one ▶"
                    onClicked: { root.extra++; root.shell("/usr/local/bin/ollie-trophy count facts"); }
                }
            }
        }
    }
}

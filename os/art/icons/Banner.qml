// Ollie OS's banner (docs/ollie-os-banner.png), for the GitHub page and the
// release notes: the Ollie cube, its name and version, and its app icons.
import QtQuick

Window {
    width: 1600
    height: 560
    visible: true
    color: "#083c38"
    readonly property string icons: Qt.resolvedUrl("../../profile/airootfs/usr/share/icons/hicolor/128x128/apps/")
    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            orientation: Gradient.Horizontal
            GradientStop { position: 0; color: "#0f766e" }
            GradientStop { position: 0.6; color: "#0b4f4a" }
            GradientStop { position: 1; color: "#3b0764" }
        }
    }
    Repeater { // stars
        model: 40
        delegate: Text {
            required property int index
            x: (index * 397) % 1600; y: (index * 151) % 560
            text: index % 5 === 0 ? "✨" : "·"; color: "#5eead4"; opacity: 0.35; font.pixelSize: index % 5 === 0 ? 22 : 40
        }
    }
    Image {
        x: 90; y: 90
        source: "../ollie-os.svg"
        sourceSize.width: 250; sourceSize.height: 250
    }
    Column {
        x: 390; y: 92
        spacing: 6
        Row {
            spacing: 18
            Text { text: "Ollie OS"; color: "white"; font.pixelSize: 108; font.bold: true }
            Rectangle {
                anchors.verticalCenter: parent.verticalCenter
                width: v.width + 36; height: 64; radius: 32; color: "#facc15"
                Text { id: v; anchors.centerIn: parent; text: "1.4"; font.pixelSize: 40; font.bold: true; color: "#422006" }
            }
        }
        Text { text: "The fun gaming computer, made by biobro 🎮"; color: "#ccfbf1"; font.pixelSize: 40 }
        Text { text: "Steam · Minecraft · retro games · Ollie Pet · Coding Quest · World Explorer · and more"; color: "#99f6e4"; font.pixelSize: 26; topPadding: 6 }
    }
    Row {
        x: 92; y: 400
        spacing: 22
        Repeater {
            model: ["games", "pet", "coop", "world", "space", "codequest", "spelling", "sticker", "comic", "animator", "soundboard", "typing"]
            delegate: Image {
                required property string modelData
                source: icons + "ollie-" + modelData + ".png"
                width: 96; height: 96
            }
        }
    }
}

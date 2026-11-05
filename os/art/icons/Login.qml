// Ollie OS's login screen picture (usr/local/share/ollie/login-background.png):
// the Ollie OS background with the cube up in the corner and the Ollie Pet
// waving hello at the bottom; the middle stays clear for the login box.
import QtQuick

Window {
    width: 1920
    height: 1080
    visible: true
    color: "#083c38"
    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: "#0f766e" }
            GradientStop { position: 0.6; color: "#0b4f4a" }
            GradientStop { position: 1; color: "#083c38" }
        }
    }
    Rectangle { x: 960 - 600; y: 540 - 600; width: 1200; height: 1200; radius: 600; color: "#145eead4" }
    Repeater {
        model: 60
        delegate: Text {
            required property int index
            x: (index * 389) % 1920; y: (index * 173) % 1080
            text: index % 6 === 0 ? "✦" : "·"; color: "#99f6e4"; opacity: 0.35; font.pixelSize: index % 6 === 0 ? 22 : 40
        }
    }
    Image { x: 60; y: 50; source: "../ollie-os.svg"; sourceSize.width: 110; sourceSize.height: 110 }
    Text { x: 190; y: 78; text: "Ollie OS"; color: "white"; font.pixelSize: 48; font.bold: true }
    // The hill and the pet.
    Rectangle { x: 1250; y: 900; width: 1100; height: 700; radius: 550; color: "#0d9488"; opacity: 0.8 }
    Loader {
        x: 1530; y: 790
        source: "../../profile/airootfs/usr/local/share/ollie/PetBody.qml"
        onLoaded: { item.stage = 2; item.scale = 1.5; item.mood = "happy"; }
    }
    Text { x: 1700; y: 760; text: "👋"; font.pixelSize: 80; rotation: 15 }
    Rectangle {
        x: 1330; y: 610; width: 380; height: 96; radius: 34
        color: "white"; border.color: "#0b4f4a"; border.width: 3
        Text { anchors.centerIn: parent; text: "Hi! Welcome back! 👋"; color: "#0b4f4a"; font.pixelSize: 32; font.bold: true }
    }
    Text { x: 1240; y: 860; text: "🌷"; font.pixelSize: 54 }
    Text { x: 1840; y: 900; text: "🌼"; font.pixelSize: 48 }
}

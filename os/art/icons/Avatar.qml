// The Ollie Pet as a person's picture on the login screen
// (usr/local/share/ollie/avatar.png, 256 px).
import QtQuick

Window {
    width: 256
    height: 256
    visible: true
    color: "transparent"
    Rectangle {
        anchors.fill: parent; radius: 128
        gradient: Gradient {
            GradientStop { position: 0; color: "#7dd3fc" }
            GradientStop { position: 1; color: "#0284c7" }
        }
    }
    Loader {
        anchors.centerIn: parent
        anchors.verticalCenterOffset: 6
        source: "../../profile/airootfs/usr/local/share/ollie/PetBody.qml"
        onLoaded: { item.stage = 1; item.scale = 1.45; item.mood = "happy"; }
    }
}

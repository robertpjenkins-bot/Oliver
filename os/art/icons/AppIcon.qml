// One Ollie OS app icon (make-app-icons.py draws each at 256 px): a rounded
// tile in the app's colours, with a soft shine, and its picture (an emoji,
// or the Ollie Pet itself) big in the middle.
import QtQuick

Window {
    id: win
    width: 256
    height: 256
    visible: true
    color: "transparent"
    property string emoji: "⭐"
    property color top: "#a855f7"
    property color bottom: "#6d28d9"
    property bool pet: false

    Rectangle { // shadow
        x: 14; y: 20; width: 228; height: 228; radius: 56
        color: "#40000000"
    }
    Rectangle {
        id: tile
        x: 14; y: 12; width: 228; height: 228; radius: 56
        gradient: Gradient {
            GradientStop { position: 0; color: win.top }
            GradientStop { position: 1; color: win.bottom }
        }
        border.color: Qt.lighter(win.top, 1.25); border.width: 3
        Rectangle { // shine, inside the rounded corners
            x: 6; y: 6; width: parent.width - 12; height: 104; radius: 50
            gradient: Gradient {
                GradientStop { position: 0; color: "#50ffffff" }
                GradientStop { position: 1; color: "#00ffffff" }
            }
        }
        Text {
            visible: !win.pet
            anchors.centerIn: parent
            anchors.verticalCenterOffset: 4
            text: win.emoji
            font.pixelSize: 132
            style: Text.Raised; styleColor: "#30000000"
        }
        Loader {
            active: win.pet
            anchors.centerIn: parent
            anchors.verticalCenterOffset: 8
            source: "../../profile/airootfs/usr/local/share/ollie/PetBody.qml"
            onLoaded: { item.stage = 1; item.scale = 1.25; }
        }
    }
}

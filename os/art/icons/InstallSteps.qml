// "How to get Ollie OS in 3 steps" (docs/install-steps.png), for the README.
import QtQuick

Window {
    width: 1600
    height: 520
    visible: true
    color: "#0f172a"
    readonly property var steps: [
        { n: "1", emoji: "⬇️", title: "Get the Installer", text: "From the Releases page:\nWindows (.exe), Mac (.dmg),\nLinux or Steam Deck (.AppImage)", c1: "#38bdf8", c2: "#0369a1" },
        { n: "2", emoji: "🛠️", title: "Run it", text: "It downloads Arch Linux\nand turns it into Ollie OS.\n(1 to 2 hours, about 20 GB free)", c1: "#a78bfa", c2: "#6d28d9" },
        { n: "3", emoji: "💾", title: "Put it on a USB stick", text: "Then start the computer from it.\nSteam Deck: hold Volume Down.\nDell: tap F12. Mac: hold Option.", c1: "#4ade80", c2: "#15803d" }
    ]
    Text { x: 0; y: 26; width: parent.width; horizontalAlignment: Text.AlignHCenter; text: "Get Ollie OS in 3 steps"; color: "white"; font.pixelSize: 54; font.bold: true }
    Row {
        x: 50; y: 120
        spacing: 50
        Repeater {
            model: steps
            delegate: Rectangle {
                required property var modelData
                required property int index
                width: 467; height: 360; radius: 36
                gradient: Gradient {
                    GradientStop { position: 0; color: modelData.c1 }
                    GradientStop { position: 1; color: modelData.c2 }
                }
                Rectangle {
                    x: 24; y: 24; width: 72; height: 72; radius: 36; color: "white"
                    Text { anchors.centerIn: parent; text: modelData.n; font.pixelSize: 44; font.bold: true; color: modelData.c2 }
                }
                Text { x: parent.width - 130; y: 18; text: modelData.emoji; font.pixelSize: 96 }
                Text { x: 28; y: 140; text: modelData.title; color: "white"; font.pixelSize: 38; font.bold: true; width: parent.width - 56; wrapMode: Text.WordWrap }
                Text { x: 28; y: 200; text: modelData.text; color: "#f1f5f9"; font.pixelSize: 26; lineHeight: 1.15 }
                Text { visible: index < 2; x: parent.width + 6; y: 150; text: "➜"; color: "#94a3b8"; font.pixelSize: 40 }
            }
        }
    }
}

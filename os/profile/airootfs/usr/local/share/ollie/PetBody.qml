// Ollie Pet's body (pet.qml and pet-desktop.qml): the Ollie cube with feet,
// eyes that blink and look where it walks, cheeks and a mouth for its mood.
// It grows: a baby (0-1 days old), a kid (2-5), then grown-up with a crown.
// A secret: a pet called Ollie is golden and sparkles. In the seasons
// (ollie-seasons) it wears a hat: a pumpkin, a snowflake, a party popper.
import QtQuick

Item {
    id: pet
    property string mood: "happy"   // happy, bored, hungry, sleepy, asleep
    property int dir: 1
    property bool stepping: false
    property real hopY: 0
    property int stage: 2           // 0 baby, 1 kid, 2 grown-up
    property bool golden: false
    property string season: ""     // halloween, winter, birthday
    readonly property string hat: ({ halloween: "🎃", winter: "❄️", birthday: "🎉" })[season] || ""
    readonly property color skin: golden ? "#fbbf24" : "#2dd4bf"
    readonly property color side: golden ? "#fde68a" : "#9eeee4"
    readonly property color dark: golden ? "#78350f" : "#0b4f4a"
    readonly property color feet: golden ? "#b45309" : "#0f766e"
    readonly property real grow: stage === 0 ? 0.62 : stage === 1 ? 0.82 : 1.0
    width: 130
    height: 130

    Item {
        id: all
        width: 130
        height: 130
        transform: Scale { origin.x: 65; origin.y: 130; xScale: pet.grow; yScale: pet.grow }
        y: -pet.hopY

        Rectangle { // shadow
            anchors.horizontalCenter: parent.horizontalCenter
            y: 124 + pet.hopY
            width: 100 - pet.hopY / 2; height: 14; radius: 7
            color: "#00000030"
        }
        Rectangle {
            x: 28; y: 114; width: 26; height: 16; radius: 6; color: pet.feet
            transform: Translate { y: pet.stepping ? -4 : 0 }
        }
        Rectangle {
            x: 76; y: 114; width: 26; height: 16; radius: 6; color: pet.feet
            transform: Translate { y: pet.stepping ? 0 : -4 }
        }
        Rectangle {
            id: body
            x: 10; y: 6; width: 110; height: 108; radius: pet.stage === 0 ? 34 : 22
            color: pet.skin
            border.color: pet.dark; border.width: 4
            Rectangle {
                x: 4; y: 4; width: parent.width - 8; height: 30; radius: 18
                color: "white"
                Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 12; color: "white" }
            }
            Rectangle {
                x: 4; y: 30; width: (parent.width - 8) / 2; height: parent.height - 34; radius: 14
                color: pet.side
            }
            Repeater {
                model: [30, 70]
                delegate: Item {
                    required property int modelData
                    // A baby's eyes are bigger.
                    x: modelData - (pet.stage === 0 ? 13 : 10); y: pet.stage === 0 ? 40 : 44
                    width: pet.stage === 0 ? 28 : 22; height: pet.stage === 0 ? 30 : 24
                    Rectangle {
                        visible: pet.mood !== "asleep" && !blink.closed
                        anchors.fill: parent; radius: width / 2; color: "white"; border.color: pet.dark; border.width: 2
                        Rectangle { x: parent.width / 2 - 5 + pet.dir * 3; y: 7; width: 10; height: parent.height / 2; radius: 5; color: "#111827"
                            Rectangle { x: 2; y: 2; width: 4; height: 4; radius: 2; color: "white" } }
                    }
                    Rectangle {
                        visible: pet.mood === "asleep" || blink.closed
                        y: parent.height / 2; width: parent.width; height: 4; radius: 2; color: pet.dark
                    }
                }
            }
            Rectangle { x: 14; y: 72; width: 14; height: 8; radius: 4; color: "#fda4af"; visible: pet.mood === "happy" }
            Rectangle { x: 82; y: 72; width: 14; height: 8; radius: 4; color: "#fda4af"; visible: pet.mood === "happy" }
            Canvas {
                x: 38; y: 74; width: 34; height: 20
                property string m: pet.mood
                property bool g: pet.golden
                onMChanged: requestPaint()
                onGChanged: requestPaint()
                onPaint: {
                    const c = getContext("2d");
                    c.reset();
                    c.strokeStyle = pet.dark; c.lineWidth = 4; c.lineCap = "round";
                    c.beginPath();
                    if (m === "happy") c.arc(17, 2, 12, 0.15 * Math.PI, 0.85 * Math.PI);
                    else if (m === "asleep") c.ellipse(12, 6, 8, 8);
                    else if (m === "hungry") c.ellipse(9, 2, 14, 14);
                    else c.arc(17, 20, 10, 1.2 * Math.PI, 1.8 * Math.PI);
                    c.stroke();
                }
            }
        }
        Text { // the season's hat
            visible: pet.hat !== ""
            x: 44; y: -34; text: pet.hat; font.pixelSize: 36
        }
        Text { // a grown-up's crown
            visible: pet.stage === 2 && pet.hat === ""
            x: 46; y: -30; text: "👑"; font.pixelSize: 34
        }
        Text { // a baby's curl
            visible: pet.stage === 0 && pet.hat === ""
            x: 56; y: -16; text: "🌱"; font.pixelSize: 24
        }
        Repeater { // a golden pet's sparkles
            model: pet.golden ? [[-6, 10, 0], [118, 30, 500], [100, -20, 1000], [4, 90, 1500]] : []
            delegate: Text {
                required property var modelData
                x: modelData[0]; y: modelData[1]; text: "✨"; font.pixelSize: 22
                SequentialAnimation on opacity { loops: Animation.Infinite
                    PauseAnimation { duration: modelData[2] }
                    NumberAnimation { from: 0.2; to: 1; duration: 700 } NumberAnimation { from: 1; to: 0.2; duration: 700 } }
            }
        }
        Text {
            visible: pet.mood === "asleep"
            x: 110; y: -10
            text: "💤"; font.pixelSize: 34
            SequentialAnimation on y { running: pet.mood === "asleep"; loops: Animation.Infinite
                NumberAnimation { to: -30; duration: 1200 } NumberAnimation { to: -10; duration: 1200 } }
        }
    }
    Item {
        id: blink
        property bool closed: false
        Timer { interval: 3200; running: pet.mood !== "asleep"; repeat: true; onTriggered: { blink.closed = true; openTimer.start(); } }
        Timer { id: openTimer; interval: 160; onTriggered: blink.closed = false }
    }
}

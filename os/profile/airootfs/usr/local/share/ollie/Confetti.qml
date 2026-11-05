// Confetti (Ollie Finder's and Welcome's secrets): burst() rains bright
// paper and stars over the whole window for a few seconds.
import QtQuick

Item {
    id: root
    anchors.fill: parent
    z: 1000
    function burst() {
        for (let i = 0; i < 90; i++)
            bits.append({
                bx: Math.random() * root.width, delay: Math.random() * 900, spin: (Math.random() - 0.5) * 900,
                drift: (Math.random() - 0.5) * 160, dur: 1800 + Math.random() * 1600,
                c: ["#f43f5e", "#facc15", "#22c55e", "#3b82f6", "#a855f7", "#14b8a6", "#fb923c"][i % 7],
                star: i % 9 === 0
            });
        clear.restart();
    }
    Timer { id: clear; interval: 4600; onTriggered: bits.clear() }
    ListModel { id: bits }
    Repeater {
        model: bits
        delegate: Item {
            id: bit
            required property real bx
            required property real delay
            required property real spin
            required property real drift
            required property real dur
            required property string c
            required property bool star
            x: bx; y: -30
            Rectangle { visible: !bit.star; width: 10; height: 16; radius: 2; color: bit.c }
            Text { visible: bit.star; text: "⭐"; font.pixelSize: 22 }
            SequentialAnimation {
                running: true
                PauseAnimation { duration: bit.delay }
                ParallelAnimation {
                    NumberAnimation { target: bit; property: "y"; to: root.height + 30; duration: bit.dur; easing.type: Easing.InQuad }
                    NumberAnimation { target: bit; property: "x"; to: bit.bx + bit.drift; duration: bit.dur }
                    NumberAnimation { target: bit; property: "rotation"; to: bit.spin; duration: bit.dur }
                }
            }
        }
    }
}

// Happy birthday! (ollie-seasons, on your birthday): a cake, confetti again
// and again, and Ollie singing along.
// Argument: name= (yours).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 760
    height: 520
    visible: true
    title: "Happy birthday!"
    flags: Qt.Window | Qt.WindowStaysOnTopHint

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: "#d946ef" }
            GradientStop { position: 1; color: "#4a044e" }
        }
    }
    ColumnLayout {
        anchors.centerIn: parent
        spacing: 14
        Label {
            text: "🎂"; font.pixelSize: 130; Layout.alignment: Qt.AlignHCenter
            SequentialAnimation on scale { loops: Animation.Infinite
                NumberAnimation { to: 1.12; duration: 500; easing.type: Easing.OutQuad }
                NumberAnimation { to: 1.0; duration: 500; easing.type: Easing.InQuad } }
        }
        Label { text: "Happy birthday" + (win.arg("name") ? ", " + win.arg("name") : "") + "!"; color: "white"; font.pixelSize: 42; font.bold: true; Layout.alignment: Qt.AlignHCenter }
        Label { text: "🎉 Have the best day ever! From Ollie and Ollie OS 🎈"; color: "#fce7f3"; font.pixelSize: 20; Layout.alignment: Qt.AlignHCenter }
        Button { text: "Thank you! 💖"; font.pixelSize: 18; highlighted: true; Layout.alignment: Qt.AlignHCenter; onClicked: Qt.quit() }
    }
    Confetti { id: confetti }
    Timer { interval: 2500; running: true; repeat: true; triggeredOnStart: true; onTriggered: confetti.burst() }
}

// Screen Time is over (ollie-timesup): covers the whole screen until
// tomorrow, or until a grown-up types the parent PIN and picks more time
// (said on the console: OLLIE-ACTION extra MINUTES). It cannot be closed.
// Arguments: pin= (the PIN's check sum), mode=bedtime (it's bedtime instead).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    visible: true
    visibility: Window.FullScreen
    flags: Qt.Window | Qt.FramelessWindowHint | Qt.WindowStaysOnTopHint
    title: "Screen Time"
    color: "#141833"
    onClosing: close => close.accepted = false

    function arg(name) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(name + "=") === 0)
                return a[i].substring(name.length + 1);
        return "";
    }
    readonly property string pinSum: arg("pin")

    function more(minutes) {
        if (pinSum !== "" && Qt.md5("ollie-os:" + pin.text) === pinSum) {
            console.log("OLLIE-ACTION extra " + minutes);
            Qt.quit();
        } else {
            wrong.visible = true;
            pin.text = "";
        }
    }

    Rectangle {
        anchors.fill: parent
        color: "#141833"
    }

    ColumnLayout {
        anchors.centerIn: parent
        spacing: 18
        Label { text: ""; font.pixelSize: 110; Layout.alignment: Qt.AlignHCenter }
        Label { text: win.arg("mode") === "bedtime" ? "Goodnight! It's bedtime" : "Screen time is over for today"; color: "white"; font.pixelSize: 44; font.bold: true; Layout.alignment: Qt.AlignHCenter }
        Label { text: win.arg("mode") === "bedtime" ? "Sleep well. See you in the morning." : "That's all for today. Come back tomorrow."; color: "#D8DBEA"; font.pixelSize: 24; Layout.alignment: Qt.AlignHCenter }

        ColumnLayout {
            Layout.alignment: Qt.AlignHCenter
            Layout.topMargin: 40
            spacing: 10
            visible: win.pinSum !== ""
            Label { text: win.arg("mode") === "bedtime" ? "Grown-up: type the parent PIN for a later bedtime tonight" : "Grown-up: type the parent PIN for more time"; color: "#A9AECA"; font.pixelSize: 16; Layout.alignment: Qt.AlignHCenter }
            TextField {
                id: pin
                Layout.alignment: Qt.AlignHCenter
                Layout.preferredWidth: 220
                placeholderText: "Parent PIN"
                echoMode: TextInput.Password
                inputMethodHints: Qt.ImhDigitsOnly
                horizontalAlignment: TextInput.AlignHCenter
                font.pixelSize: 20
            }
            RowLayout {
                Layout.alignment: Qt.AlignHCenter
                Button { text: "+15 minutes"; onClicked: win.more(15) }
                Button { text: "+30 minutes"; onClicked: win.more(30) }
                Button { text: "+1 hour"; onClicked: win.more(60) }
            }
            Label { id: wrong; text: "That is not the PIN."; color: "#F0A3A3"; visible: false; Layout.alignment: Qt.AlignHCenter }
        }
    }
}

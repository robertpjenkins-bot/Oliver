// Ollie OS Welcome (ollie-welcome): big buttons for what you'd do first.
// Each says what to open on the console (OLLIE-ACTION open NAME), and the
// tick box whether to open at every start (OLLIE-ACTION at-start on|off).
// A secret: the code ↑ ↑ ↓ ↓ ← → ← → B A makes a rainbow and confetti, and
// unlocks the Ollie Gold colours in Settings (OLLIE-ACTION secret konami).
// Arguments: atstart=on|off store=on|off version= (Ollie OS's, /etc/ollie-version).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 860
    height: 640
    visible: true
    title: "Welcome to Ollie OS"
    color: "#083c38"

    function arg(name) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(name + "=") === 0)
                return a[i].substring(name.length + 1);
        return "";
    }
    function act(what) { console.log("OLLIE-ACTION " + what) }

    readonly property var buttons: [
        { icon: "🕹️", label: "Games", hint: "All your games, drop in ROMs", act: "library" },
        { icon: "🎮", label: "Install games", hint: "The Steam store", act: "games" },
        { icon: "⛏️", label: "Minecraft", hint: "The Minecraft Launcher", act: "minecraft" },
        { icon: "🖼️", label: "Change background", hint: "Your name, or draw one", act: "background" },
        { icon: "🎨", label: "Colours & settings", hint: "Dark mode, colours, Screen Time", act: "settings" },
        { icon: "🔧", label: "Fix a problem", hint: "The Ollie OS Fixer", act: "fixer" },
        { icon: "⬆️", label: "Update Ollie OS", hint: "One click: everything new", act: "update" },
        { icon: "🔔", label: "New from biobro", hint: "Updates to install", act: "updates" }
    ].concat(arg("store") === "on" ? [{ icon: "🛍️", label: "App store", hint: "More apps (Discover)", act: "store" }] : [])

    // The secret code.
    readonly property var code: [Qt.Key_Up, Qt.Key_Up, Qt.Key_Down, Qt.Key_Down, Qt.Key_Left, Qt.Key_Right, Qt.Key_Left, Qt.Key_Right, Qt.Key_B, Qt.Key_A]
    property int codeAt: 0
    property bool rainbow: false
    function key(k) {
        codeAt = k === code[codeAt] ? codeAt + 1 : (k === code[0] ? 1 : 0);
        if (codeAt === code.length) {
            codeAt = 0; rainbow = true; confetti.burst();
            act("secret konami");
        }
    }
    Item { id: keys; focus: true; Keys.onPressed: (e) => win.key(e.key) }

    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: win.rainbow ? "#ef4444" : "#0f766e" }
            GradientStop { position: 0.25; color: win.rainbow ? "#f59e0b" : "#0d6b64" }
            GradientStop { position: 0.5; color: win.rainbow ? "#22c55e" : "#0b5e57" }
            GradientStop { position: 0.75; color: win.rainbow ? "#3b82f6" : "#0a4f49" }
            GradientStop { position: 1; color: win.rainbow ? "#a855f7" : "#083c38" }
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 28
        spacing: 18

        RowLayout {
            spacing: 16
            Image {
                source: "file:///usr/share/icons/hicolor/scalable/apps/ollie-os.svg"
                sourceSize.width: 72
                sourceSize.height: 72
            }
            ColumnLayout {
                spacing: 2
                Label { text: "Welcome to Ollie OS!"; color: "white"; font.pixelSize: 34; font.bold: true }
                Label { text: win.rainbow ? "🌈 You found the secret code! Ollie Gold is in Settings now. ✨" : "What do you want to do first?"; color: "#ccfbf1"; font.pixelSize: 18 }
            }
        }

        GridLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            columns: 3
            rowSpacing: 14
            columnSpacing: 14
            Repeater {
                model: win.buttons
                delegate: Rectangle {
                    required property var modelData
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    Layout.minimumHeight: 120
                    radius: 18
                    color: area.pressed ? "#5eead4" : (area.containsMouse ? "#2dd4bf" : "#14b8a6")
                    border.color: "#ccfbf1"
                    border.width: area.activeFocus ? 3 : 0
                    Column {
                        anchors.centerIn: parent
                        spacing: 4
                        Label { text: modelData.icon; font.pixelSize: 40; anchors.horizontalCenter: parent.horizontalCenter }
                        Label { text: modelData.label; color: "#042f2e"; font.pixelSize: 19; font.bold: true; anchors.horizontalCenter: parent.horizontalCenter }
                        Label { text: modelData.hint; color: "#064e48"; font.pixelSize: 13; anchors.horizontalCenter: parent.horizontalCenter }
                    }
                    MouseArea {
                        id: area
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: win.act("open " + modelData.act)
                    }
                }
            }
        }

        RowLayout {
            CheckBox {
                id: atStart
                checked: win.arg("atstart") === "on"
                onToggled: win.act("at-start " + (checked ? "on" : "off"))
                text: "Show this when Ollie OS starts"
                palette.windowText: "white"
            }
            Item { Layout.fillWidth: true }
            Label { text: (win.arg("version") ? "Ollie OS " + win.arg("version") + "   ·   " : "") + "Find it again in the apps menu: Ollie OS Welcome"; color: "#99f6e4"; font.pixelSize: 13 }
            Button { text: "Close"; onClicked: Qt.quit() }
        }
    }
    Confetti { id: confetti }
}

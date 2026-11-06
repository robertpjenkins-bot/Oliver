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
    width: 1040
    height: 640
    visible: true
    title: "Welcome to Ollie OS"
    color: "#F4EFE6"

    function arg(name) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(name + "=") === 0)
                return a[i].substring(name.length + 1);
        return "";
    }
    function act(what) { console.log("OLLIE-ACTION " + what) }
    // An app's icon: its picture (hicolor PNG), or its drawing (SVG).
    function iconOf(n) { return n.indexOf("/") === 0 ? "file://" + n : "file:///usr/share/icons/hicolor/128x128/apps/" + n + ".png" }

    readonly property var buttons: [
        { icon: "ollie-gamehub", label: "Game Hub", hint: "Big screen, for a controller", act: "gamehub" },
        { icon: "ollie-games", label: "Games", hint: "All your games in one place", act: "library" },
        { icon: "/usr/share/icons/hicolor/256x256/apps/steam.png", label: "Install games", hint: "The Steam store", act: "games" },
        { icon: "ollie-appcenter", label: "App Center", hint: "Free games and apps", act: "appcenter" },
        { icon: "/usr/share/icons/hicolor/scalable/apps/ollie-minecraft.svg", label: "Minecraft", hint: "The Minecraft Launcher", act: "minecraft" },
        { icon: "ollie-wallpaper-maker", label: "Change background", hint: "Your name, or draw one", act: "background" },
        { icon: "ollie-settings", label: "Settings", hint: "Colours, dark mode, Screen Time", act: "settings" },
        { icon: "ollie-fixer", label: "Fix a problem", hint: "The Ollie OS Fixer", act: "fixer" },
        { icon: "ollie-os-update", label: "Update Ollie OS", hint: "Get everything new", act: "update" },
        { icon: "ollie-update-checker", label: "New from biobro", hint: "Updates to install", act: "updates" }
    ].concat(arg("store") === "on" ? [{ icon: "/usr/share/icons/hicolor/scalable/apps/plasmadiscover.svg", label: "App store", hint: "More apps (Discover)", act: "store" }] : [])

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

    // The secret's rainbow: a band of flat stripes along the top.
    Row {
        visible: win.rainbow
        anchors.top: parent.top
        Repeater {
            model: ["#C8473D", "#E07A2F", "#D9A021", "#4F8A45", "#336B9E", "#6A4E9C"]
            Rectangle { required property string modelData; width: win.width / 6; height: 10; color: modelData }
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 32
        spacing: 20

        RowLayout {
            spacing: 18
            Image {
                source: "file:///usr/share/icons/hicolor/scalable/apps/ollie-os.svg"
                sourceSize.width: 64
                sourceSize.height: 64
            }
            ColumnLayout {
                spacing: 2
                Label { text: "Welcome to Ollie OS"; color: "#22252B"; font.family: "Noto Serif"; font.pixelSize: 34; font.bold: true }
                Label {
                    text: win.rainbow ? "You found the secret code. Ollie Gold is in Settings now."
                        : (({ halloween: "Happy Halloween. ", winter: "Happy winter. ", birthday: "Happy birthday. " })[win.arg("season")] || "") + "What do you want to do first?"
                    color: "#5B5F68"; font.pixelSize: 17
                }
            }
        }

        GridLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            columns: 4
            rowSpacing: 12
            columnSpacing: 12
            Repeater {
                model: win.buttons
                delegate: Rectangle {
                    required property var modelData
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    Layout.minimumHeight: 110
                    radius: 10
                    color: area.pressed ? "#EAE3D6" : "white"
                    border.color: area.containsMouse ? "#257A78" : "#DDD5C7"
                    border.width: area.containsMouse ? 2 : 1
                    RowLayout {
                        anchors.fill: parent
                        anchors.margins: 16
                        spacing: 14
                        Image {
                            source: win.iconOf(modelData.icon)
                            // (One that isn't there: the App Center's.)
                            onStatusChanged: if (status === Image.Error) source = win.iconOf("ollie-appcenter")
                            sourceSize: Qt.size(56, 56)
                            Layout.preferredWidth: 56; Layout.preferredHeight: 56
                        }
                        ColumnLayout {
                            spacing: 2
                            Layout.fillWidth: true
                            Label { text: modelData.label; color: "#22252B"; font.pixelSize: 18; font.bold: true; Layout.fillWidth: true; wrapMode: Text.WordWrap }
                            Label { text: modelData.hint; color: "#6B6F78"; font.pixelSize: 13; Layout.fillWidth: true; wrapMode: Text.WordWrap }
                        }
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
            }
            // Its words in the window's own ink (the box's own text follows dark mode).
            Label {
                text: "Show this when Ollie OS starts"; color: "#22252B"; font.pixelSize: 14
                MouseArea { anchors.fill: parent; onClicked: atStart.toggle(), atStart.toggled() }
            }
            Item { Layout.fillWidth: true }
            Label { text: (win.arg("version") ? "Ollie OS " + win.arg("version") + "  ·  " : "") + "It's in the apps menu too: Ollie OS Welcome"; color: "#6B6F78"; font.pixelSize: 13 }
            Button { text: "Close"; onClicked: Qt.quit() }
        }
    }
    Confetti { id: confetti }
}

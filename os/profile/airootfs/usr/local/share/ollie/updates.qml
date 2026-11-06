// Ollie OS Updates (ollie-update-checker --now): the latest updates from
// biobro, newest first, each with its date, what is new, its files and an
// Install button (or "Installed"). Says on the console what to do:
// OLLIE-ACTION install TAG, or OLLIE-ACTION refresh (check again).
// Arguments: ok=yes|no (whether GitHub could be reached), list=JSON.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 720
    height: 640
    visible: true
    title: "Ollie OS Updates"
    color: palette.window

    function arg(name) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(name + "=") === 0)
                return a[i].substring(name.length + 1);
        return "";
    }
    readonly property bool reached: arg("ok") === "yes"
    property var updates: {
        try { return JSON.parse(arg("list") || "[]"); } catch (e) { return []; }
    }
    property var installing: ({})
    readonly property int newCount: updates.filter(u => !u.installed).length


    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 22
        spacing: 14

        RowLayout {
            spacing: 14
            Image { source: "file:///usr/share/icons/hicolor/64x64/apps/ollie-update-checker.png"; sourceSize: Qt.size(48, 48) }
            ColumnLayout {
                spacing: 2
                Layout.fillWidth: true
                Label { text: "Latest updates"; color: palette.text; font.pixelSize: 30; font.bold: true }
                Label {
                    color: palette.placeholderText
                    font.pixelSize: 16
                    text: !win.reached ? "Could not check: is the internet connected?"
                        : win.updates.length === 0 ? "No updates yet."
                        : win.newCount === 0 ? "Everything is installed: Ollie OS is up to date."
                        : win.newCount === 1 ? "1 new update to install." : win.newCount + " new updates to install."
                }
            }
            Button {
                text: "Check again"
                onClicked: { console.log("OLLIE-ACTION refresh"); Qt.quit(); }
            }
        }

        ScrollView {
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            contentWidth: availableWidth

            ColumnLayout {
                width: parent.width
                spacing: 12

                Repeater {
                    model: win.updates
                    delegate: Rectangle {
                        required property var modelData
                        required property int index
                        Layout.fillWidth: true
                        implicitHeight: card.implicitHeight + 28
                        radius: 6
                        color: palette.base
                        border.color: !modelData.installed ? "#257A78" : palette.mid
                        border.width: 2

                        ColumnLayout {
                            id: card
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.top: parent.top
                            anchors.margins: 14
                            spacing: 6

                            RowLayout {
                                Layout.fillWidth: true
                                Label {
                                    text: (index === 0 ? "" : "") + modelData.title
                                    color: palette.text
                                    font.pixelSize: 20
                                    font.bold: true
                                    elide: Text.ElideRight
                                    Layout.fillWidth: true
                                }
                                Label {
                                    text: modelData.date
                                    color: palette.placeholderText
                                    font.pixelSize: 13
                                }
                            }
                            Label {
                                text: modelData.body
                                color: palette.text
                                font.pixelSize: 15
                                wrapMode: Text.WordWrap
                                Layout.fillWidth: true
                            }
                            Label {
                                visible: modelData.files.length > 0
                                text: "Files: " + modelData.files.join(", ")
                                color: palette.placeholderText
                                font.pixelSize: 12
                                wrapMode: Text.WordWrap
                                Layout.fillWidth: true
                            }
                            RowLayout {
                                Item { Layout.fillWidth: true }
                                Label {
                                    visible: modelData.installed || win.installing[modelData.tag] === true
                                    text: modelData.installed ? "✔ Installed" : "Installing… (its window shows how far it is)"
                                    color: palette.text
                                    font.pixelSize: 15
                                    font.bold: true
                                }
                                Button {
                                    visible: !modelData.installed && win.installing[modelData.tag] !== true
                                    text: "Install"
                                    highlighted: true
                                    onClicked: {
                                        console.log("OLLIE-ACTION install " + modelData.tag);
                                        const s = Object.assign({}, win.installing);
                                        s[modelData.tag] = true;
                                        win.installing = s;
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        RowLayout {
            Label {
                text: "New updates also come as a notification by themselves (the Update Checker)."
                color: palette.placeholderText
                font.pixelSize: 13
                Layout.fillWidth: true
                wrapMode: Text.WordWrap
            }
            Button { text: "Close"; onClicked: Qt.quit() }
        }
    }
}

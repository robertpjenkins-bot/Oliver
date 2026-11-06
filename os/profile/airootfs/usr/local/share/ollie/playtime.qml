// Play Time (ollie-playtime): how long you played this week, day by day
// (a bar chart, each game its own colour) and game by game.
// Argument: data=FILE ({"days": [{date, day, games: {NAME: minutes}}], "total": minutes ever}).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 920
    height: 680
    visible: true
    title: "Play Time"
    color: "#FAFAF7" // a report on paper

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    property var days: []
    property int total: 0
    Component.onCompleted: {
        const x = new XMLHttpRequest();
        x.open("GET", "file://" + arg("data"), false);
        try { x.send(); const d = JSON.parse(x.responseText); days = d.days; total = d.total; } catch (e) {}
    }
    readonly property var colours: ["#257A78", "#E07A2F", "#6A4E9C", "#336B9E", "#C8473D", "#4F8A45", "#D0587E", "#D9A021"]
    // Every game this week, most played first: [name, minutes].
    readonly property var games: {
        const t = {};
        for (const d of days) for (const g in d.games) t[g] = (t[g] || 0) + d.games[g];
        return Object.keys(t).map(g => [g, t[g]]).sort((a, b) => b[1] - a[1]);
    }
    function colourOf(name) { const i = games.findIndex(g => g[0] === name); return colours[(i < 0 ? 0 : i) % colours.length]; }
    readonly property int week: games.reduce((s, g) => s + g[1], 0)
    readonly property int most: Math.max(60, ...days.map(d => Object.values(d.games).reduce((s, m) => s + m, 0)))
    function hm(m) { return m < 60 ? m + " min" : Math.floor(m / 60) + " h " + (m % 60 ? (m % 60) + " min" : ""); }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 24
        spacing: 16
        RowLayout {
            Label { text: "Play Time"; color: "#22252B"; font.pixelSize: 32; font.bold: true }
            Item { Layout.fillWidth: true }
            ColumnLayout {
                spacing: 0
                Label { text: "This week: " + win.hm(win.week); color: "#257A78"; font.pixelSize: 20; font.bold: true; Layout.alignment: Qt.AlignRight }
                Label { text: "All time: " + win.hm(win.total); color: "#6B6F78"; font.pixelSize: 14; Layout.alignment: Qt.AlignRight }
            }
        }

        // The week, a bar a day.
        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 300
            radius: 4
            color: "white"
            border.color: "#E2DED6"
            RowLayout {
                anchors.fill: parent
                anchors.margins: 18
                spacing: 14
                Repeater {
                    model: win.days
                    delegate: ColumnLayout {
                        id: day
                        required property var modelData
                        required property int index
                        readonly property int mins: Object.values(modelData.games).reduce((s, m) => s + m, 0)
                        Layout.fillWidth: true
                        Layout.preferredWidth: 1
                        Layout.fillHeight: true
                        spacing: 6
                        Label { text: day.mins ? win.hm(day.mins) : ""; color: "#44474F"; font.pixelSize: 12; Layout.alignment: Qt.AlignHCenter }
                        Item {
                            id: barArea
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            Column {
                                anchors.bottom: parent.bottom
                                anchors.horizontalCenter: parent.horizontalCenter
                                width: Math.min(parent.width, 70)
                                Repeater {
                                    model: Object.keys(day.modelData.games).sort((a, b) => win.games.findIndex(g => g[0] === b) - win.games.findIndex(g => g[0] === a))
                                    delegate: Rectangle {
                                        required property string modelData
                                        width: parent.width
                                        height: Math.max(2, barArea.height * (day.modelData.games[modelData] / win.most))
                                        color: win.colourOf(modelData)
                                        radius: 4
                                    }
                                }
                            }
                        }
                        Label {
                            text: day.index === 6 ? "Today" : day.modelData.day
                            color: day.index === 6 ? "#257A78" : "#6B6F78"; font.pixelSize: 15; font.bold: day.index === 6
                            Layout.alignment: Qt.AlignHCenter
                        }
                    }
                }
            }
        }

        // Game by game.
        Label { text: "Games this week"; color: "#22252B"; font.pixelSize: 20; font.bold: true }
        ListView {
            id: list
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            spacing: 8
            model: win.games
            delegate: RowLayout {
                required property var modelData
                width: list.width
                spacing: 12
                Rectangle { width: 16; height: 16; radius: 8; color: win.colourOf(modelData[0]) }
                Label { text: modelData[0]; color: "#22252B"; font.pixelSize: 17; Layout.preferredWidth: 260; elide: Text.ElideRight }
                Rectangle {
                    Layout.fillWidth: true
                    height: 14; radius: 2; color: "#ECE8E1"
                    Rectangle { width: parent.width * modelData[1] / Math.max(1, win.games[0][1]); height: parent.height; radius: 7; color: win.colourOf(modelData[0]) }
                }
                Label { text: win.hm(modelData[1]); color: "#44474F"; font.pixelSize: 15; Layout.preferredWidth: 100; horizontalAlignment: Text.AlignRight }
            }
            Label {
                anchors.centerIn: parent
                visible: win.games.length === 0
                text: "No games played this week yet."
                color: "#6B6F78"; font.pixelSize: 18
            }
        }
    }
}

// Ollie Finder (ollie-finder): type, and the apps and files with those
// letters in their names show up at once (apps first, then the best
// matches). Enter opens the first; arrows move; a click opens; "folder"
// shows where a file is. Says on the console what to open:
// OLLIE-ACTION open app|file PATH, or show PATH.
// Secrets: type "ollie" (unlocks the Ollie Gold colours: OLLIE-ACTION secret
// gold), "party" (confetti) or "biobro" (who made Ollie OS).
// Arguments: index=FILE (lines: kind TAB name TAB path), home=DIR.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 760
    height: 600
    visible: true
    title: "Ollie Finder"
    color: palette.window // the system's own colours (Settings: light or dark)

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    readonly property string home: arg("home")
    property var items: []
    property var results: []
    property int current: 0

    Component.onCompleted: {
        const x = new XMLHttpRequest();
        x.open("GET", "file://" + arg("index"), false);
        x.send();
        const out = [];
        for (const line of x.responseText.split("\n")) {
            const p = line.split("\t");
            if (p.length === 3) out.push({ kind: p[0], name: p[1], path: p[2], low: p[1].toLowerCase() });
        }
        items = out;
        query.forceActiveFocus();
    }

    // A small tag saying what it is: [word, colour].
    function tag(it) {
        if (it.kind === "secret") return ["SECRET", "#D9A021"];
        if (it.kind === "app") return ["APP", "#336B9E"];
        if (it.kind === "folder") return ["FOLDER", "#7E5634"];
        const e = it.low.split(".").pop();
        if (["png", "jpg", "jpeg", "gif", "webp", "svg", "bmp"].includes(e)) return ["PICTURE", "#6A4E9C"];
        if (["mp3", "ogg", "wav", "flac", "m4a"].includes(e)) return ["MUSIC", "#8E4573"];
        if (["mp4", "mkv", "webm", "mov", "avi"].includes(e)) return ["VIDEO", "#C8473D"];
        if (["appimage", "exe", "msi", "sh"].includes(e)) return ["PROGRAM", "#4C5363"];
        if (["iso", "nes", "sfc", "gba", "gb", "n64", "z64", "nds"].includes(e)) return ["GAME", "#4F8A45"];
        if (["zip", "tar", "gz", "xz", "7z", "zst"].includes(e)) return ["ZIP", "#E07A2F"];
        return ["FILE", "#6B6F78"];
    }
    function search(t) {
        t = t.trim().toLowerCase();
        current = 0;
        if (t === "") { results = []; return; }
        const found = [];
        const secret = secrets[t];
        if (secret) found.push({ it: secret, score: -1 });
        for (const it of items) {
            const at = it.low.indexOf(t);
            if (at < 0) continue;
            // Apps first, then names that start with it, then shorter names.
            found.push({ it: it, score: (it.kind === "app" ? 0 : 1000) + (at === 0 ? 0 : 300) + it.name.length });
            if (found.length > 4000) break;
        }
        found.sort((a, b) => a.score - b.score);
        results = found.slice(0, 60).map(f => f.it);
    }
    readonly property var secrets: ({
        "ollie": { kind: "secret", name: "Secret: the Ollie Gold colours!", path: "gold", low: "ollie" },
        "party": { kind: "secret", name: "Secret: party time!", path: "party", low: "party" },
        "biobro": { kind: "secret", name: "Secret: who made Ollie OS?", path: "biobro", low: "biobro" }
    })
    property string secretText: ""
    function open(it) {
        if (it.kind === "secret") {
            if (it.path === "party") confetti.burst();
            if (it.path === "gold") {
                secretText = "You found Ollie Gold. Your colours are gold now (and in Settings).";
                console.log("OLLIE-ACTION secret gold");
            } else if (it.path === "party") {
                secretText = "Party time.";
            } else {
                secretText = "Ollie OS was made by biobro. Hi from biobro.";
            }
            return;
        }
        console.log("OLLIE-ACTION open " + (it.kind === "app" ? "app " : "file ") + it.path);
        Qt.quit();
    }
    function where(it) {
        if (it.kind === "secret") return "A secret. Press Enter.";
        if (it.kind === "app") return "App";
        const p = it.path.indexOf(home) === 0 ? "Home" + it.path.substring(home.length) : it.path;
        return p.substring(0, p.length - it.name.length - 1) || "Home";
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 14

        Label { text: "Ollie Finder"; font.pixelSize: 26; font.bold: true }
        TextField {
            id: query
            Layout.fillWidth: true
            font.pixelSize: 24
            placeholderText: "Find apps and files"
            onTextChanged: { win.secretText = ""; win.search(text) }
            onAccepted: if (win.results.length) win.open(win.results[win.current])
            Keys.onDownPressed: win.current = Math.min(win.results.length - 1, win.current + 1)
            Keys.onUpPressed: win.current = Math.max(0, win.current - 1)
            Keys.onEscapePressed: Qt.quit()
        }
        Label {
            opacity: 0.7
            font.pixelSize: 14
            text: win.secretText !== "" ? win.secretText
                : query.text.trim() === "" ? "Looking in " + win.items.length + " apps, files and folders."
                : win.results.length === 0 ? "Nothing found with \"" + query.text.trim() + "\"."
                : win.results.length + (win.results.length >= 60 ? "+" : "") + " found. Enter opens the first one."
        }
        ListView {
            id: list
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            spacing: 2
            model: win.results
            currentIndex: win.current
            delegate: Rectangle {
                required property var modelData
                required property int index
                width: list.width
                height: 54
                radius: 4
                color: index === win.current ? palette.highlight : (row.containsMouse ? palette.alternateBase : "transparent")
                readonly property color ink: index === win.current ? palette.highlightedText : palette.text
                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 14
                    anchors.rightMargin: 10
                    spacing: 12
                    Rectangle {
                        Layout.preferredWidth: 72; Layout.preferredHeight: 22; radius: 3
                        color: win.tag(modelData)[1]
                        Label { anchors.centerIn: parent; text: win.tag(modelData)[0]; color: "white"; font.pixelSize: 11; font.bold: true; font.letterSpacing: 0.5 }
                    }
                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 0
                        Label { text: modelData.name; color: parent.parent.parent.ink; font.pixelSize: 16; font.bold: true; elide: Text.ElideMiddle; Layout.fillWidth: true }
                        Label { text: win.where(modelData); color: parent.parent.parent.ink; opacity: 0.7; font.pixelSize: 12; elide: Text.ElideMiddle; Layout.fillWidth: true }
                    }
                    Button {
                        visible: modelData.kind === "file" || modelData.kind === "folder"
                        text: "Show folder"
                        icon.name: "folder-open"
                        flat: true
                        font.pixelSize: 13
                        onClicked: { console.log("OLLIE-ACTION show " + modelData.path); Qt.quit(); }
                    }
                }
                MouseArea {
                    id: row
                    anchors.fill: parent
                    anchors.rightMargin: 120
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: win.open(modelData)
                }
            }
        }
    }
    Confetti { id: confetti }
}

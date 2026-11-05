// Download Tracker (ollie-download-tracker): how many times each file of a
// project's GitHub releases was downloaded. The account is always biobro's
// (GitHub login robertpjenkins-bot); type the project, or tap one.
// For tests: --api URL (instead of GitHub's), --shot FILE (saves a picture
// of the window after loading, then quits).
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 900
    height: 760
    visible: true
    title: "Download Tracker"

    readonly property string owner: "robertpjenkins-bot"
    readonly property string ownerName: "biobro"
    readonly property var args: Qt.application.arguments
    function arg(name) { const i = args.indexOf(name); return i >= 0 && i + 1 < args.length ? args[i + 1] : "" }
    readonly property string api: arg("--api") || "https://api.github.com"

    property var releases: []
    property var repos: []
    property int total: 0
    property string status: ""
    property bool busy: false

    function get(path, done) {
        const x = new XMLHttpRequest();
        x.onreadystatechange = function () {
            if (x.readyState !== XMLHttpRequest.DONE) return;
            let data = null;
            try { data = JSON.parse(x.responseText); } catch (e) {}
            done(x.status, data);
        };
        x.open("GET", api + path);
        x.setRequestHeader("Accept", "application/vnd.github+json");
        x.send();
    }
    function sizeText(b) {
        return b >= 1e9 ? (b / 1e9).toFixed(2) + " GB" : b >= 1e6 ? (b / 1e6).toFixed(1) + " MB" : Math.max(1, Math.round(b / 1e3)) + " KB";
    }
    function show(repo) {
        repo = repo.trim();
        if (!repo) return;
        repoField.text = repo;
        busy = true;
        status = "Looking at " + repo + "...";
        get("/repos/" + owner + "/" + encodeURIComponent(repo) + "/releases?per_page=100", function (code, data) {
            busy = false;
            if (code === 200 && Array.isArray(data)) {
                let sum = 0;
                const list = data.map(function (r) {
                    const files = (r.assets || []).map(function (a) { return { name: a.name, size: a.size, count: a.download_count }; })
                        .sort(function (a, b) { return b.count - a.count; });
                    const n = files.reduce(function (s, f) { return s + f.count; }, 0);
                    sum += n;
                    return { name: r.name || r.tag_name, date: (r.published_at || "").slice(0, 10), files: files, count: n };
                });
                releases = list;
                total = sum;
                status = list.length ? "" : repo + " has no releases yet, so nothing to download.";
            } else {
                releases = [];
                total = 0;
                status = code === 404 ? "There is no public project called \"" + repo + "\" on " + ownerName + "'s GitHub."
                    : code === 403 ? "GitHub says to wait a little (it allows 60 looks an hour). Try again soon."
                    : code === 0 ? "Could not reach GitHub. Is the internet connected?"
                    : "GitHub answered " + code + ". Try again.";
            }
            if (arg("--shot")) shotTimer.start();
        });
    }

    Component.onCompleted: {
        get("/users/" + owner + "/repos?per_page=100&sort=pushed", function (code, data) {
            if (code === 200 && Array.isArray(data)) repos = data.map(function (r) { return r.name; });
        });
        show("Ollie-OS");
    }
    Timer {
        id: shotTimer
        interval: 800
        onTriggered: page.grabToImage(function (r) { r.saveToFile(win.arg("--shot")); Qt.quit(); })
    }

    Rectangle {
        id: page
        anchors.fill: parent
        color: win.palette.window
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 12

        Label { text: win.ownerName + "'s downloads"; font.pixelSize: 26; font.bold: true }
        RowLayout {
            Layout.fillWidth: true
            TextField {
                id: repoField
                Layout.fillWidth: true
                placeholderText: "Project name, for example Ollie-OS"
                onAccepted: win.show(text)
            }
            Button { text: "Show"; highlighted: true; enabled: !win.busy; onClicked: win.show(repoField.text) }
            Button { text: "Refresh"; enabled: !win.busy && repoField.text !== ""; onClicked: win.show(repoField.text) }
        }
        Flow {
            Layout.fillWidth: true
            spacing: 6
            visible: win.repos.length > 0
            Repeater {
                model: win.repos
                delegate: Button {
                    required property string modelData
                    text: modelData
                    flat: modelData !== repoField.text
                    onClicked: win.show(modelData)
                }
            }
        }

        RowLayout {
            visible: win.releases.length > 0
            spacing: 10
            Label { text: win.total.toLocaleString(Qt.locale(), "f", 0); font.pixelSize: 40; font.bold: true }
            Label { text: "downloads in all"; font.pixelSize: 18; Layout.alignment: Qt.AlignBottom; bottomPadding: 6 }
        }
        Label { visible: win.status !== ""; text: win.status; wrapMode: Text.Wrap; Layout.fillWidth: true; font.pixelSize: 16 }

        ScrollView {
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            ColumnLayout {
                width: win.width - 32
                spacing: 18
                Repeater {
                    model: win.releases
                    delegate: ColumnLayout {
                        required property var modelData
                        Layout.fillWidth: true
                        spacing: 4
                        RowLayout {
                            Layout.fillWidth: true
                            Label { text: modelData.name; font.pixelSize: 18; font.bold: true; Layout.fillWidth: true; elide: Text.ElideRight }
                            Label { text: modelData.count + " downloads  ·  " + modelData.date; opacity: 0.75 }
                        }
                        Label { visible: modelData.files.length === 0; text: "No files in it."; opacity: 0.7 }
                        Repeater {
                            model: modelData.files
                            delegate: RowLayout {
                                required property var modelData
                                Layout.fillWidth: true
                                Label { text: modelData.name; Layout.fillWidth: true; elide: Text.ElideMiddle }
                                Label { text: win.sizeText(modelData.size); opacity: 0.7; Layout.preferredWidth: 90; horizontalAlignment: Text.AlignRight }
                                Label { text: modelData.count; font.bold: true; Layout.preferredWidth: 70; horizontalAlignment: Text.AlignRight }
                            }
                        }
                    }
                }
            }
        }
    }
    }
}

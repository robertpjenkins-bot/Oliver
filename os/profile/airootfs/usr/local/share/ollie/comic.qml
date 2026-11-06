// Comic Maker (ollie-comic): a comic strip of 1 to 4 panels. Each panel gets
// a scene (sky, space, forest, city, beach, night, under the sea); click a
// panel, then add characters, speech and thought bubbles and sound words
// ("POW!"), and drag them where they go. Save puts it in Pictures/Comics.
// Says on the console: OLLIE-ACTION saved FILE.
// Argument: out=DIR.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: win
    width: 1200
    height: 760
    visible: true
    title: "Comic Maker"
    color: "#F7F3E8" // newsprint

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }

    property int panels: 3
    property int panel: 0                   // the panel things are added to
    property var scenes: ["sky", "forest", "space", "city"]
    property var things: []                 // {id, panel, kind: char|say|think|pow, text, x, y, size, flip}
    property int nextId: 1
    property int selected: -1               // a thing's id
    readonly property var sceneList: [
        { k: "sky", t: "Sky", top: "#A9D3E8", bottom: "#B7D7A0", deco: "☁️" },
        { k: "forest", t: "Forest", top: "#CFE0A8", bottom: "#4F7A3E", deco: "🌲" },
        { k: "space", t: "Space", top: "#1B2140", bottom: "#2E2A55", deco: "⭐" },
        { k: "city", t: "City", top: "#F0D9A0", bottom: "#8A8178", deco: "🏢" },
        { k: "beach", t: "Beach", top: "#9ED8E0", bottom: "#EBD7A2", deco: "🌴" },
        { k: "night", t: "Night", top: "#24224A", bottom: "#161A2C", deco: "🌙" },
        { k: "sea", t: "Under the sea", top: "#3E8FA8", bottom: "#24496E", deco: "🫧" }
    ]
    function scene(k) { return sceneList.find(s => s.k === k) || sceneList[0] }
    function thing(id) { return things.find(t => t.id === id) }
    function add(kind, text) {
        // Side by side, not on top of each other.
        const n = things.filter(t => t.panel === panel && (t.kind === "char") === (kind === "char")).length;
        const t = { id: nextId++, panel: panel, kind: kind, text: text, x: 0.08 + (n % 3) * 0.3, y: kind === "char" ? 0.5 : 0.06 + (n % 2) * 0.16, size: 1, flip: n % 2 === 1 };
        things = things.concat([t]);
        selected = t.id;
        if (kind === "say" || kind === "think") { words.text = text; words.forceActiveFocus(); words.selectAll(); }
    }
    function change(id, f) {
        things = things.map(t => { if (t.id !== id) return t; const c = Object.assign({}, t); f(c); return c; });
    }
    function remove(id) { things = things.filter(t => t.id !== id); selected = -1; }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 14
        spacing: 14

        // --- The tools ---
        ColumnLayout {
            Layout.preferredWidth: 310
            Layout.maximumWidth: 310
            Layout.fillHeight: true
            spacing: 8
            Label { text: "COMIC MAKER"; font.pixelSize: 28; font.weight: Font.Black; font.letterSpacing: 1; color: "#111111" }
            Label { text: "Panels"; font.bold: true }
            RowLayout {
                Repeater {
                    model: [1, 2, 3, 4]
                    delegate: Button { required property int modelData; text: modelData; highlighted: win.panels === modelData
                        onClicked: { win.panels = modelData; win.panel = Math.min(win.panel, modelData - 1); } }
                }
            }
            Label { text: "Scene for panel " + (win.panel + 1); font.bold: true }
            Flow {
                Layout.preferredWidth: 310
                spacing: 4
                Repeater {
                    model: win.sceneList
                    delegate: Button { required property var modelData; text: modelData.t; highlighted: win.scenes[win.panel] === modelData.k
                        onClicked: { const s = win.scenes.slice(); s[win.panel] = modelData.k; win.scenes = s; } }
                }
            }
            Label { text: "Add to panel " + (win.panel + 1); font.bold: true }
            Flow {
                Layout.preferredWidth: 310
                spacing: 2
                Repeater {
                    model: ["🧒", "👧", "🧑‍🚀", "🦸", "🧙", "🦖", "🐱", "🐶", "🤖", "👾", "👻", "🐉", "🦄", "🧟", "🚗", "🛸"]
                    delegate: Button { required property string modelData; text: modelData; font.pixelSize: 22; flat: true; onClicked: win.add("char", modelData) }
                }
            }
            RowLayout {
                Button { text: "Speech"; onClicked: win.add("say", "Hello!") }
                Button { text: "Thought"; onClicked: win.add("think", "Hmm...") }
            }
            Flow {
                Layout.preferredWidth: 310
                spacing: 4
                Repeater {
                    model: ["POW!", "BOOM!", "ZAP!", "WOW!", "OOPS!"]
                    delegate: Button { required property string modelData; text: modelData; onClicked: win.add("pow", modelData) }
                }
            }

            // The thing picked.
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: chosen.implicitHeight + 20
                radius: 0
                color: "white"
                border.color: "#111111"; border.width: 2
                visible: win.selected >= 0
                ColumnLayout {
                    id: chosen
                    anchors.fill: parent
                    anchors.margins: 10
                    Label { text: "The one you picked:"; font.bold: true }
                    TextField {
                        id: words
                        Layout.fillWidth: true
                        visible: { const t = win.thing(win.selected); return t !== undefined && t.kind !== "char"; }
                        placeholderText: "What do they say?"
                        onTextEdited: win.change(win.selected, t => t.text = text)
                    }
                    RowLayout {
                        Button { text: "Smaller"; onClicked: win.change(win.selected, t => t.size = Math.max(0.5, t.size - 0.2)) }
                        Button { text: "Bigger"; onClicked: win.change(win.selected, t => t.size = Math.min(3, t.size + 0.2)) }
                        Button { text: "Flip"; onClicked: win.change(win.selected, t => t.flip = !t.flip) }
                        Button { text: "Delete"; onClicked: win.remove(win.selected) }
                    }
                }
            }
            Item { Layout.fillHeight: true }
            Button {
                text: "Save my comic"
                highlighted: true
                font.pixelSize: 18
                Layout.fillWidth: true
                onClicked: {
                    win.selected = -1;
                    Qt.callLater(() => strip.grabToImage(r => {
                        const f = win.arg("out") + "/comic-" + Date.now() + ".png";
                        if (r.saveToFile(f)) { win.act("trophy count comic"); win.act("saved " + f); savedNote.text = "Saved in Pictures, Comics!"; }
                    }));
                }
            }
            Label { id: savedNote; color: "#3E7B36" }
        }

        // --- The strip ---
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Rectangle {
                id: strip
                anchors.centerIn: parent
                readonly property bool grid: win.panels === 4
                width: grid ? Math.min(parent.width, parent.height * 1.2) : Math.min(parent.width, parent.height * win.panels * 0.9)
                height: grid ? width / 1.2 : width / (win.panels * 0.9)
                color: "white"
                border.color: "#111111"
                border.width: 6
                Grid {
                    anchors.fill: parent
                    anchors.margins: 12
                    columns: strip.grid ? 2 : win.panels
                    spacing: 12
                    Repeater {
                        model: win.panels
                        delegate: Rectangle {
                            id: box
                            required property int index
                            readonly property var sc: win.scene(win.scenes[index] || "sky")
                            width: (strip.width - 24 - (strip.grid ? 12 : 12 * (win.panels - 1))) / (strip.grid ? 2 : win.panels)
                            height: (strip.height - 24 - (strip.grid ? 12 : 0)) / (strip.grid ? 2 : 1)
                            clip: true
                            border.color: win.panel === index ? "#C8473D" : "#111111"
                            border.width: win.panel === index ? 6 : 4
                            color: box.sc.top // flat colours, like a printed comic: sky, then ground
                            Rectangle { anchors.bottom: parent.bottom; anchors.left: parent.left; anchors.right: parent.right; anchors.margins: parent.border.width; height: parent.height * 0.3; color: box.sc.bottom }
                            MouseArea { anchors.fill: parent; onClicked: { win.panel = index; win.selected = -1; } }
                            Text { x: parent.width * 0.72; y: parent.height * 0.06; text: box.sc.deco; font.pixelSize: parent.height * 0.14; opacity: 0.85 }
                            Text { x: parent.width * 0.06; y: parent.height * 0.66; text: box.sc.deco; font.pixelSize: parent.height * 0.18; opacity: 0.7 }

                            Repeater {
                                model: win.things.filter(t => t.panel === box.index)
                                delegate: Item {
                                    id: it
                                    required property var modelData
                                    readonly property real u: box.height / 100
                                    x: modelData.x * box.width
                                    y: modelData.y * box.height
                                    width: shapeItem.width
                                    height: shapeItem.height
                                    Item {
                                        id: shapeItem
                                        width: it.modelData.kind === "char" ? 34 * it.u * it.modelData.size : bubbleText.width + 30
                                        height: it.modelData.kind === "char" ? 34 * it.u * it.modelData.size : bubbleText.height + 24
                                        Text {
                                            visible: it.modelData.kind === "char"
                                            anchors.centerIn: parent
                                            text: it.modelData.text
                                            font.pixelSize: 30 * it.u * it.modelData.size
                                            transform: Scale { origin.x: shapeItem.width / 2; xScale: it.modelData.flip ? -1 : 1 }
                                        }
                                        Rectangle { // speech / thought bubble
                                            visible: it.modelData.kind === "say" || it.modelData.kind === "think"
                                            anchors.fill: parent
                                            radius: it.modelData.kind === "think" ? height / 2 : 14
                                            color: "white"; border.color: "#111111"; border.width: 3
                                            Rectangle { // the tail
                                                visible: it.modelData.kind === "say"
                                                width: 18; height: 18; rotation: 45; color: "white"
                                                border.color: "#111111"; border.width: 3
                                                x: it.modelData.flip ? parent.width - 40 : 22; y: parent.height - 10
                                                z: -1
                                            }
                                            Repeater { // thought dots
                                                model: it.modelData.kind === "think" ? 2 : 0
                                                delegate: Rectangle { required property int index; width: 14 - index * 5; height: width; radius: width / 2
                                                    color: "white"; border.color: "#111111"; border.width: 2
                                                    x: 24 + index * 14; y: shapeItem.height + 4 + index * 10 }
                                            }
                                        }
                                        Text {
                                            id: bubbleText
                                            visible: it.modelData.kind !== "char"
                                            anchors.centerIn: parent
                                            text: it.modelData.text
                                            font.pixelSize: (it.modelData.kind === "pow" ? 9 : 5) * it.u * it.modelData.size
                                            font.bold: true
                                            font.family: "Comic Neue, Comic Sans MS, sans-serif"
                                            color: it.modelData.kind === "pow" ? "#facc15" : "#111111"
                                            style: it.modelData.kind === "pow" ? Text.Outline : Text.Normal
                                            styleColor: "#b91c1c"
                                            rotation: it.modelData.kind === "pow" ? -8 : 0
                                            width: it.modelData.kind === "pow" ? implicitWidth : Math.min(implicitWidth, box.width * 0.6)
                                            wrapMode: Text.WordWrap
                                            horizontalAlignment: Text.AlignHCenter
                                        }
                                    }
                                    Rectangle {
                                        anchors.fill: parent; anchors.margins: -4
                                        color: "transparent"; border.color: "#C8473D"; border.width: 3; radius: 0
                                        visible: win.selected === it.modelData.id
                                    }
                                    MouseArea {
                                        anchors.fill: parent
                                        drag.target: it
                                        cursorShape: Qt.OpenHandCursor
                                        onPressed: { win.selected = it.modelData.id; win.panel = box.index; if (it.modelData.kind !== "char") words.text = it.modelData.text; }
                                        onReleased: { const id = it.modelData.id, nx = it.x / box.width, ny = it.y / box.height; win.change(id, t => { t.x = nx; t.y = ny; }); }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

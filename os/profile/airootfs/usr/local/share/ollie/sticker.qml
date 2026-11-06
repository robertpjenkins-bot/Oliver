// Sticker Maker (ollie-sticker): make a sticker from a picture, a drawing or
// an emoji, in a shape (circle, square, heart, star) with an outline and a
// caption; it is saved in Pictures/Stickers. Then drag it to where it goes
// on the desktop: the stickers are drawn into the desktop's background
// (the background without stickers is kept, so they can all come off).
// Says on the console: OLLIE-ACTION wallpaper FILE | restore.
// Arguments: wall=FILE (the background now), out=DIR (Pictures/Stickers),
// data=DIR (where the backgrounds with stickers go), config=FILE,
// screen=WxH.
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Dialogs
import QtCore

ApplicationWindow {
    id: win
    width: 1100
    height: 720
    visible: true
    title: "Sticker Maker"
    color: "#EEE6D8" // craft paper

    function arg(n) {
        const a = Qt.application.arguments;
        for (let i = 0; i < a.length; i++)
            if (a[i].indexOf(n + "=") === 0)
                return a[i].substring(n.length + 1);
        return "";
    }
    function act(w) { console.log("OLLIE-ACTION " + w) }

    Settings {
        id: saved
        location: win.arg("config") ? "file://" + win.arg("config") : ""
        property string base: ""          // the background without stickers
        property string placed: "[]"      // [{file, x, y, size}] (x, y, size: parts of the screen)
    }
    readonly property string wall: arg("wall")
    readonly property string baseWall: saved.base !== "" && wall.indexOf("/ollie/stickers/") >= 0 ? saved.base : wall
    readonly property real screenW: parseInt(arg("screen").split("x")[0]) || Math.round(Screen.width * Screen.devicePixelRatio) || 1920
    readonly property real screenH: parseInt(arg("screen").split("x")[1]) || Math.round(Screen.height * Screen.devicePixelRatio) || 1080
    property var placedList: JSON.parse(saved.placed || "[]")

    property string mode: "make"       // make, place
    property string source: "emoji"    // picture, draw, emoji
    property string picture: ""
    property string emoji: "🦖"
    property string shape: "circle"
    property string outline: "#ffffff"
    property var strokes: []
    property string ink: "#ef4444"
    property string made: ""           // the sticker file just made

    function repaint() { sticker.requestPaint() }
    function saveSticker(thenPlace) {
        const f = win.arg("out") + "/sticker-" + Date.now() + ".png";
        art.grabToImage(r => {
            if (!r.saveToFile(f)) return;
            win.made = f;
            if (thenPlace) { win.mode = "place"; newOne.x = place.width * 0.4; newOne.y = place.height * 0.3; }
            else { savedNote.text = "Saved in Pictures, Stickers!"; }
        });
    }
    onSourceChanged: repaint()
    onPictureChanged: { sticker.loadImage("file://" + picture); repaint(); }
    onEmojiChanged: repaint()
    onShapeChanged: repaint()
    onOutlineChanged: repaint()

    FileDialog {
        id: pick
        title: "Pick a picture for the sticker"
        nameFilters: ["Pictures (*.png *.jpg *.jpeg *.webp *.bmp *.gif)"]
        currentFolder: StandardPaths.writableLocation(StandardPaths.PicturesLocation)
        onAccepted: { win.picture = decodeURIComponent(selectedFile.toString().replace("file://", "")); win.source = "picture"; }
    }

    // The sticker's shape, as a path, in a box of size s.
    function shapePath(c, s, grow) {
        const m = 24 - grow, w = s - 2 * m;
        c.beginPath();
        if (shape === "circle") c.ellipse(m, m, w, w);
        else if (shape === "square") c.roundedRect(m, m, w, w, 48, 48);
        else if (shape === "heart") {
            const x = s / 2, top = m + w * 0.28;
            c.moveTo(x, s - m);
            c.bezierCurveTo(m - w * 0.05, top + w * 0.35, m + w * 0.02, m - w * 0.05, x, top);
            c.bezierCurveTo(s - m - w * 0.02, m - w * 0.05, s - m + w * 0.05, top + w * 0.35, x, s - m);
        } else {
            for (let i = 0; i < 10; i++) {
                const r = i % 2 === 0 ? w / 2 : w / 4.6, a = -Math.PI / 2 + i * Math.PI / 5;
                const px = s / 2 + r * Math.cos(a), py = s / 2 + r * Math.sin(a) + w * 0.03;
                if (i === 0) c.moveTo(px, py); else c.lineTo(px, py);
            }
        }
        c.closePath();
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 18
        visible: win.mode === "make"

        // The sticker.
        ColumnLayout {
            spacing: 10
            Label { text: "Sticker Maker"; font.pixelSize: 30; font.bold: true; color: "#22252B" }
            Rectangle {
                width: 440; height: 440; radius: 20
                color: "#f5f5f4"
                // A checkerboard: see-through around the sticker.
                Grid {
                    anchors.fill: parent; anchors.margins: 0; columns: 11; clip: true
                    Repeater { model: 121; delegate: Rectangle { width: 40; height: 40; color: (index % 11 + Math.floor(index / 11)) % 2 ? "#e7e5e4" : "#fafaf9" } }
                }
                Item {
                id: art
                anchors.centerIn: parent
                width: 400; height: 400
                Canvas {
                    id: sticker
                    anchors.fill: parent
                    onImageLoaded: requestPaint()
                    onPaint: {
                        const c = getContext("2d");
                        c.reset();
                        if (win.outline !== "none") {
                            win.shapePath(c, width, 14);
                            c.fillStyle = win.outline; c.fill();
                        }
                        c.save();
                        win.shapePath(c, width, 0);
                        c.clip();
                        c.fillStyle = win.source === "draw" ? "#ffffff" : "#fde68a";
                        c.fillRect(0, 0, width, height);
                        if (win.source === "picture" && win.picture && isImageLoaded("file://" + win.picture)) {
                            const img = c.createImageData("file://" + win.picture);
                            const iw = img.width, ih = img.height, k = Math.max(width / iw, height / ih);
                            c.drawImage("file://" + win.picture, (width - iw * k) / 2, (height - ih * k) / 2, iw * k, ih * k);
                        } else if (win.source === "draw") {
                            c.lineCap = "round"; c.lineJoin = "round";
                            for (const s of win.strokes) {
                                c.strokeStyle = s.color; c.lineWidth = 14;
                                c.beginPath();
                                for (let i = 0; i < s.pts.length; i++) i ? c.lineTo(s.pts[i].x, s.pts[i].y) : c.moveTo(s.pts[i].x, s.pts[i].y);
                                c.stroke();
                            }
                        }
                        c.restore();
                    }
                    MouseArea {
                        anchors.fill: parent
                        enabled: win.source === "draw"
                        onPressed: m => { win.strokes = win.strokes.concat([{ color: win.ink, pts: [{ x: m.x, y: m.y }] }]); }
                        onPositionChanged: m => { const s = win.strokes; s[s.length - 1].pts.push({ x: m.x, y: m.y }); win.strokes = s; sticker.requestPaint(); }
                    }
                }
                Text { // the emoji
                    visible: win.source === "emoji"
                    anchors.centerIn: parent
                    anchors.verticalCenterOffset: win.shape === "heart" ? -10 : win.shape === "star" ? 12 : 0
                    text: win.emoji
                    font.pixelSize: win.shape === "star" ? 150 : 210
                }
                Text { // the words
                    anchors.horizontalCenter: parent.horizontalCenter
                    anchors.bottom: parent.bottom; anchors.bottomMargin: 26
                    text: caption.text.trim()
                    font.pixelSize: 46; font.bold: true
                    color: "white"; style: Text.Outline; styleColor: "#1f2937"
                }
                }
            }
            Label { text: win.source === "draw" ? "Draw on it with the mouse!" : ""; color: "#a21caf" }
        }

        // The choices.
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 12
            Label { text: "1. What is on it?"; font.bold: true; font.pixelSize: 17; color: "#701a75" }
            RowLayout {
                Button { text: "Emoji"; highlighted: win.source === "emoji"; onClicked: win.source = "emoji" }
                Button { text: "A picture"; highlighted: win.source === "picture"; onClicked: pick.open() }
                Button { text: "Draw"; highlighted: win.source === "draw"; onClicked: win.source = "draw" }
            }
            Flow {
                visible: win.source === "emoji"
                Layout.fillWidth: true
                spacing: 4
                Repeater {
                    model: ["🦖", "🐱", "🐶", "🦄", "🐸", "🐼", "🦊", "🐙", "🚀", "⚽", "🎮", "🍕", "🍩", "🌈", "⭐", "🔥", "💎", "👑", "😎", "🤖", "👾", "🎸", "🍦", "🌵"]
                    delegate: Button { required property string modelData; text: modelData; font.pixelSize: 22; flat: win.emoji !== modelData; onClicked: win.emoji = modelData }
                }
            }
            RowLayout {
                visible: win.source === "draw"
                Repeater {
                    model: ["#ef4444", "#f97316", "#facc15", "#22c55e", "#3b82f6", "#a855f7", "#ec4899", "#111827"]
                    delegate: Rectangle { required property string modelData; width: 34; height: 34; radius: 17; color: modelData
                        border.width: win.ink === modelData ? 4 : 1; border.color: "#701a75"
                        MouseArea { anchors.fill: parent; onClicked: win.ink = parent.modelData } }
                }
                Button { text: "Undo"; onClicked: { win.strokes = win.strokes.slice(0, -1); win.repaint(); } }
                Button { text: "Clear"; onClicked: { win.strokes = []; win.repaint(); } }
            }
            Label { text: "2. Shape"; font.bold: true; font.pixelSize: 17; color: "#701a75" }
            RowLayout {
                Repeater {
                    model: [{ k: "circle", t: "Circle" }, { k: "square", t: "Square" }, { k: "heart", t: "Heart" }, { k: "star", t: "Star" }]
                    delegate: Button { required property var modelData; text: modelData.t; highlighted: win.shape === modelData.k; onClicked: win.shape = modelData.k }
                }
            }
            Label { text: "3. Outline"; font.bold: true; font.pixelSize: 17; color: "#701a75" }
            RowLayout {
                Repeater {
                    model: [{ k: "#ffffff", t: "White" }, { k: "#111827", t: "Black" }, { k: "#facc15", t: "Gold" }, { k: "#ec4899", t: "Pink" }, { k: "none", t: "None" }]
                    delegate: Button { required property var modelData; text: modelData.t; highlighted: win.outline === modelData.k; onClicked: win.outline = modelData.k }
                }
            }
            Label { text: "4. Words on it (if you like)"; font.bold: true; font.pixelSize: 17; color: "#701a75" }
            TextField { id: caption; Layout.preferredWidth: 300; maximumLength: 14; placeholderText: "For example: COOL!"; onTextChanged: win.repaint() }
            Item { Layout.fillHeight: true }
            RowLayout {
                Button {
                    text: "Save the sticker"
                    onClicked: win.saveSticker(false)
                }
                Button {
                    text: "Put it on my desktop →"
                    highlighted: true
                    font.pixelSize: 17
                    onClicked: win.saveSticker(true)
                }
            }
            Label { id: savedNote; color: "#15803d" }
        }
    }

    // --- Where it goes on the desktop ---
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 12
        visible: win.mode === "place"
        Label { text: "Drag your sticker to where it goes on the desktop"; font.pixelSize: 24; font.bold: true; color: "#22252B" }
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Rectangle {
                id: place
                anchors.centerIn: parent
                width: Math.min(parent.width, parent.height * win.screenW / win.screenH)
                height: width * win.screenH / win.screenW
                color: "#0f766e"
                clip: true
                Image { anchors.fill: parent; source: win.baseWall ? "file://" + win.baseWall : ""; fillMode: Image.PreserveAspectCrop; asynchronous: false }
                Repeater {
                    model: win.placedList
                    delegate: Image {
                        required property var modelData
                        source: "file://" + modelData.file
                        x: modelData.x * place.width; y: modelData.y * place.height
                        width: modelData.size * place.width; height: width
                    }
                }
                Image {
                    id: newOne
                    visible: win.mode === "place"
                    source: win.made ? "file://" + win.made : ""
                    width: size.value * place.width; height: width
                    MouseArea { anchors.fill: parent; drag.target: parent; cursorShape: Qt.OpenHandCursor }
                }
            }
        }
        RowLayout {
            Label { text: "Size:" }
            Slider { id: size; from: 0.06; to: 0.4; value: 0.16; Layout.preferredWidth: 260 }
            Item { Layout.fillWidth: true }
            Button { text: "← Back"; onClicked: win.mode = "make" }
            Button {
                text: "Take all stickers off"
                visible: win.placedList.length > 0
                onClicked: { saved.placed = "[]"; win.placedList = []; win.act("restore " + win.baseWall); }
            }
            Button {
                text: "Put it here"
                highlighted: true
                font.pixelSize: 17
                onClicked: {
                    if (saved.base === "" || win.wall.indexOf("/ollie/stickers/") < 0) saved.base = win.baseWall;
                    const list = win.placedList.concat([{ file: win.made, x: newOne.x / place.width, y: newOne.y / place.height, size: size.value }]);
                    win.placedList = list;
                    saved.placed = JSON.stringify(list);
                    newOne.visible = false;
                    Qt.callLater(() => place.grabToImage(r => {
                        const f = win.arg("data") + "/wallpaper-" + Date.now() + ".png";
                        if (r.saveToFile(f)) { win.act("trophy count sticker"); win.act("wallpaper " + f); }
                        newOne.visible = true;
                        win.mode = "make";
                    }, Qt.size(win.screenW, win.screenH)));
                }
            }
        }
    }
}

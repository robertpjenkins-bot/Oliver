import QtQuick
import QtQuick.Layouts
import org.kde.plasma.plasmoid
import org.kde.plasma.core as PlasmaCore
import org.kde.plasma.components as PlasmaComponents
import org.kde.kirigami as Kirigami

// A search box that lives directly in the panel. Enter opens the results in
// the default browser (Google Chrome); something that looks like a web
// address is opened as-is instead of being searched for.
PlasmoidItem {
    id: root

    preferredRepresentation: fullRepresentation

    function open(query) {
        const q = query.trim();
        if (q.length === 0)
            return;
        let url;
        if (/^https?:\/\//i.test(q))
            url = q;
        else if (/^[^\s\/]+\.[a-z]{2,}(\/\S*)?$/i.test(q))
            url = "https://" + q;
        else
            url = "https://www.google.com/search?q=" + encodeURIComponent(q);
        Qt.openUrlExternally(url);
    }

    fullRepresentation: PlasmaComponents.TextField {
        id: field

        Layout.minimumWidth: Kirigami.Units.gridUnit * 12
        Layout.preferredWidth: Kirigami.Units.gridUnit * 22
        Layout.alignment: Qt.AlignVCenter

        placeholderText: "Search Google"
        leftPadding: icon.width + Kirigami.Units.largeSpacing * 2

        Kirigami.Icon {
            id: icon
            source: "search"
            width: Kirigami.Units.iconSizes.small
            height: width
            anchors.left: parent.left
            anchors.leftMargin: Kirigami.Units.largeSpacing
            anchors.verticalCenter: parent.verticalCenter
        }

        // A panel only passes keystrokes to a widget that asks for them.
        onActiveFocusChanged: Plasmoid.status = activeFocus
            ? PlasmaCore.Types.AcceptingInputStatus
            : PlasmaCore.Types.ActiveStatus

        onAccepted: {
            root.open(text);
            text = "";
            focus = false;
        }
        Keys.onEscapePressed: {
            text = "";
            focus = false;
        }
    }
}

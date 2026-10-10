// The Annie OS desktop, applied the first time a user logs in.
//
// One bar along the bottom, apps on the left and search on the right:
//   [unicorn menu] [pinned apps ...........] [search box] [tray] [clock]
//
// The Command / Windows key opens the menu too (/etc/xdg/kwinrc).

var wallpaper = "/usr/share/wallpapers/AnnieOS/";

desktopsArray = desktopsForActivity(currentActivity());
for (var i = 0; i < desktopsArray.length; i++) {
    var d = desktopsArray[i];
    d.wallpaperPlugin = "org.kde.image";
    d.currentConfigGroup = ["Wallpaper", "org.kde.image", "General"];
    d.writeConfig("Image", wallpaper);
}

var panel = new Panel;
panel.location = "bottom";
panel.height = Math.round(gridUnit * 2.6);

var menu = panel.addWidget("org.kde.plasma.kickoff");
menu.currentConfigGroup = ["General"];
menu.writeConfig("icon", "annie-os");

var apps = panel.addWidget("org.kde.plasma.icontasks");
apps.currentConfigGroup = ["General"];
apps.writeConfig("launchers", [
    "applications:firefox.desktop",
    "applications:org.kde.dolphin.desktop",
    "applications:steam.desktop",
    "applications:annie-paint.desktop",
    "applications:annie-disk-writer.desktop",
    "applications:annie-fixer.desktop",
    "applications:annie-dc.desktop",
    "applications:org.kde.konsole.desktop",
    "applications:systemsettings.desktop"
]);

panel.addWidget("org.kde.plasma.panelspacer");
panel.addWidget("org.annieos.googlesearch");
panel.addWidget("org.kde.plasma.marginsseparator");
panel.addWidget("org.kde.plasma.systemtray");
panel.addWidget("org.kde.plasma.digitalclock");
panel.addWidget("org.kde.plasma.showdesktop");

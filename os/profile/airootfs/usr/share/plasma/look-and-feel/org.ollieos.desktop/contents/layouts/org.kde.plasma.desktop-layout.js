// The Ollie OS desktop, applied the first time a user logs in.
//
// Fun Fact of the Day on the desktop, top right; one bar along the bottom:
//   [cube menu] [Google search ...........] [pinned apps] [tray] [clock]

var wallpaper = "/usr/share/wallpapers/OllieOS/";

desktopsArray = desktopsForActivity(currentActivity());
for (var i = 0; i < desktopsArray.length; i++) {
    var d = desktopsArray[i];
    d.wallpaperPlugin = "org.kde.image";
    d.currentConfigGroup = ["Wallpaper", "org.kde.image", "General"];
    d.writeConfig("Image", wallpaper);
    // Fun Fact of the Day, in the top right corner.
    var g = screenGeometry(d.screen);
    d.addWidget("org.ollieos.funfact", g.width - 440, 40, 400, 210);
}

var panel = new Panel;
panel.location = "bottom";
panel.height = Math.round(gridUnit * 2.6);

var menu = panel.addWidget("org.kde.plasma.kickoff");
menu.currentConfigGroup = ["General"];
menu.writeConfig("icon", "ollie-os");

panel.addWidget("org.ollieos.googlesearch");
panel.addWidget("org.kde.plasma.panelspacer");

var apps = panel.addWidget("org.kde.plasma.icontasks");
apps.currentConfigGroup = ["General"];
apps.writeConfig("launchers", [
    "applications:firefox.desktop",
    "applications:steam.desktop",
    "applications:org.kde.dolphin.desktop",
    "applications:oliver.desktop",
    "applications:org.kde.konsole.desktop",
    "applications:ollie-dc.desktop",
    "applications:systemsettings.desktop"
]);

panel.addWidget("org.kde.plasma.marginsseparator");
panel.addWidget("org.kde.plasma.systemtray");
panel.addWidget("org.kde.plasma.digitalclock");
panel.addWidget("org.kde.plasma.showdesktop");

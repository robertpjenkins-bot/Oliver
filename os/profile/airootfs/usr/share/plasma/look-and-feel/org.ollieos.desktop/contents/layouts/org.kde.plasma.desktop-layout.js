// The Ollie OS desktop, applied the first time a user logs in.
//
// One bar along the bottom:
//   [cube menu] [Google search ...........] [pinned apps] [tray] [clock]

var wallpaper = "/usr/share/wallpapers/OllieOS/";

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

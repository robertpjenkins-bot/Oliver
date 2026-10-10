package design

// App is one program the designer can choose to pre-install.
//
// Packages are Arch Linux package names from the official repositories.
// Desktop and Exec are both recorded because desktop-file names drift between
// releases: the desktop shell tries the .desktop id first and falls back to
// finding whichever installed application runs Exec.
type App struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Packages    []string `json:"packages"`
	Desktop     string   `json:"desktop"`
	Exec        string   `json:"exec"`
	Icon        string   `json:"icon"`
	Emoji       string   `json:"emoji"`
	Color       string   `json:"color"`
	// Base apps are part of every OS and cannot be unticked.
	Base bool `json:"base,omitempty"`
	// Multilib apps need Arch's 32-bit repository switched on.
	Multilib bool `json:"multilib,omitempty"`
}

// Categories lists the catalog's categories in display order.
var Categories = []string{"Built in", "Internet", "Media", "Graphics", "Office", "Learning", "Coding", "Games", "System"}

// Catalog is every app OS Maker knows how to install.
var Catalog = []App{
	{ID: "files", Name: "Files", Category: "Built in", Description: "Browse your folders", Packages: []string{"thunar"}, Desktop: "thunar.desktop", Exec: "thunar", Icon: "system-file-manager", Emoji: "📁", Color: "#e8a33d", Base: true},
	{ID: "terminal", Name: "Terminal", Category: "Built in", Description: "Type commands", Packages: []string{"xfce4-terminal"}, Desktop: "xfce4-terminal.desktop", Exec: "xfce4-terminal", Icon: "utilities-terminal", Emoji: "⌨️", Color: "#2d3436", Base: true},
	{ID: "settings", Name: "Settings", Category: "Built in", Description: "Change how the OS works", Packages: []string{"xfce4-settings"}, Desktop: "xfce-settings-manager.desktop", Exec: "xfce4-settings-manager", Icon: "preferences-system", Emoji: "⚙️", Color: "#636e72", Base: true},
	{ID: "texteditor", Name: "Text Editor", Category: "Built in", Description: "Write notes and text files", Packages: []string{"mousepad"}, Desktop: "org.xfce.mousepad.desktop", Exec: "mousepad", Icon: "accessories-text-editor", Emoji: "📝", Color: "#0984e3", Base: true},
	{ID: "images", Name: "Image Viewer", Category: "Built in", Description: "Look at pictures", Packages: []string{"ristretto"}, Desktop: "org.xfce.ristretto.desktop", Exec: "ristretto", Icon: "image-viewer", Emoji: "🖼️", Color: "#00b894", Base: true},

	{ID: "firefox", Name: "Firefox", Category: "Internet", Description: "Web browser", Packages: []string{"firefox"}, Desktop: "firefox.desktop", Exec: "firefox", Icon: "firefox", Emoji: "🦊", Color: "#ff7139"},
	{ID: "chromium", Name: "Chromium", Category: "Internet", Description: "Web browser (Chrome's open-source twin)", Packages: []string{"chromium"}, Desktop: "chromium.desktop", Exec: "chromium", Icon: "chromium", Emoji: "🌐", Color: "#4285f4"},
	{ID: "thunderbird", Name: "Thunderbird", Category: "Internet", Description: "Email", Packages: []string{"thunderbird"}, Desktop: "org.mozilla.Thunderbird.desktop", Exec: "thunderbird", Icon: "thunderbird", Emoji: "✉️", Color: "#0a84ff"},
	{ID: "discord", Name: "Discord", Category: "Internet", Description: "Chat with friends", Packages: []string{"discord"}, Desktop: "discord.desktop", Exec: "discord", Icon: "discord", Emoji: "💬", Color: "#5865f2"},
	{ID: "telegram", Name: "Telegram", Category: "Internet", Description: "Messaging", Packages: []string{"telegram-desktop"}, Desktop: "org.telegram.desktop.desktop", Exec: "Telegram", Icon: "org.telegram.desktop", Emoji: "✈️", Color: "#27a7e7"},
	{ID: "transmission", Name: "Transmission", Category: "Internet", Description: "Torrent downloads", Packages: []string{"transmission-gtk"}, Desktop: "transmission-gtk.desktop", Exec: "transmission-gtk", Icon: "transmission", Emoji: "⬇️", Color: "#b2382f"},

	{ID: "vlc", Name: "VLC", Category: "Media", Description: "Plays any video or music", Packages: []string{"vlc", "vlc-plugins-all"}, Desktop: "vlc.desktop", Exec: "vlc", Icon: "vlc", Emoji: "🎬", Color: "#ff8800"},
	{ID: "spotify", Name: "Spotify", Category: "Media", Description: "Music streaming", Packages: []string{"spotify-launcher"}, Desktop: "spotify-launcher.desktop", Exec: "spotify-launcher", Icon: "spotify-launcher", Emoji: "🎵", Color: "#1db954"},
	{ID: "audacity", Name: "Audacity", Category: "Media", Description: "Record and edit sound", Packages: []string{"audacity"}, Desktop: "audacity.desktop", Exec: "audacity", Icon: "audacity", Emoji: "🎙️", Color: "#0000cc"},
	{ID: "obs", Name: "OBS Studio", Category: "Media", Description: "Record your screen and stream", Packages: []string{"obs-studio"}, Desktop: "com.obsproject.Studio.desktop", Exec: "obs", Icon: "com.obsproject.Studio", Emoji: "🔴", Color: "#302e31"},
	{ID: "kdenlive", Name: "Kdenlive", Category: "Media", Description: "Edit videos", Packages: []string{"kdenlive"}, Desktop: "org.kde.kdenlive.desktop", Exec: "kdenlive", Icon: "kdenlive", Emoji: "🎞️", Color: "#527eb2"},

	{ID: "gimp", Name: "GIMP", Category: "Graphics", Description: "Edit photos", Packages: []string{"gimp"}, Desktop: "gimp.desktop", Exec: "gimp", Icon: "gimp", Emoji: "🖌️", Color: "#5c5543"},
	{ID: "krita", Name: "Krita", Category: "Graphics", Description: "Digital painting", Packages: []string{"krita"}, Desktop: "org.kde.krita.desktop", Exec: "krita", Icon: "krita", Emoji: "🎨", Color: "#3babff"},
	{ID: "inkscape", Name: "Inkscape", Category: "Graphics", Description: "Draw shapes and logos", Packages: []string{"inkscape"}, Desktop: "org.inkscape.Inkscape.desktop", Exec: "inkscape", Icon: "org.inkscape.Inkscape", Emoji: "✒️", Color: "#000000"},
	{ID: "blender", Name: "Blender", Category: "Graphics", Description: "Make 3D models and animations", Packages: []string{"blender"}, Desktop: "blender.desktop", Exec: "blender", Icon: "blender", Emoji: "🧊", Color: "#ea7600"},

	{ID: "libreoffice", Name: "LibreOffice", Category: "Office", Description: "Documents, spreadsheets, slides", Packages: []string{"libreoffice-fresh"}, Desktop: "libreoffice-startcenter.desktop", Exec: "libreoffice", Icon: "libreoffice-startcenter", Emoji: "📄", Color: "#18a303"},
	{ID: "evince", Name: "PDF Viewer", Category: "Office", Description: "Read PDF files", Packages: []string{"evince"}, Desktop: "org.gnome.Evince.desktop", Exec: "evince", Icon: "org.gnome.Evince", Emoji: "📕", Color: "#c0392b"},
	{ID: "calculator", Name: "Calculator", Category: "Office", Description: "Do maths", Packages: []string{"galculator"}, Desktop: "galculator.desktop", Exec: "galculator", Icon: "accessories-calculator", Emoji: "🧮", Color: "#6c5ce7"},
	{ID: "keepassxc", Name: "KeePassXC", Category: "Office", Description: "Keep your passwords safe", Packages: []string{"keepassxc"}, Desktop: "org.keepassxc.KeePassXC.desktop", Exec: "keepassxc", Icon: "keepassxc", Emoji: "🔑", Color: "#6cac4d"},

	{ID: "vscode", Name: "Code - OSS", Category: "Coding", Description: "Code editor (open-source VS Code)", Packages: []string{"code"}, Desktop: "code-oss.desktop", Exec: "code-oss", Icon: "com.visualstudio.code.oss", Emoji: "🧑‍💻", Color: "#007acc"},
	{ID: "geany", Name: "Geany", Category: "Coding", Description: "Small, fast code editor", Packages: []string{"geany"}, Desktop: "geany.desktop", Exec: "geany", Icon: "geany", Emoji: "🫖", Color: "#d4a017"},
	{ID: "python", Name: "Python", Category: "Coding", Description: "Python programming language", Packages: []string{"python", "python-pip"}, Exec: "python3", Icon: "text-x-python", Emoji: "🐍", Color: "#3776ab"},
	{ID: "git", Name: "Git", Category: "Coding", Description: "Version control", Packages: []string{"git"}, Exec: "git", Icon: "git", Emoji: "🌿", Color: "#f05032"},

	{ID: "steam", Name: "Steam", Category: "Games", Description: "Buy and play PC games", Packages: []string{"steam", "lib32-mesa"}, Desktop: "steam.desktop", Exec: "steam", Icon: "steam", Emoji: "🎮", Color: "#1b2838", Multilib: true},
	{ID: "prism", Name: "Minecraft (Prism Launcher)", Category: "Games", Description: "Play Minecraft: Java Edition with your Microsoft account", Packages: []string{"prismlauncher"}, Desktop: "org.prismlauncher.PrismLauncher.desktop", Exec: "prismlauncher", Icon: "org.prismlauncher.PrismLauncher", Emoji: "⛏️", Color: "#62b47a"},
	{ID: "supertuxkart", Name: "SuperTuxKart", Category: "Games", Description: "Kart racing", Packages: []string{"supertuxkart"}, Desktop: "supertuxkart.desktop", Exec: "supertuxkart", Icon: "supertuxkart", Emoji: "🏎️", Color: "#e84393"},
	{ID: "supertux", Name: "SuperTux", Category: "Games", Description: "Jump-and-run platformer", Packages: []string{"supertux"}, Desktop: "supertux2.desktop", Exec: "supertux2", Icon: "supertux", Emoji: "🐧", Color: "#fdcb6e"},
	{ID: "luanti", Name: "Luanti", Category: "Games", Description: "Block-building sandbox (was Minetest)", Packages: []string{"luanti"}, Desktop: "org.luanti.luanti.desktop", Exec: "luanti", Icon: "org.luanti.luanti", Emoji: "🧱", Color: "#8e6e53"},
	{ID: "0ad", Name: "0 A.D.", Category: "Games", Description: "Ancient-history strategy", Packages: []string{"0ad"}, Desktop: "0ad.desktop", Exec: "0ad", Icon: "0ad", Emoji: "🏛️", Color: "#a67c52"},
	{ID: "solitaire", Name: "Solitaire", Category: "Games", Description: "Card games", Packages: []string{"aisleriot"}, Desktop: "sol.desktop", Exec: "sol", Icon: "gnome-aisleriot", Emoji: "🃏", Color: "#27ae60"},

	{ID: "htop", Name: "htop", Category: "System", Description: "See what's running", Packages: []string{"htop"}, Desktop: "htop.desktop", Exec: "htop", Icon: "htop", Emoji: "📊", Color: "#2ecc71"},
	{ID: "fastfetch", Name: "fastfetch", Category: "System", Description: "Show off your system info", Packages: []string{"fastfetch"}, Exec: "fastfetch", Icon: "utilities-terminal", Emoji: "✨", Color: "#9b59b6"},
	{ID: "gparted", Name: "GParted", Category: "System", Description: "Edit disk partitions", Packages: []string{"gparted"}, Desktop: "gparted.desktop", Exec: "gparted", Icon: "gparted", Emoji: "💽", Color: "#e74c3c"},
	{ID: "archives", Name: "Archive Manager", Category: "System", Description: "Open .zip and other archives", Packages: []string{"file-roller"}, Desktop: "org.gnome.FileRoller.desktop", Exec: "file-roller", Icon: "org.gnome.FileRoller", Emoji: "🗜️", Color: "#95a5a6"},
	{ID: "flameshot", Name: "Flameshot", Category: "System", Description: "Take screenshots", Packages: []string{"flameshot"}, Desktop: "org.flameshot.Flameshot.desktop", Exec: "flameshot", Icon: "org.flameshot.Flameshot", Emoji: "📸", Color: "#fc5c65"},

	{ID: "store", Name: "App Store", Category: "System", Description: "Get thousands more apps from Flathub, like the official Minecraft launcher", Packages: []string{"gnome-software", "flatpak"}, Desktop: "org.gnome.Software.desktop", Exec: "gnome-software", Icon: "org.gnome.Software", Emoji: "🛍️", Color: "#e66100"},
	{ID: "vivaldi", Name: "Vivaldi", Category: "Internet", Description: "Web browser with lots of options", Packages: []string{"vivaldi"}, Desktop: "vivaldi-stable.desktop", Exec: "vivaldi-stable", Icon: "vivaldi", Emoji: "🌀", Color: "#ef3939"},
	{ID: "tor", Name: "Tor Browser", Category: "Internet", Description: "Private web browsing", Packages: []string{"torbrowser-launcher"}, Desktop: "torbrowser.desktop", Exec: "torbrowser-launcher", Icon: "torbrowser", Emoji: "🧅", Color: "#7d4698"},
	{ID: "epiphany", Name: "GNOME Web", Category: "Internet", Description: "Simple web browser", Packages: []string{"epiphany"}, Desktop: "org.gnome.Epiphany.desktop", Exec: "epiphany", Icon: "org.gnome.Epiphany", Emoji: "🌍", Color: "#3584e4"},
	{ID: "signal", Name: "Signal", Category: "Internet", Description: "Private messaging", Packages: []string{"signal-desktop"}, Desktop: "signal.desktop", Exec: "signal-desktop", Icon: "signal-desktop", Emoji: "🔵", Color: "#3a76f0"},
	{ID: "element", Name: "Element", Category: "Internet", Description: "Chat on Matrix", Packages: []string{"element-desktop"}, Desktop: "io.element.Element.desktop", Exec: "element-desktop", Icon: "io.element.Element", Emoji: "🟢", Color: "#0dbd8b"},
	{ID: "qbittorrent", Name: "qBittorrent", Category: "Internet", Description: "Torrent downloads", Packages: []string{"qbittorrent"}, Desktop: "org.qbittorrent.qBittorrent.desktop", Exec: "qbittorrent", Icon: "qbittorrent", Emoji: "🧲", Color: "#2f67ba"},
	{ID: "filezilla", Name: "FileZilla", Category: "Internet", Description: "Upload files to servers", Packages: []string{"filezilla"}, Desktop: "filezilla.desktop", Exec: "filezilla", Icon: "filezilla", Emoji: "📡", Color: "#bf0000"},
	{ID: "remmina", Name: "Remmina", Category: "Internet", Description: "Control other computers", Packages: []string{"remmina"}, Desktop: "org.remmina.Remmina.desktop", Exec: "remmina", Icon: "org.remmina.Remmina", Emoji: "🖥️", Color: "#3a7bd5"},
	{ID: "kdeconnect", Name: "KDE Connect", Category: "Internet", Description: "Link your phone to your computer", Packages: []string{"kdeconnect"}, Desktop: "org.kde.kdeconnect.app.desktop", Exec: "kdeconnect-app", Icon: "kdeconnect", Emoji: "📱", Color: "#1d99f3"},
	{ID: "mpv", Name: "mpv", Category: "Media", Description: "Lightweight video player", Packages: []string{"mpv"}, Desktop: "mpv.desktop", Exec: "mpv", Icon: "mpv", Emoji: "⏯️", Color: "#6a1b9a"},
	{ID: "celluloid", Name: "Celluloid", Category: "Media", Description: "Simple video player", Packages: []string{"celluloid"}, Desktop: "io.github.celluloid_player.Celluloid.desktop", Exec: "celluloid", Icon: "io.github.celluloid_player.Celluloid", Emoji: "📽️", Color: "#e66100"},
	{ID: "rhythmbox", Name: "Rhythmbox", Category: "Media", Description: "Music library and radio", Packages: []string{"rhythmbox"}, Desktop: "org.gnome.Rhythmbox3.desktop", Exec: "rhythmbox", Icon: "org.gnome.Rhythmbox3", Emoji: "🎧", Color: "#c88800"},
	{ID: "strawberry", Name: "Strawberry", Category: "Media", Description: "Music player", Packages: []string{"strawberry"}, Desktop: "org.strawberrymusicplayer.strawberry.desktop", Exec: "strawberry", Icon: "strawberry", Emoji: "🍓", Color: "#e01b24"},
	{ID: "kodi", Name: "Kodi", Category: "Media", Description: "Media center for your TV", Packages: []string{"kodi"}, Desktop: "kodi.desktop", Exec: "kodi", Icon: "kodi", Emoji: "📺", Color: "#17b2e7"},
	{ID: "shotcut", Name: "Shotcut", Category: "Media", Description: "Video editor", Packages: []string{"shotcut"}, Desktop: "org.shotcut.Shotcut.desktop", Exec: "shotcut", Icon: "org.shotcut.Shotcut", Emoji: "✂️", Color: "#1e6fb8"},
	{ID: "openshot", Name: "OpenShot", Category: "Media", Description: "Easy video editor", Packages: []string{"openshot"}, Desktop: "org.openshot.OpenShot.desktop", Exec: "openshot-qt", Icon: "openshot-qt", Emoji: "🎥", Color: "#2b5797"},
	{ID: "handbrake", Name: "HandBrake", Category: "Media", Description: "Convert videos to other formats", Packages: []string{"handbrake"}, Desktop: "fr.handbrake.ghb.desktop", Exec: "ghb", Icon: "fr.handbrake.ghb", Emoji: "🍹", Color: "#c4342d"},
	{ID: "lmms", Name: "LMMS", Category: "Media", Description: "Make beats and music", Packages: []string{"lmms"}, Desktop: "lmms.desktop", Exec: "lmms", Icon: "lmms", Emoji: "🎹", Color: "#27ae60"},
	{ID: "musescore", Name: "MuseScore", Category: "Media", Description: "Write sheet music", Packages: []string{"musescore"}, Desktop: "org.musescore.MuseScore.desktop", Exec: "mscore", Icon: "mscore", Emoji: "🎼", Color: "#1a5fb4"},
	{ID: "cheese", Name: "Cheese", Category: "Media", Description: "Webcam photos and videos", Packages: []string{"cheese"}, Desktop: "org.gnome.Cheese.desktop", Exec: "cheese", Icon: "org.gnome.Cheese", Emoji: "📷", Color: "#d4a017"},
	{ID: "darktable", Name: "darktable", Category: "Graphics", Description: "Edit camera photos", Packages: []string{"darktable"}, Desktop: "org.darktable.darktable.desktop", Exec: "darktable", Icon: "darktable", Emoji: "🌗", Color: "#444444"},
	{ID: "pinta", Name: "Pinta", Category: "Graphics", Description: "Simple paint program", Packages: []string{"pinta"}, Desktop: "com.github.PintaProject.Pinta.desktop", Exec: "pinta", Icon: "com.github.PintaProject.Pinta", Emoji: "🖍️", Color: "#f39c12"},
	{ID: "mypaint", Name: "MyPaint", Category: "Graphics", Description: "Sketching and painting", Packages: []string{"mypaint"}, Desktop: "org.mypaint.MyPaint.desktop", Exec: "mypaint", Icon: "org.mypaint.MyPaint", Emoji: "✏️", Color: "#9b59b6"},
	{ID: "kolourpaint", Name: "KolourPaint", Category: "Graphics", Description: "Paint, like MS Paint", Packages: []string{"kolourpaint"}, Desktop: "org.kde.kolourpaint.desktop", Exec: "kolourpaint", Icon: "kolourpaint", Emoji: "🪣", Color: "#e67e22"},
	{ID: "synfig", Name: "Synfig Studio", Category: "Graphics", Description: "Make 2D animations", Packages: []string{"synfigstudio"}, Desktop: "org.synfig.SynfigStudio.desktop", Exec: "synfigstudio", Icon: "org.synfig.SynfigStudio", Emoji: "🏃", Color: "#d35400"},
	{ID: "freecad", Name: "FreeCAD", Category: "Graphics", Description: "3D design for real objects", Packages: []string{"freecad"}, Desktop: "org.freecad.FreeCAD.desktop", Exec: "freecad", Icon: "org.freecad.FreeCAD", Emoji: "📐", Color: "#cb333b"},
	{ID: "digikam", Name: "digiKam", Category: "Graphics", Description: "Organise your photos", Packages: []string{"digikam"}, Desktop: "org.kde.digikam.desktop", Exec: "digikam", Icon: "digikam", Emoji: "🗂️", Color: "#2980b9"},
	{ID: "scribus", Name: "Scribus", Category: "Graphics", Description: "Make posters and magazines", Packages: []string{"scribus"}, Desktop: "scribus.desktop", Exec: "scribus", Icon: "scribus", Emoji: "📰", Color: "#2c3e50"},
	{ID: "okular", Name: "Okular", Category: "Office", Description: "Read PDFs and e-books", Packages: []string{"okular"}, Desktop: "org.kde.okular.desktop", Exec: "okular", Icon: "okular", Emoji: "📖", Color: "#3498db"},
	{ID: "xournalpp", Name: "Xournal++", Category: "Office", Description: "Handwritten notes", Packages: []string{"xournalpp"}, Desktop: "com.github.xournalpp.xournalpp.desktop", Exec: "xournalpp", Icon: "com.github.xournalpp.xournalpp", Emoji: "🖊️", Color: "#1abc9c"},
	{ID: "obsidian", Name: "Obsidian", Category: "Office", Description: "Notes that link together", Packages: []string{"obsidian"}, Desktop: "obsidian.desktop", Exec: "obsidian", Icon: "obsidian", Emoji: "💎", Color: "#7c3aed"},
	{ID: "calibre", Name: "Calibre", Category: "Office", Description: "E-book library", Packages: []string{"calibre"}, Desktop: "calibre-gui.desktop", Exec: "calibre", Icon: "calibre-gui", Emoji: "📚", Color: "#b5651d"},
	{ID: "foliate", Name: "Foliate", Category: "Office", Description: "E-book reader", Packages: []string{"foliate"}, Desktop: "com.github.johnfactotum.Foliate.desktop", Exec: "foliate", Icon: "com.github.johnfactotum.Foliate", Emoji: "🍃", Color: "#2ec27e"},
	{ID: "gnomecal", Name: "Calendar", Category: "Office", Description: "Plan your days", Packages: []string{"gnome-calendar"}, Desktop: "org.gnome.Calendar.desktop", Exec: "gnome-calendar", Icon: "org.gnome.Calendar", Emoji: "🗓️", Color: "#3584e4"},
	{ID: "scanner", Name: "Document Scanner", Category: "Office", Description: "Scan paper documents", Packages: []string{"simple-scan"}, Desktop: "org.gnome.SimpleScan.desktop", Exec: "simple-scan", Icon: "org.gnome.SimpleScan", Emoji: "🖨️", Color: "#77767b"},
	{ID: "gnucash", Name: "GnuCash", Category: "Office", Description: "Keep track of money", Packages: []string{"gnucash"}, Desktop: "gnucash.desktop", Exec: "gnucash", Icon: "gnucash", Emoji: "💰", Color: "#27ae60"},
	{ID: "stellarium", Name: "Stellarium", Category: "Learning", Description: "Planetarium: see the stars", Packages: []string{"stellarium"}, Desktop: "org.stellarium.Stellarium.desktop", Exec: "stellarium", Icon: "stellarium", Emoji: "🌌", Color: "#1a237e"},
	{ID: "gcompris", Name: "GCompris", Category: "Learning", Description: "Over 100 learning games for kids", Packages: []string{"gcompris-qt"}, Desktop: "org.kde.gcompris.desktop", Exec: "gcompris-qt", Icon: "gcompris-qt", Emoji: "🧸", Color: "#f9a825"},
	{ID: "tuxpaint", Name: "Tux Paint", Category: "Learning", Description: "Drawing for kids", Packages: []string{"tuxpaint"}, Desktop: "tuxpaint.desktop", Exec: "tuxpaint", Icon: "tuxpaint", Emoji: "🦄", Color: "#ff6f00"},
	{ID: "marble", Name: "Marble", Category: "Learning", Description: "Globe and maps", Packages: []string{"marble"}, Desktop: "org.kde.marble.desktop", Exec: "marble", Icon: "marble", Emoji: "🗺️", Color: "#0277bd"},
	{ID: "anki", Name: "Anki", Category: "Learning", Description: "Flashcards for studying", Packages: []string{"anki"}, Desktop: "anki.desktop", Exec: "anki", Icon: "anki", Emoji: "🗃️", Color: "#5c9ce6"},
	{ID: "kicad", Name: "KiCad", Category: "Learning", Description: "Design electronic circuit boards", Packages: []string{"kicad", "kicad-library"}, Desktop: "org.kicad.kicad.desktop", Exec: "kicad", Icon: "kicad", Emoji: "🔌", Color: "#314cb0"},
	{ID: "godot", Name: "Godot", Category: "Coding", Description: "Make your own games", Packages: []string{"godot"}, Desktop: "org.godotengine.Godot.desktop", Exec: "godot", Icon: "godot", Emoji: "🤖", Color: "#478cbf"},
	{ID: "neovim", Name: "Neovim", Category: "Coding", Description: "Text editor for the terminal", Packages: []string{"neovim"}, Desktop: "nvim.desktop", Exec: "nvim", Icon: "nvim", Emoji: "📜", Color: "#57a143"},
	{ID: "emacs", Name: "Emacs", Category: "Coding", Description: "Extensible text editor", Packages: []string{"emacs"}, Desktop: "emacs.desktop", Exec: "emacs", Icon: "emacs", Emoji: "🦬", Color: "#7f5ab6"},
	{ID: "kate", Name: "Kate", Category: "Coding", Description: "Code editor from KDE", Packages: []string{"kate"}, Desktop: "org.kde.kate.desktop", Exec: "kate", Icon: "kate", Emoji: "🪶", Color: "#1d99f3"},
	{ID: "nodejs", Name: "Node.js", Category: "Coding", Description: "JavaScript outside the browser", Packages: []string{"nodejs", "npm"}, Exec: "node", Icon: "nodejs", Emoji: "🟩", Color: "#339933"},
	{ID: "rust", Name: "Rust", Category: "Coding", Description: "Rust programming language", Packages: []string{"rust"}, Exec: "cargo", Icon: "rust", Emoji: "🦀", Color: "#b7410e"},
	{ID: "golang", Name: "Go", Category: "Coding", Description: "Go programming language", Packages: []string{"go"}, Exec: "go", Icon: "go", Emoji: "🐹", Color: "#00add8"},
	{ID: "java", Name: "Java", Category: "Coding", Description: "Java programming language", Packages: []string{"jdk-openjdk"}, Exec: "java", Icon: "java", Emoji: "☕", Color: "#f89820"},
	{ID: "meld", Name: "Meld", Category: "Coding", Description: "Compare files and folders", Packages: []string{"meld"}, Desktop: "org.gnome.Meld.desktop", Exec: "meld", Icon: "org.gnome.Meld", Emoji: "🔀", Color: "#8bc34a"},
	{ID: "sqlitebrowser", Name: "DB Browser for SQLite", Category: "Coding", Description: "Look inside databases", Packages: []string{"sqlitebrowser"}, Desktop: "sqlitebrowser.desktop", Exec: "sqlitebrowser", Icon: "sqlitebrowser", Emoji: "🗄️", Color: "#2b7bb9"},
	{ID: "wireshark", Name: "Wireshark", Category: "Coding", Description: "See network traffic", Packages: []string{"wireshark-qt"}, Desktop: "org.wireshark.Wireshark.desktop", Exec: "wireshark", Icon: "org.wireshark.Wireshark", Emoji: "🦈", Color: "#1679a7"},
	{ID: "boxes", Name: "GNOME Boxes", Category: "Coding", Description: "Run other operating systems in a window", Packages: []string{"gnome-boxes"}, Desktop: "org.gnome.Boxes.desktop", Exec: "gnome-boxes", Icon: "org.gnome.Boxes", Emoji: "📦", Color: "#c64600"},
	{ID: "xonotic", Name: "Xonotic", Category: "Games", Description: "Fast arena shooter", Packages: []string{"xonotic"}, Desktop: "xonotic-glx.desktop", Exec: "xonotic-glx", Icon: "xonotic", Emoji: "🎯", Color: "#6f2c91"},
	{ID: "openttd", Name: "OpenTTD", Category: "Games", Description: "Build a transport empire", Packages: []string{"openttd", "openttd-opengfx"}, Desktop: "openttd.desktop", Exec: "openttd", Icon: "openttd", Emoji: "🚂", Color: "#4a8a2a"},
	{ID: "wesnoth", Name: "Battle for Wesnoth", Category: "Games", Description: "Fantasy strategy", Packages: []string{"wesnoth"}, Desktop: "wesnoth.desktop", Exec: "wesnoth", Icon: "wesnoth", Emoji: "⚔️", Color: "#8b5a2b"},
	{ID: "warzone", Name: "Warzone 2100", Category: "Games", Description: "Real-time strategy", Packages: []string{"warzone2100"}, Desktop: "warzone2100.desktop", Exec: "warzone2100", Icon: "warzone2100", Emoji: "🚀", Color: "#555555"},
	{ID: "hedgewars", Name: "Hedgewars", Category: "Games", Description: "Funny turn-based hedgehog battles", Packages: []string{"hedgewars"}, Desktop: "hedgewars.desktop", Exec: "hedgewars", Icon: "hedgewars", Emoji: "🦔", Color: "#c0392b"},
	{ID: "etr", Name: "Extreme Tux Racer", Category: "Games", Description: "Slide down snowy hills", Packages: []string{"extremetuxracer"}, Desktop: "etr.desktop", Exec: "etr", Icon: "etr", Emoji: "⛷️", Color: "#2e86de"},
	{ID: "teeworlds", Name: "Teeworlds", Category: "Games", Description: "2D online shooter", Packages: []string{"teeworlds"}, Desktop: "teeworlds.desktop", Exec: "teeworlds", Icon: "teeworlds", Emoji: "💥", Color: "#d35400"},
	{ID: "neverball", Name: "Neverball", Category: "Games", Description: "Tilt the floor to roll the ball", Packages: []string{"neverball"}, Desktop: "neverball.desktop", Exec: "neverball", Icon: "neverball", Emoji: "🎱", Color: "#16a085"},
	{ID: "frozenbubble", Name: "Frozen Bubble", Category: "Games", Description: "Pop the bubbles", Packages: []string{"frozen-bubble"}, Desktop: "frozen-bubble.desktop", Exec: "frozen-bubble", Icon: "frozen-bubble", Emoji: "🫧", Color: "#3498db"},
	{ID: "doom", Name: "Doom (Freedoom)", Category: "Games", Description: "Classic Doom-style shooter", Packages: []string{"gzdoom", "freedoom"}, Desktop: "gzdoom.desktop", Exec: "gzdoom", Icon: "gzdoom", Emoji: "👹", Color: "#8b0000"},
	{ID: "scummvm", Name: "ScummVM", Category: "Games", Description: "Play classic adventure games", Packages: []string{"scummvm"}, Desktop: "org.scummvm.scummvm.desktop", Exec: "scummvm", Icon: "org.scummvm.scummvm", Emoji: "🗝️", Color: "#cc6600"},
	{ID: "chess", Name: "Chess", Category: "Games", Description: "Play chess", Packages: []string{"gnome-chess"}, Desktop: "org.gnome.Chess.desktop", Exec: "gnome-chess", Icon: "org.gnome.Chess", Emoji: "♟️", Color: "#3d3846"},
	{ID: "mines", Name: "Mines", Category: "Games", Description: "Minesweeper", Packages: []string{"gnome-mines"}, Desktop: "org.gnome.Mines.desktop", Exec: "gnome-mines", Icon: "org.gnome.Mines", Emoji: "💣", Color: "#333333"},
	{ID: "sudoku", Name: "Sudoku", Category: "Games", Description: "Number puzzles", Packages: []string{"gnome-sudoku"}, Desktop: "org.gnome.Sudoku.desktop", Exec: "gnome-sudoku", Icon: "org.gnome.Sudoku", Emoji: "🔢", Color: "#1c71d8"},
	{ID: "g2048", Name: "2048", Category: "Games", Description: "Slide the tiles to 2048", Packages: []string{"gnome-2048"}, Desktop: "org.gnome.TwentyFortyEight.desktop", Exec: "gnome-2048", Icon: "org.gnome.TwentyFortyEight", Emoji: "🟨", Color: "#c9a227"},
	{ID: "tetris", Name: "Quadrapassel", Category: "Games", Description: "Falling-blocks puzzle, like Tetris", Packages: []string{"quadrapassel"}, Desktop: "org.gnome.Quadrapassel.desktop", Exec: "quadrapassel", Icon: "org.gnome.Quadrapassel", Emoji: "🟦", Color: "#3584e4"},
	{ID: "mahjongg", Name: "Mahjongg", Category: "Games", Description: "Match the tiles", Packages: []string{"gnome-mahjongg"}, Desktop: "org.gnome.Mahjongg.desktop", Exec: "gnome-mahjongg", Icon: "org.gnome.Mahjongg", Emoji: "🀄", Color: "#c01c28"},
	{ID: "lutris", Name: "Lutris", Category: "Games", Description: "One place for all your games", Packages: []string{"lutris"}, Desktop: "net.lutris.Lutris.desktop", Exec: "lutris", Icon: "net.lutris.Lutris", Emoji: "🦦", Color: "#ff9800"},
	{ID: "mgba", Name: "mGBA", Category: "Games", Description: "Game Boy Advance emulator", Packages: []string{"mgba-qt"}, Desktop: "io.mgba.mGBA.desktop", Exec: "mgba-qt", Icon: "io.mgba.mGBA", Emoji: "👾", Color: "#6b4ea0"},
	{ID: "dolphin", Name: "Dolphin", Category: "Games", Description: "GameCube and Wii emulator", Packages: []string{"dolphin-emu"}, Desktop: "dolphin-emu.desktop", Exec: "dolphin-emu", Icon: "dolphin-emu", Emoji: "🐬", Color: "#2fa3e3"},
	{ID: "desmume", Name: "DeSmuME", Category: "Games", Description: "Nintendo DS emulator", Packages: []string{"desmume"}, Desktop: "org.desmume.DeSmuME.desktop", Exec: "desmume", Icon: "org.desmume.DeSmuME", Emoji: "🎴", Color: "#8e44ad"},
	{ID: "btop", Name: "btop", Category: "System", Description: "Good-looking system monitor", Packages: []string{"btop"}, Desktop: "btop.desktop", Exec: "btop", Icon: "btop", Emoji: "🧭", Color: "#00897b"},
	{ID: "baobab", Name: "Disk Usage", Category: "System", Description: "See what fills your disk", Packages: []string{"baobab"}, Desktop: "org.gnome.baobab.desktop", Exec: "baobab", Icon: "org.gnome.baobab", Emoji: "🥧", Color: "#c88800"},
	{ID: "disks", Name: "Disks", Category: "System", Description: "Format and check drives", Packages: []string{"gnome-disk-utility"}, Desktop: "org.gnome.DiskUtility.desktop", Exec: "gnome-disks", Icon: "org.gnome.DiskUtility", Emoji: "💾", Color: "#555555"},
	{ID: "timeshift", Name: "Timeshift", Category: "System", Description: "Back up and restore your system", Packages: []string{"timeshift"}, Desktop: "timeshift-gtk.desktop", Exec: "timeshift-launcher", Icon: "timeshift", Emoji: "⏪", Color: "#4caf50"},
	{ID: "bleachbit", Name: "BleachBit", Category: "System", Description: "Clean up junk files", Packages: []string{"bleachbit"}, Desktop: "org.bleachbit.BleachBit.desktop", Exec: "bleachbit", Icon: "bleachbit", Emoji: "🧹", Color: "#2e7d32"},
	{ID: "wine", Name: "Wine", Category: "System", Description: "Run Windows programs", Packages: []string{"wine", "wine-mono", "wine-gecko"}, Desktop: "wine.desktop", Exec: "winecfg", Icon: "wine", Emoji: "🍷", Color: "#800020", Multilib: true},
	{ID: "openrgb", Name: "OpenRGB", Category: "System", Description: "Control RGB lights", Packages: []string{"openrgb"}, Desktop: "org.openrgb.OpenRGB.desktop", Exec: "openrgb", Icon: "OpenRGB", Emoji: "🌈", Color: "#c2185b"},
}

// AppByID returns the catalog entry with the given id.
func AppByID(id string) (App, bool) {
	for _, a := range Catalog {
		if a.ID == id {
			return a, true
		}
	}
	return App{}, false
}

// Keyboard is a keyboard layout, named for both X11 and the text console
// since the two use different names for some layouts.
type Keyboard struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Console string `json:"console"`
}

// Keyboards lists the layouts offered in the designer.
var Keyboards = []Keyboard{
	{"us", "English (US)", "us"},
	{"gb", "English (UK)", "uk"},
	{"de", "German", "de-latin1"},
	{"fr", "French", "fr-latin1"},
	{"es", "Spanish", "es"},
	{"it", "Italian", "it"},
	{"pt", "Portuguese", "pt-latin1"},
	{"br", "Portuguese (Brazil)", "br-abnt2"},
	{"nl", "Dutch", "nl"},
	{"se", "Swedish", "sv-latin1"},
	{"no", "Norwegian", "no-latin1"},
	{"dk", "Danish", "dk-latin1"},
	{"fi", "Finnish", "fi"},
	{"pl", "Polish", "pl"},
	{"ru", "Russian", "ru"},
	{"jp", "Japanese", "jp106"},
}

// KeyboardByID returns the layout with the given X11 name.
func KeyboardByID(id string) (Keyboard, bool) {
	for _, k := range Keyboards {
		if k.ID == id {
			return k, true
		}
	}
	return Keyboard{}, false
}

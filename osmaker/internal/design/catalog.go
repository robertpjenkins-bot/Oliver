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
var Categories = []string{"Built in", "Internet", "Media", "Graphics", "Office", "Coding", "Games", "System"}

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
	{ID: "prism", Name: "Minecraft (Prism)", Category: "Games", Description: "Launcher for Minecraft: Java Edition", Packages: []string{"prismlauncher"}, Desktop: "org.prismlauncher.PrismLauncher.desktop", Exec: "prismlauncher", Icon: "org.prismlauncher.PrismLauncher", Emoji: "⛏️", Color: "#62b47a"},
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

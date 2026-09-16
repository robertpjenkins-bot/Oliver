// Package procoder provides Pro Coder: a small launcher that opens Scratch,
// MakeCode Arcade and CCP — Oliver's own block coding program — inside one
// window, and turns a shared project into something that can be kept.
//
// The launcher itself is a single HTML page, embedded in the binary. That
// choice is deliberate: the three things it launches are web applications, so
// the honest way to show them is a browser view, and a page needs no toolkit,
// no cgo and no second build. Oliver stays one static file.
package procoder

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strings"
)

// appHTML is the launcher. It is carried inside the binary so that
// `oliver procoder` works on a machine that has nothing else installed.
//
//go:embed app.html
var appHTML []byte

// App returns the launcher page.
func App() []byte { return appHTML }

// Kind names the sort of project a URL points at.
type Kind string

const (
	KindScratch   Kind = "scratch"
	KindTurboWarp Kind = "turbowarp"
	KindMakeCode  Kind = "makecode"
	KindWeb       Kind = "web"
)

// Target is a project URL that Pro Coder understands well enough to convert.
//
// Page and Play are different URLs on purpose. Page is where a person goes to
// look at the project; Play is the address that actually runs it inside a
// frame. Scratch and MakeCode both refuse to be framed on their normal pages
// and both publish a player address that allows it, so converting a project
// means knowing which of the two to put in the wrapper.
type Target struct {
	Kind  Kind
	ID    string
	Title string
	Page  string
	Play  string
}

var (
	scratchURL   = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?scratch\.mit\.edu/projects/(\d+)`)
	turboWarpURL = regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?turbowarp\.org/(\d+)`)
	makeCodeURL  = regexp.MustCompile(`(?i)^(?:https?://)?(?:arcade\.)?makecode\.com/(_[A-Za-z0-9]+|S?[0-9]{4,6}(?:-[0-9]{4,6}){2,4})`)
	bareID       = regexp.MustCompile(`^\d{4,}$`)
	schemeChars  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.\-]*$`)
)

// Recognise works out what a pasted URL is.
//
// Anything that is not one of the sites Pro Coder knows is still accepted as
// an ordinary web address: converting a page nobody has heard of is a
// reasonable thing to want, and the wrapper is the same either way. What is
// not accepted is a scheme other than http or https — a javascript: or data:
// URL pasted into the Convert box would be code, not a project, and it would
// end up running inside whatever the wrapper is opened by.
func Recognise(raw string) (Target, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Target{}, fmt.Errorf("no project address given")
	}

	if m := scratchURL.FindStringSubmatch(s); m != nil {
		return scratchTarget(m[1]), nil
	}
	if m := turboWarpURL.FindStringSubmatch(s); m != nil {
		id := m[1]
		return Target{
			Kind:  KindTurboWarp,
			ID:    id,
			Title: "TurboWarp project " + id,
			Page:  "https://turbowarp.org/" + id,
			Play:  "https://turbowarp.org/" + id + "/embed",
		}, nil
	}
	if m := makeCodeURL.FindStringSubmatch(s); m != nil {
		id := m[1]
		return Target{
			Kind:  KindMakeCode,
			ID:    id,
			Title: "MakeCode Arcade game " + id,
			Page:  "https://arcade.makecode.com/" + id,
			// ---run is MakeCode's player address: the game, without the
			// editor around it, and framable.
			Play: "https://arcade.makecode.com/---run?id=" + url.QueryEscape(id),
		}, nil
	}
	// A project number on its own is what people copy out of a Scratch URL
	// when they only take the last part of it.
	if bareID.MatchString(s) {
		return scratchTarget(s), nil
	}

	// "example.com/thing" is how people write an address when they are not
	// thinking about schemes, so fill one in — but only when what is there is
	// not a scheme already. Otherwise "javascript:..." would quietly become
	// "https://javascript:...", and the error would describe the wrong thing.
	if scheme := schemePrefix(s); scheme == "" {
		s = "https://" + s
	} else if scheme != "http" && scheme != "https" {
		return Target{}, fmt.Errorf("only http and https addresses can be converted, not %s:", scheme)
	}
	u, err := url.Parse(s)
	if err != nil {
		return Target{}, fmt.Errorf("that does not look like a web address: %s", raw)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return Target{}, fmt.Errorf("only http and https addresses can be converted, not %s:", u.Scheme)
	}
	if u.Host == "" {
		return Target{}, fmt.Errorf("that address has no site in it: %s", raw)
	}
	return Target{
		Kind:  KindWeb,
		ID:    u.Host,
		Title: u.Host,
		Page:  u.String(),
		Play:  u.String(),
	}, nil
}

// schemePrefix returns the URL scheme a string starts with, or "" when it
// starts with none.
func schemePrefix(s string) string {
	if i := strings.IndexByte(s, ':'); i > 0 && schemeChars.MatchString(s[:i]) {
		return strings.ToLower(s[:i])
	}
	return ""
}

func scratchTarget(id string) Target {
	return Target{
		Kind:  KindScratch,
		ID:    id,
		Title: "Scratch project " + id,
		Page:  "https://scratch.mit.edu/projects/" + id,
		// /embed is the one Scratch address that allows framing: the project
		// player with a green flag and nothing else.
		Play: "https://scratch.mit.edu/projects/" + id + "/embed",
	}
}

// Name suggests a file name for a converted project.
func (t Target) Name() string {
	switch t.Kind {
	case KindScratch:
		return "scratch-" + t.ID
	case KindTurboWarp:
		return "turbowarp-" + t.ID
	case KindMakeCode:
		return "arcade-" + strings.TrimPrefix(t.ID, "_")
	default:
		return strings.NewReplacer(".", "-", ":", "-").Replace(t.ID)
	}
}

// wrapper is the standalone page a converted project becomes: the project
// filling the window, with its address on show and a way out to the real site
// for the days when framing is refused.
var wrapper = template.Must(template.New("wrapper").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Name}}</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body { margin: 0; height: 100vh; display: flex; flex-direction: column;
         background: #10131a; color: #e8ecf4;
         font: 15px/1.4 system-ui, -apple-system, "Segoe UI", sans-serif; }
  header { display: flex; align-items: center; gap: 10px; padding: 8px 12px;
           background: #1a1f2b; border-bottom: 1px solid #2b3345; }
  h1 { font-size: 15px; margin: 0; white-space: nowrap; }
  .url { flex: 1; min-width: 0; font: 12px ui-monospace, monospace; color: #9fb0cc;
         background: #10131a; border: 1px solid #2b3345; border-radius: 6px;
         padding: 6px 8px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  a.open { color: #10131a; background: #59c2ff; text-decoration: none;
           padding: 6px 12px; border-radius: 6px; font-weight: 600; white-space: nowrap; }
  main { flex: 1; position: relative; }
  iframe { position: absolute; inset: 0; width: 100%; height: 100%; border: 0; background: #000; }
  .note { position: absolute; left: 50%; bottom: 14px; transform: translateX(-50%);
          background: #1a1f2b; border: 1px solid #2b3345; border-radius: 8px;
          padding: 8px 14px; color: #9fb0cc; font-size: 13px; }
</style>
</head>
<body>
<header>
  <h1>{{.Name}}</h1>
  <div class="url" title="{{.Page}}">{{.Page}}</div>
  <a class="open" href="{{.Page}}" target="_blank" rel="noopener">Open the original</a>
</header>
<main>
  <iframe src="{{.Play}}" allow="autoplay; fullscreen; gamepad; microphone" allowfullscreen></iframe>
  <div class="note">Nothing showing? The site may not allow being framed — use “Open the original”.</div>
</main>
</body>
</html>
`))

// WrapperHTML builds the standalone page for a project. The result is one
// file with nothing to install: it can be opened by double-clicking it,
// emailed, or put on a memory stick.
func WrapperHTML(t Target, name string) ([]byte, error) {
	if name == "" {
		name = t.Title
	}
	var b strings.Builder
	err := wrapper.Execute(&b, struct {
		Name string
		Page template.URL
		Play template.URL
	}{
		Name: name,
		// Both addresses were checked in Recognise, so they are http(s) and
		// safe to mark as URLs the template may emit unescaped in an
		// attribute.
		Page: template.URL(t.Page),
		Play: template.URL(t.Play),
	})
	if err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

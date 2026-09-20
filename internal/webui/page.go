package webui

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
)

// The page is compiled into the binary. Oliver ships as one file — an AppImage
// on a stick, often — so a front end that needed a directory of assets beside
// it would not survive being copied anywhere.
//
//go:embed assets/index.html
var pageHTML string

// tokenPlaceholder is replaced with the session token as the page is served.
// The token is hex, so it cannot break out of the string it lands in.
const tokenPlaceholder = "__OLIVER_TOKEN__"

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	if !loopbackHost(r.Host) {
		fail(w, http.StatusForbidden, fmt.Errorf("this server answers only on localhost"))
		return
	}
	// The browser was opened at a URL carrying the token; that is the only
	// thing a fresh page load can bring with it.
	if !s.validToken(r.URL.Query().Get("t")) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "This page needs the session link Oliver printed when it started.\n"+
			"Open that link, or run `oliver ui` again for a new one.\n")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Nothing here is fetched from anywhere else, and saying so means a
	// mistake in the page cannot become a request to somewhere else.
	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; form-action 'none'; base-uri 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, strings.Replace(pageHTML, tokenPlaceholder, s.Token, 1))
}

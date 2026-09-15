package gamekit

import (
	"strings"
	"testing"
)

// Pointing the studio at a link is a reasonable thing to try, and the most
// likely link is the wrong one. What matters is that it says so at once, and
// names the thing that does work.
func TestEndpointsThatAreNotAPIs(t *testing.T) {
	for _, raw := range []string{
		"https://claude.ai/code",
		"https://claude.ai",
		"claude.ai/code",
		"https://www.claude.ai/chat/123",
		"https://claude.com/code",
	} {
		_, err := CheckEndpoint(raw)
		if err == nil {
			t.Errorf("%q was accepted as an API address", raw)
			continue
		}
		if !strings.Contains(err.Error(), "claude-code") {
			t.Errorf("%q: the error does not say what to do instead:\n%v", raw, err)
		}
	}

	if _, err := CheckEndpoint("https://console.anthropic.com"); err == nil ||
		!strings.Contains(err.Error(), "sk-ant-") {
		t.Errorf("the console should be explained as where keys come from: %v", err)
	}
}

// A gateway that speaks the same protocol is a legitimate thing to point at.
func TestEndpointsThatAreAPIs(t *testing.T) {
	cases := map[string]string{
		"https://api.anthropic.com":             "https://api.anthropic.com",
		"https://api.anthropic.com/":            "https://api.anthropic.com",
		"https://api.anthropic.com/v1":          "https://api.anthropic.com",
		"https://api.anthropic.com/v1/messages": "https://api.anthropic.com",
		"http://127.0.0.1:8899":                 "http://127.0.0.1:8899",
		"https://gateway.example.com/anthropic": "https://gateway.example.com/anthropic",
		"gateway.example.com":                   "https://gateway.example.com",
		"":                                      "",
	}
	for raw, want := range cases {
		got, err := CheckEndpoint(raw)
		if err != nil {
			t.Errorf("CheckEndpoint(%q): %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("CheckEndpoint(%q) = %q, want %q", raw, got, want)
		}
	}

	if _, err := CheckEndpoint("ftp://example.com"); err == nil {
		t.Error("a non-web address was accepted")
	}
}

package gamekit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The order credentials are looked for matters: what somebody typed into the
// studio beats what the environment happens to hold, and an OAuth login is
// better than nothing at all.
func TestCredentialsAreFoundInOrder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OLIVER_STUDIO_SETTINGS", filepath.Join(dir, "studio.json"))
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	// An empty PATH keeps the machine's own `ant` login out of the test.
	t.Setenv("PATH", dir)

	if c := FindCredential(); c.ok() {
		t.Errorf("found a credential on a machine with none: %+v", c)
	}

	t.Setenv("ANTHROPIC_AUTH_TOKEN", "oauth-token")
	if c := FindCredential(); c.Token != "oauth-token" || c.Key != "" {
		t.Errorf("auth token: %+v", c)
	}

	t.Setenv("ANTHROPIC_API_KEY", "env-key")
	if c := FindCredential(); c.Key != "env-key" {
		t.Errorf("api key should beat auth token: %+v", c)
	}

	if err := SavePreferences(Preferences{APIKey: "saved-key"}); err != nil {
		t.Fatal(err)
	}
	if c := FindCredential(); c.Key != "saved-key" {
		t.Errorf("the saved key should win: %+v", c)
	}

	// A saved key is a secret: the file must not be readable by anyone else.
	info, err := os.Stat(SettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("the settings file is %v, want 0600", mode)
	}

	if err := ForgetKey(); err != nil {
		t.Fatal(err)
	}
	if c := FindCredential(); c.Key != "env-key" {
		t.Errorf("after forgetting the saved key: %+v", c)
	}
}

// An API key and an OAuth token are different headers, not different values of
// the same one. Sending a token as x-api-key fails in a way that reads like a
// bad key, which is a bad afternoon.
func TestCredentialsUseTheRightHeader(t *testing.T) {
	var gotKey, gotAuth, gotBeta string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotAuth = r.Header.Get("authorization")
		gotBeta = r.Header.Get("anthropic-beta")
		w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	if err := CheckCredential(context.Background(), srv.Client(), srv.URL, Credential{Key: "k"}); err != nil {
		t.Fatal(err)
	}
	if gotKey != "k" || gotAuth != "" {
		t.Errorf("api key sent as key=%q auth=%q", gotKey, gotAuth)
	}

	gotKey, gotAuth = "", ""
	if err := CheckCredential(context.Background(), srv.Client(), srv.URL, Credential{Token: "t"}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer t" || gotKey != "" {
		t.Errorf("oauth token sent as key=%q auth=%q", gotKey, gotAuth)
	}
	if !strings.Contains(gotBeta, "oauth") {
		t.Errorf("an oauth token needs its beta header, got %q", gotBeta)
	}
}

// A key that does not work has to be rejected while somebody is still looking
// at the box they typed it into.
func TestCheckCredentialReportsARejectedKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"invalid x-api-key"}}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := CheckCredential(context.Background(), srv.Client(), srv.URL, Credential{Key: "nope"})
	if err == nil || !strings.Contains(err.Error(), "not accepted") {
		t.Errorf("got %v, want a clear rejection", err)
	}
}

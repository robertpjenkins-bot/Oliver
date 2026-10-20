package main

import "testing"

func TestCleanDroppedPath(t *testing.T) {
	for in, want := range map[string]string{
		"/home/deck/Downloads/batocera-rk3326-r36s-43.img.gz": "/home/deck/Downloads/batocera-rk3326-r36s-43.img.gz",
		"'/home/deck/Downloads/My Files/b.img.gz' ":           "/home/deck/Downloads/My Files/b.img.gz",
		`"/home/deck/Downloads/b.img.gz"`:                     "/home/deck/Downloads/b.img.gz",
		`/home/deck/Downloads/My\ Files/b\ \(1\).img.gz`:  "/home/deck/Downloads/My Files/b (1).img.gz",
		"file:///home/deck/Downloads/My%20Files/b.img.gz":     "/home/deck/Downloads/My Files/b.img.gz",
		"": "",
	} {
		if got := cleanDroppedPath(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

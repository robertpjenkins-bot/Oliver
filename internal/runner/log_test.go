package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/robertpjenkins-bot/oliver/internal/pe"
	"github.com/robertpjenkins-bot/oliver/internal/prefix"
)

func TestExecuteRecordsOutputAndExitStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OLIVER_LOG_DIR", dir)

	// A stand-in for a program that prints and then fails on startup.
	script := filepath.Join(dir, "fake-wine")
	body := "#!/bin/sh\necho 'hello from stdout'\necho 'err:module:import_dll Library FOO.dll not found' >&2\nexit 53\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	m := &prefix.Manager{Root: t.TempDir()}
	p := m.For("", "demo")
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}

	plan := &Plan{
		Program: "/games/demo/demo.exe",
		Launch:  Step{Desc: "run demo.exe", Argv: []string{script}, Dir: dir},
	}
	logPath := LogPathFor("demo")

	res, err := Execute(plan, p, logPath, func(string, ...any) {})
	if err == nil {
		t.Fatal("expected the failing program to report an error")
	}
	if res.ExitCode != 53 {
		t.Errorf("ExitCode = %d, want 53", res.ExitCode)
	}
	if !res.Vanished {
		t.Error("a program that failed instantly should be reported as vanished")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	log := string(data)
	for _, want := range []string{
		"/games/demo/demo.exe",  // the header names what ran
		"hello from stdout",     // the program's own output
		"Library FOO.dll",       // and its errors
		"exited with status 53", // and how it ended
	} {
		if !strings.Contains(log, want) {
			t.Errorf("log missing %q:\n%s", want, log)
		}
	}
}

func TestSuccessfulRunIsNotReportedAsVanished(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OLIVER_LOG_DIR", dir)

	script := filepath.Join(dir, "ok")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := &prefix.Manager{Root: t.TempDir()}
	p := m.For("", "ok")
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}

	res, err := Execute(&Plan{Launch: Step{Argv: []string{script}, Dir: dir}}, p, LogPathFor("ok"), func(string, ...any) {})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Vanished || res.ExitCode != 0 {
		t.Errorf("a clean exit must not be flagged: %+v", res)
	}
}

func TestDiagnosePicksOutRealErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run.log")
	content := strings.Join([]string{
		"oliver run 2026-01-01T00:00:00Z",
		"fixme:some:noise this is chatter",
		"trace:another:thing more chatter",
		"err:module:import_dll Library MSVCP140.dll not found",
		"fixme:more:noise",
		"wine: failed to initialise the prefix",
		"ordinary program output",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Diagnose(path, 10)
	if len(got) != 2 {
		t.Fatalf("Diagnose = %v, want the two error lines", got)
	}
	if !strings.Contains(got[0], "MSVCP140") || !strings.Contains(got[1], "failed to initialise") {
		t.Errorf("Diagnose picked the wrong lines: %v", got)
	}
}

func TestDiagnoseFallsBackToTheTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quiet.log")
	if err := os.WriteFile(path, []byte("line one\nline two\nline three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Diagnose(path, 2)
	if len(got) != 2 || got[1] != "line three" {
		t.Errorf("Diagnose = %v, want the last two lines", got)
	}
}

func TestTailIgnoresBlankLinesAndMissingFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.log")
	if err := os.WriteFile(path, []byte("a\n\n\nb\n\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Tail(path, 2)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Errorf("Tail = %v, want [b c]", got)
	}
	if Tail(filepath.Join(dir, "absent.log"), 5) != nil {
		t.Error("Tail on a missing file should be empty")
	}
}

func TestLatestLogPicksTheNewest(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OLIVER_LOG_DIR", dir)

	if _, err := LatestLog(); err == nil {
		t.Error("expected an error when nothing has been logged")
	}

	for _, name := range []string{"old", "new"} {
		if err := os.WriteFile(LogPathFor(name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Make "new" unambiguously the most recent.
	older := filepath.Join(dir, "old.log")
	if err := os.Chtimes(older, timeZero(), timeZero()); err != nil {
		t.Fatal(err)
	}

	got, err := LatestLog()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "new.log" {
		t.Errorf("LatestLog = %s, want new.log", filepath.Base(got))
	}

	logs, err := Logs()
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Errorf("Logs() = %v, want 2", logs)
	}
}

func TestPlanRecordsTheProgramItLaunches(t *testing.T) {
	plan, err := Build(Options{
		Info: info(pe.KindPE, pe.ArchX64), Backend: wineBackend(), Prefix: testPrefix(t, true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Program != "/games/demo/demo.exe" {
		t.Errorf("Program = %q, want the executable path", plan.Program)
	}
}

func timeZero() time.Time { return time.Unix(1, 0) }

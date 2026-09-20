package ui

import (
	"bytes"
	"strings"
	"testing"
)

func newTestUI(input string) (*UI, *bytes.Buffer, *bytes.Buffer) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	return &UI{Out: out, Err: errOut, In: strings.NewReader(input), Interactive: true}, out, errOut
}

func TestAskReturnsTheAnswerAndTheQuestionGoesToStderr(t *testing.T) {
	u, out, errOut := newTestUI("  sdb  \n")
	answer, ok := u.Ask("Which drive?")
	if !ok || answer != "sdb" {
		t.Errorf("Ask = %q, %v", answer, ok)
	}
	if !strings.Contains(errOut.String(), "Which drive?") {
		t.Errorf("the question did not go to stderr: %q", errOut.String())
	}
	if out.Len() != 0 {
		t.Errorf("the question contaminated stdout: %q", out.String())
	}
}

func TestQuestionsAreAnsweredInOrderAcrossViews(t *testing.T) {
	// A view writing to stderr must share the prompt reader: two readers over
	// one input each buffer what the other is waiting for.
	u, _, _ := newTestUI("2\nsdc\n")

	first, _ := u.OnStderr().Ask("Which drive?")
	second, ok := u.Ask("Type the name:")
	if first != "2" || second != "sdc" || !ok {
		t.Errorf("answers came back as %q then %q (ok=%v), want 2 then sdc", first, second, ok)
	}
}

func TestOnStderrMovesTablesOffStandardOutput(t *testing.T) {
	u, out, errOut := newTestUI("")
	view := u.OnStderr()
	table := view.NewTable("DEVICE")
	table.Row("/dev/sdb")
	table.Render()
	view.Printf("pick one")

	if out.Len() != 0 {
		t.Errorf("stdout should be untouched, got %q", out.String())
	}
	if !strings.Contains(errOut.String(), "/dev/sdb") || !strings.Contains(errOut.String(), "pick one") {
		t.Errorf("stderr = %q", errOut.String())
	}
}

func TestConfirmWordAcceptsOnlyThatWord(t *testing.T) {
	for _, c := range []struct {
		typed string
		want  bool
	}{
		{"sdb", true},
		{"SDB", true},
		{"yes", false},
		{"", false},
		{"sdb1", false},
	} {
		u, _, _ := newTestUI(c.typed + "\n")
		if got := u.ConfirmWord("Type sdb:", "sdb"); got != c.want {
			t.Errorf("typing %q gave %v, want %v", c.typed, got, c.want)
		}
	}
}

func TestPromptsFallBackWhenThereIsNobodyToAsk(t *testing.T) {
	u := &UI{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, In: strings.NewReader("y\n")}
	if _, ok := u.Ask("Which drive?"); ok {
		t.Error("asked a question with no terminal attached")
	}
	if !u.Confirm("Carry on?", true) || u.Confirm("Carry on?", false) {
		t.Error("Confirm should return its default without a terminal")
	}
	if u.ConfirmWord("Type sdb:", "sdb") {
		t.Error("ConfirmWord must never agree on behalf of a user who is not there")
	}
}

func TestEndOfInputIsNotAnAnswer(t *testing.T) {
	u, _, _ := newTestUI("")
	if _, ok := u.Ask("Which drive?"); ok {
		t.Error("closed input was treated as an answer")
	}
}

func TestProgressWithoutATerminalWritesOneLinePerTenth(t *testing.T) {
	u := &UI{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	bar := u.NewBar("writing")
	for done := int64(0); done <= 100; done++ {
		bar.Update(done, 100, "detail")
	}
	bar.Done("finished")

	errOut := u.Err.(*bytes.Buffer).String()
	lines := strings.Count(errOut, "\n")
	if lines != 12 { // 0% through 100% inclusive, plus the summary
		t.Errorf("wrote %d lines, want one per tenth plus a summary:\n%s", lines, errOut)
	}
	if strings.Contains(errOut, "\r") {
		t.Error("a log without a terminal should hold no carriage returns")
	}
	if !strings.HasSuffix(errOut, "writing finished\n") {
		t.Errorf("the last line should be the summary: %q", errOut)
	}
}

func TestProgressOnATerminalRedrawsOneLine(t *testing.T) {
	u, _, errOut := newTestUI("")
	bar := u.NewBar("writing")
	bar.Update(50, 100, "half")
	bar.Done("finished")

	if !strings.Contains(errOut.String(), "\r") {
		t.Error("a terminal bar should redraw in place")
	}
	if strings.Count(errOut.String(), "\n") != 1 {
		t.Errorf("a bar should leave exactly one line behind: %q", errOut.String())
	}
}

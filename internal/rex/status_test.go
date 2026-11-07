package rex

import (
	"encoding/base64"
	"slices"
	"testing"
	"time"
)

func TestStatusScanner(t *testing.T) {
	msg := base64.StdEncoding.EncodeToString([]byte("Bash: rm -rf build"))
	out := "hi \x1b]0;title\x07 \x1b[1m\x1b]7501;?\x1b\\ \x1b]7501;state=working:app=claude-code\x07" +
		"\x1b]75012;no\x07 \x1b]7501;state=blocked:app=claude-code:kind=permission:msg=" + msg + "\x1b\\" +
		"\x1b]7501;cut\x1b[0m \x1b]7501;state=clear:id=task-1\x07"
	want := []string{"?", "state=working:app=claude-code", "state=blocked:app=claude-code:kind=permission:msg=" + msg, "state=clear:id=task-1"}
	// Every way of cutting it in two reads finds the same.
	for cut := range len(out) + 1 {
		var sc statusScanner
		var got []string
		found := func(p string) { got = append(got, p) }
		sc.scan([]byte(out[:cut]), found)
		sc.scan([]byte(out[cut:]), found)
		if !slices.Equal(got, want) {
			t.Fatalf("cut at %d: found %q", cut, got)
		}
	}
}

func TestParseStatus(t *testing.T) {
	now := time.Now()
	msg := base64.StdEncoding.EncodeToString([]byte("Bash: rm -rf build"))
	st, ok := parseStatus("state=blocked:app=claude-code:kind=permission:progress=40:msg="+msg, now)
	if !ok || st == nil || st.State != "blocked" || st.Kind != "permission" || st.Msg != "Bash: rm -rf build" || st.App != "claude-code" {
		t.Errorf("blocked: %+v %v", st, ok)
	}
	if st, ok := parseStatus("state=clear", now); !ok || st != nil {
		t.Errorf("clear: %+v %v", st, ok)
	}
	// The items of subagents, which carry an id, are not the agent's.
	if _, ok := parseStatus("state=working:app=claude-code:id=task-1", now); ok {
		t.Error("took a subagent's state")
	}
	if _, ok := parseStatus("state=clear:id=task-1", now); ok {
		t.Error("took a subagent's clear")
	}
}

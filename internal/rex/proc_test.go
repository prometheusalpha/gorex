package rex

import (
	"strings"
	"testing"
)

func TestProgramName(t *testing.T) {
	for _, c := range []struct {
		p    procInfo
		want string
	}{
		{procInfo{name: "2.1.319", args: []string{"claude"}}, "claude"},
		{procInfo{name: "2.1.319", args: []string{"/Users/a/.local/share/claude/versions/2.1.319"}}, "claude"},
		{procInfo{name: "node", args: []string{"node", "/usr/local/lib/node_modules/@openai/codex/bin/codex.js"}}, "codex"},
		{procInfo{name: "bun", args: []string{"bun", "/Users/a/.bun/bin/omp"}}, "omp"},
		{procInfo{name: "zsh", args: []string{"-zsh"}}, "zsh"},
	} {
		if got := c.p.programName(); got != c.want {
			t.Errorf("%+v: %q, not %q", c.p, got, c.want)
		}
	}
}

// TestSessionEnv keeps what Claude Code sets for its children out of the
// sessions of a server it started, and the user's settings of it in.
func TestSessionEnv(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_CHILD_SESSION", "1")
	t.Setenv("CLAUDE_CODE_SESSION_KIND", "bg")
	t.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
	env := strings.Join(sessionEnv("s1", nil), "\n")
	for _, k := range []string{"CLAUDECODE=", "CLAUDE_CODE_CHILD_SESSION=", "CLAUDE_CODE_SESSION_KIND="} {
		if strings.Contains(env, "\n"+k) || strings.HasPrefix(env, k) {
			t.Errorf("%s reaches the session", k)
		}
	}
	if !strings.Contains(env, "CLAUDE_CODE_USE_BEDROCK=1") {
		t.Error("a setting of Claude Code was dropped")
	}
}

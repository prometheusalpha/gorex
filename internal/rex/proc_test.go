package rex

import "testing"

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

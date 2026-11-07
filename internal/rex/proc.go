package rex

import (
	"path/filepath"
	"strings"
)

// procInfo is what inspect finds of a process.
type procInfo struct {
	name string
	args []string
	dir  string
}

// interpreters run a script named by their first argument, which names
// the program better than they do: node …/codex.js is codex.
var interpreters = map[string]bool{
	"node": true, "bun": true, "deno": true, "python": true, "python3": true,
	"ruby": true, "perl": true, "php": true, "sh": true, "bash": true,
}

// programName returns the name a process is best known by.
func (p procInfo) programName() string {
	name := strings.TrimPrefix(p.name, "-")
	if len(p.args) > 0 {
		base := strings.TrimPrefix(filepath.Base(p.args[0]), "-")
		if ext := filepath.Ext(base); strings.EqualFold(ext, ".exe") {
			base = strings.TrimSuffix(base, ext)
		}
		if base != "" && len(base) > len(name) && strings.HasPrefix(base, name) {
			// p_comm is cut at 16 bytes.
			name = base
		}
	}
	// Claude Code runs as …/claude/versions/2.1.319, whose name is its
	// version: the command it was typed as, or the directory of its
	// versions, names it.
	if isVersion(name) && len(p.args) > 0 {
		if base := filepath.Base(p.args[0]); !isVersion(base) {
			return base
		}
		if dir := filepath.Dir(p.args[0]); filepath.Base(dir) == "versions" {
			return filepath.Base(filepath.Dir(dir))
		}
	}
	if interpreters[name] && len(p.args) > 1 {
		for _, a := range p.args[1:] {
			if strings.HasPrefix(a, "-") {
				continue
			}
			script := filepath.Base(a)
			if ext := filepath.Ext(script); ext == ".js" || ext == ".mjs" || ext == ".cjs" || ext == ".ts" || ext == ".py" || ext == ".rb" {
				script = strings.TrimSuffix(script, ext)
			}
			// npm runs …\npm\bin\npm-cli.js where no link names it npm.
			script = strings.TrimSuffix(script, "-cli")
			// node …/bin/codex.js; but node demo/snake.mjs stays node.
			if strings.Contains(filepath.ToSlash(a), "/bin/") || strings.Contains(a, "node_modules") {
				return script
			}
			break
		}
	}
	return name
}

// isVersion reports whether a name is a version, as 2.1.319.
func isVersion(s string) bool {
	return s != "" && strings.Trim(s, "0123456789.") == "" && strings.Contains(s, ".")
}

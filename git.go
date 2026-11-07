package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// gitState is the branch of a directory and how far it is from its
// upstream, as the sidebar shows them.
type gitState struct {
	Branch        string // "" outside a repository
	Ahead, Behind int
	at            time.Time
	busy          bool
}

// gitEvery is how stale a directory's state may get. It asks git rather
// than watching .git, which a checkout from a shell changes behind us.
const gitEvery = 3 * time.Second

// gitOf returns what is known of dir, and asks git again in the
// background when it is stale.
func (a *App) gitOf(dir string) gitState {
	if dir == "" {
		return gitState{}
	}
	if a.git == nil {
		a.git = map[string]gitState{}
	}
	s := a.git[dir]
	if s.busy || time.Since(s.at) < gitEvery {
		return s
	}
	s.busy = true
	a.git[dir] = s
	win := a.win
	go func() {
		n := readGit(dir)
		n.at = time.Now()
		if win == nil { // a view without a window, as in tests
			a.post(func() { a.git[dir] = n })
			return
		}
		win.Update(func() { a.git[dir] = n })
	}()
	return s
}

// readGit counts against the upstream as the last fetch left it: fetching
// here would go to the network every few seconds.
func readGit(dir string) gitState {
	var s gitState
	head, ok := git(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if !ok {
		return s
	}
	s.Branch = head
	if head == "HEAD" { // detached
		if sha, ok := git(dir, "rev-parse", "--short", "HEAD"); ok {
			s.Branch = sha
		}
		return s
	}
	if counts, ok := git(dir, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); ok {
		if f := strings.Fields(counts); len(f) == 2 {
			s.Ahead, _ = strconv.Atoi(f[0])
			s.Behind, _ = strconv.Atoi(f[1])
		}
	}
	return s
}

func git(dir string, args ...string) (string, bool) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	// Asking every few seconds must not take index.lock from a git the
	// user runs.
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	hideConsole(cmd)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err == nil
}

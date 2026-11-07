//go:build darwin || linux

package rex

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestAgentStatus runs an agent that probes for OSC 7501 as Claude Code
// does: the server answers it, and keeps the state it then reports, until
// its shell is back.
func TestAgentStatus(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "rex")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("GOREX_DIR", dir)
	go Serve()
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(filepath.Join(dir, "server.sock")); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	c, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Shutdown()

	answer := filepath.Join(dir, "answer")
	agent := filepath.Join(dir, "agent.sh")
	script := `stty raw -echo
printf '\033]7501;?\033\\'
dd bs=1 count=` + strconv.Itoa(len(statusReply)) + ` 2>/dev/null > ` + answer + `
printf '\033]7501;state=blocked:app=claude-code:kind=question\033\\'
sleep 2
`
	if err := os.WriteFile(agent, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := c.Create(CreateOptions{Command: []string{"/bin/sh"}, Dir: dir, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	s := c.Stream(info.ID, 80, 24)
	defer s.Close()
	s.Write([]byte("/bin/sh " + agent + "\n"))

	state := func() *AgentStatus {
		list, _ := c.List()
		for _, in := range list {
			if in.ID == info.ID {
				return in.Agent
			}
		}
		return nil
	}
	deadline := time.Now().Add(5 * time.Second)
	for st := state(); st == nil || st.State != "blocked"; st = state() {
		if time.Now().After(deadline) {
			b, _ := os.ReadFile(answer)
			t.Fatalf("state %+v; the agent read %q", st, b)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if b, _ := os.ReadFile(answer); string(b) != string(statusReply) {
		t.Errorf("the probe's answer %q", b)
	}
	if st := state(); st.Kind != "question" || st.App != "claude-code" {
		t.Errorf("state %+v", st)
	}
	// Its shell back, the agent's state goes with it.
	for st := state(); st != nil; st = state() {
		if time.Now().After(deadline.Add(3 * time.Second)) {
			t.Fatalf("state %+v after the agent ended", st)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

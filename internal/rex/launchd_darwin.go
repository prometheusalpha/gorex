package rex

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// labelEnv tells the server the label of the launchd job that runs it.
// Sessions drop it with the other GOREX_ variables, so a server a test
// starts in a session never takes the job for its own.
const labelEnv = "GOREX_SERVER_LABEL"

// spawn has launchd start the server, as a job of its own. A child of the
// app stays in the app's coalition even in a session of its own: once
// that app has quit, when a GUI app a session's shell ran quits too,
// loginwindow takes the server for that app's and ends it with SIGTERM.
// A job has a coalition of its own.
func spawn(exe, logPath string) error {
	sum := sha256.Sum256([]byte(Dir()))
	label := "dev.gorex.server." + hex.EncodeToString(sum[:6])
	cwd, _ := os.Getwd()
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
`)
	str := func(s string) {
		b.WriteString("<string>")
		xml.EscapeText(&b, []byte(s))
		b.WriteString("</string>\n")
	}
	key := func(k string) {
		b.WriteString("<key>")
		xml.EscapeText(&b, []byte(k))
		b.WriteString("</key>")
	}
	key("Label")
	str(label)
	key("ProgramArguments")
	b.WriteString("<array>")
	str(exe)
	str("-server")
	b.WriteString("</array>\n")
	// The server and its shells get the app's environment, as when the
	// app started it itself.
	key("EnvironmentVariables")
	b.WriteString("<dict>\n")
	for _, kv := range append(os.Environ(), labelEnv+"="+label) {
		if k, v, ok := strings.Cut(kv, "="); ok && k != "" {
			key(k)
			str(v)
		}
	}
	b.WriteString("</dict>\n")
	if cwd != "" {
		key("WorkingDirectory")
		str(cwd)
	}
	key("StandardInPath")
	str(os.DevNull)
	key("StandardOutPath")
	str(logPath)
	key("StandardErrorPath")
	str(logPath)
	key("RunAtLoad")
	b.WriteString("<true/>\n")
	// Run once: the server exits when idle, and on purpose.
	key("KeepAlive")
	b.WriteString("<false/>\n")
	// Not throttled as a background daemon: the shells run what the user
	// waits on.
	key("ProcessType")
	str("Interactive")
	// What sessions left running, as a program started with nohup, stays
	// when the server exits.
	key("AbandonProcessGroup")
	b.WriteString("<true/>\n")
	b.WriteString("</dict></plist>\n")
	plist := filepath.Join(Dir(), "server.plist")
	if err := os.WriteFile(plist, b.Bytes(), 0o600); err != nil {
		return err
	}
	domain := "gui/" + strconv.Itoa(os.Getuid())
	bootstrap := func() error {
		out, err := exec.Command("launchctl", "bootstrap", domain, plist).CombinedOutput()
		if err != nil {
			return fmt.Errorf("launchctl bootstrap: %v: %s", err, bytes.TrimSpace(out))
		}
		return nil
	}
	if bootstrap() == nil {
		return nil
	}
	// The job is loaded already. One with a process runs a server that
	// has yet to listen, as another app's Connect started: it will.
	// Otherwise it is the job of a server that died without ending it.
	job := domain + "/" + label
	out, _ := exec.Command("launchctl", "print", job).Output()
	if bytes.Contains(out, []byte("\tpid = ")) {
		return nil
	}
	exec.Command("launchctl", "bootout", job).Run()
	return bootstrap()
}

// EndJob removes the launchd job that runs this server, if one does, so
// that it does not stay loaded once the server exits. Call it just before
// exiting: launchd ends the job's process with SIGTERM, which it ignores.
func EndJob() {
	label := os.Getenv(labelEnv)
	if label == "" {
		return
	}
	signal.Ignore(syscall.SIGTERM)
	exec.Command("launchctl", "remove", label).Run()
}

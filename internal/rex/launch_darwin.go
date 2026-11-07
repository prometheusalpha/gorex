package rex

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// spawnOpen starts the server as an instance of the app of its own, which
// LaunchServices starts. macOS ties all the app starts to it, whatever
// their session, parent or responsibility, and even the launchd jobs it
// submits: loginwindow and launchd end them when the app quits. What
// LaunchServices starts is not the app's.
//
// The server gets LaunchServices' environment, as the app from the Dock,
// with GOREX_DIR.
func spawnOpen(exe, logPath string) error {
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(bundle, ".app") {
		return errors.ErrUnsupported // a test's binary
	}
	return exec.Command("/usr/bin/open", "-n", "-g", "-j", "-a", bundle,
		"--env", "GOREX_DIR="+Dir(), "--stdout", logPath, "--stderr", logPath,
		"--args", "-server").Run()
}

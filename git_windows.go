package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// hideConsole keeps git from flashing a console window up every few
// seconds, as a GUI app has none for it to share.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

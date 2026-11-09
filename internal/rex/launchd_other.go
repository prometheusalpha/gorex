//go:build linux || windows

package rex

func spawn(exe, logPath string) error { return spawnProcess(exe, logPath) }

// EndJob does nothing: only on macOS does launchd start the server.
func EndJob() {}

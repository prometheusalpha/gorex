//go:build linux || windows

package rex

import "errors"

// spawnOpen starts the server through LaunchServices; on macOS only.
func spawnOpen(exe, logPath string) error { return errors.ErrUnsupported }

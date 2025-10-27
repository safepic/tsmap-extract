// SPDX-License-Identifier: LGPL-3.0-or-later
// Author: Michel Prunet - Safe Pic Technologies
package tsmap

import (
	"os"
	"runtime"
)

var (
	// Couleurs ANSI si TTY Linux/macOS
	useColor = func() bool {
		fi, err := os.Stdout.Stat()
		return err == nil && (fi.Mode()&os.ModeCharDevice) != 0 &&
			(runtime.GOOS == "linux" || runtime.GOOS == "darwin")
	}()
	cRed = ansi("\033[31m")
	cGrn = ansi("\033[32m")
	cYel = ansi("\033[33m")
	cCyn = ansi("\033[36m")
	cRst = "\033[0m"
)

func ansi(code string) string {
	if useColor {
		return code
	}
	return ""
}

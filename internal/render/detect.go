package render

import (
	"os/exec"
	"runtime"
	"strings"
)

func DetectLightMode() bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err != nil {
		return true
	}

	return !strings.Contains(strings.ToLower(string(out)), "dark")
}

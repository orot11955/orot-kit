package detect

import (
	"os"
	"runtime"
)

type SystemInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	Home     string `json:"home"`
	Shell    string `json:"shell"`
}

func System() SystemInfo {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	return SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
		Home:     home,
		Shell:    os.Getenv("SHELL"),
	}
}

func IsLinux() bool {
	return runtime.GOOS == "linux"
}

func IsDarwin() bool {
	return runtime.GOOS == "darwin"
}

package runtime

import (
	"runtime"
	"strings"
)

type DownloadPlan struct {
	Name    string
	Version string
	OS      string
	Arch    string
	URL     string
}

func NewDownloadPlan(baseURL string, name string, version string) DownloadPlan {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	return DownloadPlan{
		Name: name, Version: version, OS: osName, Arch: arch,
		URL: RuntimeDownloadURL(baseURL, name, version, osName, arch),
	}
}

func RuntimeDownloadURL(baseURL string, name string, version string, osName string, arch string) string {
	return strings.TrimRight(baseURL, "/") + "/" + name + "/" + version + "/" + osName + "/" + arch
}

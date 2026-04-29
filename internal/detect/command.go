package detect

import "os/exec"

func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func FirstCommand(names ...string) string {
	for _, name := range names {
		if CommandExists(name) {
			return name
		}
	}
	return ""
}

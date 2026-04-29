package detect

type RuntimeSpec struct {
	Name        string
	Command     string
	VersionArgs []string
	PathCommand string
}

func RuntimeSpecs() map[string]RuntimeSpec {
	return map[string]RuntimeSpec{
		"node":   {Name: "node", Command: "node", VersionArgs: []string{"-v"}, PathCommand: "which"},
		"go":     {Name: "go", Command: "go", VersionArgs: []string{"version"}, PathCommand: "which"},
		"python": {Name: "python", Command: "python", VersionArgs: []string{"--version"}, PathCommand: "which"},
		"java":   {Name: "java", Command: "java", VersionArgs: []string{"-version"}, PathCommand: "which"},
	}
}

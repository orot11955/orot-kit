package runtime

var Supported = []string{"node", "go", "python", "java"}

type Spec struct {
	Name         string
	Command      string
	VersionArgs  []string
	ShimNames    []string
	SeedVersions []string
}

func Specs() map[string]Spec {
	return map[string]Spec{
		"node": {
			Name: "node", Command: "node", VersionArgs: []string{"-v"},
			ShimNames:    []string{"node", "npm", "npx", "corepack"},
			SeedVersions: []string{"20.11.1", "22.3.0"},
		},
		"go": {
			Name: "go", Command: "go", VersionArgs: []string{"version"},
			ShimNames:    []string{"go", "gofmt"},
			SeedVersions: []string{"1.22.4", "1.23.0"},
		},
		"python": {
			Name: "python", Command: "python", VersionArgs: []string{"--version"},
			ShimNames:    []string{"python", "python3", "pip", "pip3"},
			SeedVersions: []string{"3.11.9", "3.12.3"},
		},
		"java": {
			Name: "java", Command: "java", VersionArgs: []string{"-version"},
			ShimNames:    []string{"java", "javac", "jar"},
			SeedVersions: []string{"17", "21"},
		},
	}
}

func SupportedRuntime(name string) bool {
	_, ok := Specs()[name]
	return ok
}

package registry

type Runtime struct {
	Name     string
	Homepage string
}

var Runtimes = []Runtime{
	{Name: "node", Homepage: "https://nodejs.org"},
	{Name: "go", Homepage: "https://go.dev"},
	{Name: "python", Homepage: "https://www.python.org"},
	{Name: "java", Homepage: "https://adoptium.net"},
}

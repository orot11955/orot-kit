package builder

type Choice struct {
	Label string
	Value string
}

type State struct {
	Title    string
	Detected []string
	Fields   map[string]string
}

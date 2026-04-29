package output

type Section struct {
	Name string   `json:"name"`
	Text string   `json:"text,omitempty"`
	Rows []string `json:"rows,omitempty"`
}

type Result struct {
	Title    string    `json:"title,omitempty"`
	Command  []string  `json:"command,omitempty"`
	Summary  string    `json:"summary,omitempty"`
	Result   string    `json:"result,omitempty"`
	Sections []Section `json:"sections,omitempty"`
	Hint     []string  `json:"hint,omitempty"`
}

package builder

import "strings"

func (p Prompt) Confirm(label string, defaultYes bool) (bool, error) {
	suffix := "[Y/n]"
	defaultValue := "y"
	if !defaultYes {
		suffix = "[y/N]"
		defaultValue = "n"
	}
	value, err := p.Ask(label+" "+suffix, defaultValue)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return defaultYes, nil
	}
}

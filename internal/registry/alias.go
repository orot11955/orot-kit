package registry

var DefaultAliases = map[string]string{
	".":     "ls -al .",
	"..":    "ls -al ..",
	"...":   "ls -al ../..",
	"ports": "port",
}

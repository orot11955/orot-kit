package config

type Config struct {
	Language string             `json:"language" yaml:"language"`
	Output   OutputConfig       `json:"output" yaml:"output"`
	Server   ServerConfig       `json:"server" yaml:"server"`
	SSH      SSHConfig          `json:"ssh" yaml:"ssh"`
	Services map[string]Service `json:"services" yaml:"services"`
}

type OutputConfig struct {
	ShowCommand bool   `json:"show_command" yaml:"show_command"`
	Format      string `json:"format" yaml:"format"`
}

type ServerConfig struct {
	InstallBaseURL string `json:"install_base_url" yaml:"install_base_url"`
	RuntimeBaseURL string `json:"runtime_base_url" yaml:"runtime_base_url"`
}

type SSHConfig struct {
	Hosts map[string]SSHHost `json:"hosts" yaml:"hosts"`
}

type SSHHost struct {
	Host         string `json:"host" yaml:"host"`
	User         string `json:"user" yaml:"user"`
	Port         int    `json:"port" yaml:"port"`
	IdentityFile string `json:"identity_file" yaml:"identity_file"`
}

type Service struct {
	Type string `json:"type" yaml:"type"`
	Name string `json:"name" yaml:"name"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

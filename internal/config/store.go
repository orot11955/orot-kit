package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".kit/config.yaml"
	}
	return filepath.Join(home, ".kit", "config.yaml")
}

func Default() Config {
	return Config{
		Language: "ko",
		Output: OutputConfig{
			ShowCommand: true,
			Format:      "text",
		},
		Services: map[string]Service{},
		SSH: SSHConfig{
			Hosts: map[string]SSHHost{},
		},
	}
}

func Load() (Config, error) {
	return LoadPath(DefaultPath())
}

func LoadPath(path string) (Config, error) {
	cfg := Default()
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	section := ""
	subsection := ""
	currentHost := ""
	currentService := ""
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, "language:") {
			cfg.Language = cleanYAMLValue(strings.TrimSpace(strings.TrimPrefix(line, "language:")))
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			subsection = ""
			currentHost = ""
			currentService = ""
			continue
		}
		if section == "server" && strings.HasPrefix(line, "  ") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := cleanYAMLValue(strings.TrimSpace(parts[1]))
			switch key {
			case "install_base_url":
				cfg.Server.InstallBaseURL = value
			case "runtime_base_url":
				cfg.Server.RuntimeBaseURL = value
			}
			continue
		}
		if section == "ssh" && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			subsection = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if section == "ssh" && subsection == "hosts" && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") && strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
			currentHost = cleanYAMLValue(strings.TrimSuffix(trimmed, ":"))
			if cfg.SSH.Hosts == nil {
				cfg.SSH.Hosts = map[string]SSHHost{}
			}
			cfg.SSH.Hosts[currentHost] = SSHHost{Port: 22}
			continue
		}
		if section == "ssh" && currentHost != "" && strings.HasPrefix(line, "      ") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := cleanYAMLValue(strings.TrimSpace(parts[1]))
			host := cfg.SSH.Hosts[currentHost]
			switch key {
			case "host":
				host.Host = value
			case "user":
				host.User = value
			case "port":
				if port, err := strconv.Atoi(value); err == nil {
					host.Port = port
				}
			case "identity_file":
				host.IdentityFile = value
			}
			cfg.SSH.Hosts[currentHost] = host
			continue
		}
		if section == "services" && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			currentService = cleanYAMLValue(strings.TrimSuffix(trimmed, ":"))
			if cfg.Services == nil {
				cfg.Services = map[string]Service{}
			}
			cfg.Services[currentService] = Service{}
			continue
		}
		if section == "services" && currentService != "" && strings.HasPrefix(line, "    ") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := cleanYAMLValue(strings.TrimSpace(parts[1]))
			service := cfg.Services[currentService]
			switch key {
			case "type":
				service.Type = value
			case "name":
				service.Name = value
			case "path":
				service.Path = value
			}
			cfg.Services[currentService] = service
		}
	}
	return cfg, scanner.Err()
}

func Save(cfg Config) error {
	return SavePath(DefaultPath(), cfg)
}

func SavePath(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var builder strings.Builder
	if cfg.Language == "" {
		cfg.Language = "ko"
	}
	fmt.Fprintf(&builder, "language: %s\n\n", quoteYAMLValue(cfg.Language))
	builder.WriteString("output:\n")
	fmt.Fprintf(&builder, "  show_command: %t\n", cfg.Output.ShowCommand)
	if cfg.Output.Format == "" {
		cfg.Output.Format = "text"
	}
	fmt.Fprintf(&builder, "  format: %s\n\n", quoteYAMLValue(cfg.Output.Format))
	builder.WriteString("server:\n")
	if cfg.Server.InstallBaseURL != "" {
		fmt.Fprintf(&builder, "  install_base_url: %s\n", quoteYAMLValue(cfg.Server.InstallBaseURL))
	}
	if cfg.Server.RuntimeBaseURL != "" {
		fmt.Fprintf(&builder, "  runtime_base_url: %s\n", quoteYAMLValue(cfg.Server.RuntimeBaseURL))
	}
	builder.WriteString("\n")
	builder.WriteString("ssh:\n")
	builder.WriteString("  hosts:\n")
	names := SSHHostNames(cfg)
	for _, name := range names {
		host := cfg.SSH.Hosts[name]
		if host.Port == 0 {
			host.Port = 22
		}
		fmt.Fprintf(&builder, "    %s:\n", quoteYAMLKey(name))
		fmt.Fprintf(&builder, "      host: %s\n", quoteYAMLValue(host.Host))
		fmt.Fprintf(&builder, "      user: %s\n", quoteYAMLValue(host.User))
		fmt.Fprintf(&builder, "      port: %d\n", host.Port)
		if host.IdentityFile != "" {
			fmt.Fprintf(&builder, "      identity_file: %s\n", quoteYAMLValue(host.IdentityFile))
		}
	}
	builder.WriteString("\nservices:\n")
	serviceNames := ServiceNames(cfg)
	for _, name := range serviceNames {
		service := cfg.Services[name]
		fmt.Fprintf(&builder, "  %s:\n", quoteYAMLKey(name))
		fmt.Fprintf(&builder, "    type: %s\n", quoteYAMLValue(service.Type))
		fmt.Fprintf(&builder, "    name: %s\n", quoteYAMLValue(service.Name))
		if service.Path != "" {
			fmt.Fprintf(&builder, "    path: %s\n", quoteYAMLValue(service.Path))
		}
	}
	return os.WriteFile(path, []byte(builder.String()), 0o600)
}

func UpsertSSHHost(name string, host SSHHost) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if cfg.SSH.Hosts == nil {
		cfg.SSH.Hosts = map[string]SSHHost{}
	}
	if host.Port == 0 {
		host.Port = 22
	}
	cfg.SSH.Hosts[name] = host
	return Save(cfg)
}

func SSHHostNames(cfg Config) []string {
	names := make([]string, 0, len(cfg.SSH.Hosts))
	for name := range cfg.SSH.Hosts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ServiceNames(cfg Config) []string {
	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func cleanYAMLValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if value[0] == '"' && value[len(value)-1] == '"' {
			unquoted, err := strconv.Unquote(value)
			if err == nil {
				return unquoted
			}
		}
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			return strings.Trim(value, "'")
		}
	}
	return value
}

func quoteYAMLKey(value string) string {
	if value == "" || strings.ContainsAny(value, " :#{}[],&*?|-<>=!%@\\\"'") {
		return strconv.Quote(value)
	}
	return value
}

func quoteYAMLValue(value string) string {
	return strconv.Quote(value)
}

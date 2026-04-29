package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadSSHHost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.SSH.Hosts["orbit"] = SSHHost{
		Host:         "10.0.0.10",
		User:         "deploy",
		Port:         2222,
		IdentityFile: "~/.ssh/orbit",
	}
	if err := SavePath(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	host := loaded.SSH.Hosts["orbit"]
	if host.Host != "10.0.0.10" || host.User != "deploy" || host.Port != 2222 || host.IdentityFile != "~/.ssh/orbit" {
		t.Fatalf("loaded host mismatch: %#v", host)
	}
}

func TestSaveLoadService(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.Services["orot"] = Service{
		Type: "docker-compose",
		Name: "web",
		Path: "/srv/orot",
	}
	if err := SavePath(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	service := loaded.Services["orot"]
	if service.Type != "docker-compose" || service.Name != "web" || service.Path != "/srv/orot" {
		t.Fatalf("loaded service mismatch: %#v", service)
	}
}

func TestSaveLoadServerRuntimeBaseURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.Server.RuntimeBaseURL = "https://kit.example.com/runtime"
	if err := SavePath(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server.RuntimeBaseURL != "https://kit.example.com/runtime" {
		t.Fatalf("runtime_base_url = %q", loaded.Server.RuntimeBaseURL)
	}
}

package cmd

import (
	"testing"

	"github.com/orot-dev/orot-kit/internal/config"
)

func TestServiceCommandsConfiguredCompose(t *testing.T) {
	cfg := config.Default()
	cfg.Services["orot"] = config.Service{Type: "docker-compose", Name: "web", Path: "/srv/orot"}
	commands, summary, err := serviceCommands("status", "orot", serviceOptions{tail: 50}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 {
		t.Fatalf("command count = %d", len(commands))
	}
	if got := commands[0].String(); got != "docker compose --project-directory /srv/orot ps web" {
		t.Fatalf("command = %q", got)
	}
	if summary == "" {
		t.Fatal("summary is empty")
	}
}

func TestParseServiceArgs(t *testing.T) {
	action, name := parseServiceArgs([]string{"nginx", "restart"})
	if action != "restart" || name != "nginx" {
		t.Fatalf("action/name = %s/%s", action, name)
	}
	action, name = parseServiceArgs([]string{"logs", "nginx"})
	if action != "logs" || name != "nginx" {
		t.Fatalf("action/name = %s/%s", action, name)
	}
}

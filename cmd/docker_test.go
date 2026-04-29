package cmd

import "testing"

func TestDockerCleanCommand(t *testing.T) {
	command, summary, err := dockerCleanCommand("volumes")
	if err != nil {
		t.Fatal(err)
	}
	if got := command.String(); got != "docker volume prune" {
		t.Fatalf("command = %q", got)
	}
	if summary == "" {
		t.Fatal("summary is empty")
	}
}

func TestDockerComposeBaseArgs(t *testing.T) {
	args := dockerComposeBaseArgs(dockerOptions{projectDir: "/srv/app", composeFile: "/srv/app/docker-compose.yml"})
	got := dockerComposeRunner(args).String()
	want := "docker compose --project-directory /srv/app -f /srv/app/docker-compose.yml"
	if got != want {
		t.Fatalf("compose command = %q, want %q", got, want)
	}
}

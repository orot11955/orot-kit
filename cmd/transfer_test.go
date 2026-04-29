package cmd

import (
	"strings"
	"testing"

	"github.com/orot-dev/orot-kit/internal/config"
)

func TestBuildTransferCommandUsesConfiguredSSHHost(t *testing.T) {
	cfg := config.Default()
	cfg.SSH.Hosts["orbit"] = config.SSHHost{
		Host:         "10.0.0.10",
		User:         "deploy",
		Port:         2222,
		IdentityFile: "~/.ssh/orbit",
	}
	command, err := buildTransferCommand("send", transferOptions{
		server: "orbit",
		local:  "./dist",
		remote: "/srv/app",
		method: "rsync",
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	got := command.String()
	if !strings.Contains(got, "deploy@10.0.0.10:/srv/app") {
		t.Fatalf("transfer command target missing: %q", got)
	}
	if !strings.Contains(got, "ssh -p 2222 -i") {
		t.Fatalf("transfer command ssh options missing: %q", got)
	}
}

package cmd

import "testing"

func TestArchiveCommandTarGz(t *testing.T) {
	command, err := archiveCommand("logs", "tar.gz", "logs.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if got := command.String(); got != "tar -czf logs.tar.gz logs" {
		t.Fatalf("archive command = %q", got)
	}
}

func TestExtractCommandZip(t *testing.T) {
	command, err := extractCommand("logs.zip", "out")
	if err != nil {
		t.Fatal(err)
	}
	if got := command.String(); got != "unzip logs.zip -d out" {
		t.Fatalf("extract command = %q", got)
	}
}

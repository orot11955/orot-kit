package cmd

import "testing"

func TestCurlDownloadCommand(t *testing.T) {
	command := curlDownloadCommand("https://kit.local/bin/kit-linux-amd64", "kit", 3, 90)
	want := "curl -fL --retry 3 --connect-timeout 10 --max-time 90 --output kit https://kit.local/bin/kit-linux-amd64"
	if got := command.String(); got != want {
		t.Fatalf("download command = %q, want %q", got, want)
	}
}

func TestCurlDownloadCommandNormalizesBounds(t *testing.T) {
	command := curlDownloadCommand("https://kit.local/bin/kit-linux-amd64", "kit", -1, 0)
	want := "curl -fL --retry 0 --connect-timeout 10 --max-time 60 --output kit https://kit.local/bin/kit-linux-amd64"
	if got := command.String(); got != want {
		t.Fatalf("download command = %q, want %q", got, want)
	}
}

func TestDefaultDownloadOutput(t *testing.T) {
	if got := defaultDownloadOutput("https://kit.local/bin/kit-linux-amd64?token=1"); got != "kit-linux-amd64" {
		t.Fatalf("default output = %q", got)
	}
	if got := defaultDownloadOutput("https://kit.local/"); got != "download.out" {
		t.Fatalf("default output = %q", got)
	}
}

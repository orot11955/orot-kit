package runtime

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type InstallPlan struct {
	Runtime string
	Version string
	Path    string
	Source  string
}

type InstallRequest struct {
	Runtime        string
	Version        string
	Source         string
	RuntimeBaseURL string
	OS             string
	Arch           string
	SHA256         string
	Use            bool
}

type InstallResult struct {
	Plan  InstallPlan
	Shims []Shim
}

func (m Manager) Install(ctx context.Context, request InstallRequest) (InstallResult, error) {
	if !SupportedRuntime(request.Runtime) {
		return InstallResult{}, fmt.Errorf("unsupported runtime: %s", request.Runtime)
	}
	if request.Version == "" {
		return InstallResult{}, fmt.Errorf("runtime version is required")
	}
	dest := m.VersionDir(request.Runtime, request.Version)
	if _, err := os.Stat(dest); err == nil {
		return InstallResult{}, fmt.Errorf("%s %s is already installed at %s", request.Runtime, request.Version, dest)
	}
	if err := os.MkdirAll(m.RuntimeDir(request.Runtime), 0o755); err != nil {
		return InstallResult{}, err
	}
	source := request.Source
	cleanup := func() {}
	if source == "" {
		if request.RuntimeBaseURL == "" {
			return InstallResult{}, fmt.Errorf("runtime_base_url or --from is required for install")
		}
		url := RuntimeDownloadURL(request.RuntimeBaseURL, request.Runtime, request.Version, request.OS, request.Arch)
		downloaded, remove, err := Download(ctx, url)
		if err != nil {
			return InstallResult{}, err
		}
		source = downloaded
		cleanup = remove
	} else if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		downloaded, remove, err := Download(ctx, source)
		if err != nil {
			return InstallResult{}, err
		}
		source = downloaded
		cleanup = remove
	}
	defer cleanup()
	if request.SHA256 != "" {
		if err := VerifySHA256(source, request.SHA256); err != nil {
			return InstallResult{}, err
		}
	}
	tmp, err := os.MkdirTemp(m.RuntimeDir(request.Runtime), ".install-"+request.Version+"-")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(tmp)
	info, err := os.Stat(source)
	if err != nil {
		return InstallResult{}, err
	}
	if info.IsDir() {
		if err := copyDir(source, dest); err != nil {
			return InstallResult{}, err
		}
	} else {
		if err := extractArchive(source, tmp); err != nil {
			return InstallResult{}, err
		}
		candidate, err := installCandidate(tmp)
		if err != nil {
			return InstallResult{}, err
		}
		if err := os.Rename(candidate, dest); err != nil {
			return InstallResult{}, err
		}
	}
	result := InstallResult{Plan: InstallPlan{Runtime: request.Runtime, Version: request.Version, Path: dest, Source: source}}
	if request.Use {
		shims, err := m.Use(request.Runtime, request.Version)
		if err != nil {
			return InstallResult{}, err
		}
		result.Shims = shims
	}
	return result, nil
}

func (m Manager) Remove(name string, version string) error {
	if version == "" {
		return fmt.Errorf("runtime version is required")
	}
	if current, ok := m.CurrentVersion(name); ok && current.Version == version {
		return fmt.Errorf("cannot remove current %s version %s; switch versions first", name, version)
	}
	return os.RemoveAll(m.VersionDir(name, version))
}

func extractArchive(source string, dest string) error {
	lower := strings.ToLower(source)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(source, dest)
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(source, dest)
	default:
		if err := extractTarGz(source, dest); err == nil {
			return nil
		}
		if err := extractZip(source, dest); err == nil {
			return nil
		}
		return fmt.Errorf("unsupported runtime archive type: %s", source)
	}
}

func extractTarGz(source string, dest string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(dest, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&0o777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, reader)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}

func extractZip(source string, dest string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		target, err := safeJoin(dest, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode()&0o777)
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeInErr := in.Close()
		closeOutErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeInErr != nil {
			return closeInErr
		}
		if closeOutErr != nil {
			return closeOutErr
		}
	}
	return nil
}

func safeJoin(root string, name string) (string, error) {
	target := filepath.Clean(filepath.Join(root, name))
	root = filepath.Clean(root)
	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes destination: %s", name)
	}
	return target, nil
}

func installCandidate(tmp string) (string, error) {
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return "", err
	}
	visible := []os.DirEntry{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		visible = append(visible, entry)
	}
	if len(visible) == 1 && visible[0].IsDir() && visible[0].Name() != "bin" {
		return filepath.Join(tmp, visible[0].Name()), nil
	}
	return tmp, nil
}

func copyDir(source string, dest string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode()&0o777)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func Download(ctx context.Context, url string) (string, func(), error) {
	tmp, err := os.CreateTemp("", "kit-runtime-*")
	if err != nil {
		return "", nil, err
	}
	path := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(path)
		return "", nil, err
	}

	command := exec.CommandContext(ctx, "curl", CurlDownloadArgs(path, url)...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		os.Remove(path)
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", nil, fmt.Errorf("curl download failed: %s", message)
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func CurlDownloadArgs(outputPath string, url string) []string {
	return []string{"-fL", "--retry", "2", "--connect-timeout", "10", "--output", outputPath, url}
}

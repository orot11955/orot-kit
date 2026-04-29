package runtime

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ServerConfig struct {
	Address  string
	CacheDir string
}

func NewCacheServer(cacheDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/runtime", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/runtime" {
			handleRuntimeCacheFile(w, r, cacheDir)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"runtimes": Supported,
			"cacheDir": cacheDir,
		})
	})
	return mux
}

func handleRuntimeCacheFile(w http.ResponseWriter, r *http.Request, cacheDir string) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/runtime/"), "/")
	if len(parts) == 2 && parts[1] == "versions" {
		versions := cachedVersions(cacheDir, parts[0])
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"runtime": parts[0], "versions": versions})
		return
	}
	checksum := false
	if len(parts) == 5 && parts[4] == "checksum" {
		checksum = true
		parts = parts[:4]
	}
	if len(parts) != 4 {
		http.NotFound(w, r)
		return
	}
	path := findCachedRuntimeFile(cacheDir, parts[0], parts[1], parts[2], parts[3])
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if checksum {
		sum, err := FileSHA256(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(sum + "\n"))
		return
	}
	http.ServeFile(w, r, path)
}

func cachedVersions(cacheDir string, name string) []string {
	versions := map[string]bool{}
	root := filepath.Join(cacheDir, name)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		for _, version := range Specs()[name].SeedVersions {
			if strings.Contains(entry.Name(), version) {
				versions[version] = true
			}
		}
		return nil
	})
	out := []string{}
	for version := range versions {
		out = append(out, version)
	}
	return out
}

func findCachedRuntimeFile(cacheDir string, name string, version string, osName string, arch string) string {
	roots := []string{
		filepath.Join(cacheDir, name, osName+"-"+arch),
		filepath.Join(cacheDir, name, osName, arch),
		filepath.Join(cacheDir, name),
	}
	for _, root := range roots {
		var found string
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || found != "" {
				return nil
			}
			if strings.Contains(entry.Name(), version) {
				found = path
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

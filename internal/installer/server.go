package installer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	kitruntime "github.com/orot-dev/orot-kit/internal/runtime"
	"github.com/orot-dev/orot-kit/pkg/version"
)

type Config struct {
	BinDir          string
	RuntimeCacheDir string
	BaseURL         string
	AssetsDir       string
	StatsFile       string
}

type Server struct {
	config     Config
	counters   map[string]int64
	osCounters map[string]int64
	mu         sync.Mutex
}

type DownloadStats struct {
	Total  int64            `json:"total"`
	Mac    int64            `json:"mac"`
	Linux  int64            `json:"linux"`
	Other  int64            `json:"other"`
	ByPath map[string]int64 `json:"by_path"`
}

type statsFileData struct {
	Counters   map[string]int64 `json:"counters"`
	OSCounters map[string]int64 `json:"os_counters"`
}

func NewServer() http.Handler {
	return NewServerWithConfig(DefaultConfig())
}

func NewServerWithConfig(config Config) http.Handler {
	server := &Server{
		config:     config.withDefaults(),
		counters:   map[string]int64{},
		osCounters: map[string]int64{},
	}
	server.loadStats()
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleIndex)
	mux.HandleFunc("/assets/", server.handleAsset)
	mux.HandleFunc("/favicon.ico", server.handleFavicon)
	mux.HandleFunc("/install.sh", server.handleInstallScript)
	mux.HandleFunc("/uninstall.sh", server.handleUninstallScript)
	mux.HandleFunc("/version", server.handleVersion)
	mux.HandleFunc("/healthz", server.handleHealth)
	mux.HandleFunc("/stats", server.handleStats)
	mux.HandleFunc("/bin/", server.handleBinary)
	mux.HandleFunc("/runtime", server.handleRuntime)
	mux.HandleFunc("/runtime/", server.handleRuntime)
	return mux
}

func DefaultConfig() Config {
	return Config{
		BinDir:          "dist",
		RuntimeCacheDir: defaultRuntimeCacheDir(),
		BaseURL:         "http://localhost:8080",
		AssetsDir:       "assets",
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.BinDir == "" {
		c.BinDir = defaults.BinDir
	}
	if c.RuntimeCacheDir == "" {
		c.RuntimeCacheDir = defaults.RuntimeCacheDir
	}
	if c.BaseURL == "" {
		c.BaseURL = defaults.BaseURL
	}
	if c.AssetsDir == "" {
		c.AssetsDir = defaults.AssetsDir
	}
	return c
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	config := s.config
	config.BaseURL = s.requestBaseURL(r)
	_, _ = w.Write([]byte(renderPage(NewPageData(config, s.snapshotStats()))))
}

func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	_, _ = w.Write([]byte(renderInstallScript(s.requestBaseURL(r))))
}

func (s *Server) handleUninstallScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	_, _ = w.Write([]byte(renderUninstallScript()))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	baseURL := s.requestBaseURL(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"version":       version.Version,
		"commit":        version.Commit,
		"build_date":    version.BuildDate,
		"base_url":      baseURL,
		"install_url":   baseURL + "/install.sh",
		"uninstall_url": baseURL + "/uninstall.sh",
		"binaries":      s.binaryMetadata(baseURL),
		"runtime": map[string]string{
			"base_url": baseURL + "/runtime",
			"cache":    s.config.RuntimeCacheDir,
		},
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"downloads": s.snapshotStats(),
	})
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/assets/")
	path, ok := s.safeAssetPath(name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	path, ok := s.safeAssetPath("favicon.ico")
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) handleBinary(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/bin/")
	checksum := false
	if strings.HasSuffix(name, "/checksum") {
		checksum = true
		name = strings.TrimSuffix(name, "/checksum")
	}
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.config.BinDir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	if checksum {
		serveChecksum(w, path)
		return
	}
	if r.Method == http.MethodGet {
		s.countDownload("/bin/"+name, binaryDownloadOS(name))
	}
	http.ServeFile(w, r, path)
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	baseURL := s.requestBaseURL(r)
	if r.URL.Path == "/runtime" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"runtimes":  kitruntime.Supported,
			"cache_dir": s.config.RuntimeCacheDir,
			"base_url":  baseURL + "/runtime",
		})
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/runtime/"), "/")
	if len(parts) == 2 && parts[1] == "versions" {
		versions := cachedRuntimeVersions(s.config.RuntimeCacheDir, parts[0])
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"runtime":  parts[0],
			"versions": versions,
		})
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
	file := findCachedRuntimeFile(s.config.RuntimeCacheDir, parts[0], parts[1], parts[2], parts[3])
	if file == "" {
		http.NotFound(w, r)
		return
	}
	if checksum {
		serveChecksum(w, file)
		return
	}
	if r.URL.Query().Get("meta") == "1" {
		s.serveRuntimeMetadata(w, r, file, parts)
		return
	}
	if r.Method == http.MethodGet {
		s.countDownload("/runtime/"+strings.Join(parts, "/"), runtimeDownloadOS(parts[2]))
	}
	http.ServeFile(w, r, file)
}

func (s *Server) serveRuntimeMetadata(w http.ResponseWriter, r *http.Request, path string, parts []string) {
	info, err := os.Stat(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sum, err := kitruntime.FileSHA256(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":         parts[0],
		"version":      parts[1],
		"os":           parts[2],
		"arch":         parts[3],
		"file_name":    filepath.Base(path),
		"file_size":    info.Size(),
		"checksum":     sum,
		"cached":       true,
		"download_url": s.requestBaseURL(r) + "/runtime/" + strings.Join(parts, "/"),
	})
}

func (s *Server) binaryMetadata(baseURL string) []map[string]any {
	entries, err := os.ReadDir(s.config.BinDir)
	if err != nil {
		return []map[string]any{}
	}
	out := []map[string]any{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "kit-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out = append(out, map[string]any{
			"name":         entry.Name(),
			"size":         info.Size(),
			"platform":     binaryDownloadOS(entry.Name()),
			"download_url": baseURL + "/bin/" + entry.Name(),
			"checksum_url": baseURL + "/bin/" + entry.Name() + "/checksum",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return fmt.Sprint(out[i]["name"]) < fmt.Sprint(out[j]["name"])
	})
	return out
}

func (s *Server) requestBaseURL(r *http.Request) string {
	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = firstHeaderValue(r.Host)
	}
	if host == "" {
		return strings.TrimRight(s.config.BaseURL, "/")
	}

	scheme := firstHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = firstHeaderValue(r.Header.Get("X-Forwarded-Scheme"))
	}
	if scheme == "" && strings.EqualFold(firstHeaderValue(r.Header.Get("X-Forwarded-Ssl")), "on") {
		scheme = "https"
	}
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}
	return scheme + "://" + strings.TrimRight(host, "/")
}

func firstHeaderValue(value string) string {
	if value == "" {
		return ""
	}
	if index := strings.Index(value, ","); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func (s *Server) safeAssetPath(name string) (string, bool) {
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Join(s.config.AssetsDir, clean), true
}

func (s *Server) countDownload(key string, osName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[key]++
	switch osName {
	case "mac":
		s.osCounters["mac"]++
	case "linux":
		s.osCounters["linux"]++
	default:
		s.osCounters["other"]++
	}
	s.saveStatsLocked()
}

func (s *Server) snapshotStats() DownloadStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int64, len(s.counters))
	for key, value := range s.counters {
		out[key] = value
	}
	mac := s.osCounters["mac"]
	linux := s.osCounters["linux"]
	other := s.osCounters["other"]
	return DownloadStats{
		Total:  mac + linux + other,
		Mac:    mac,
		Linux:  linux,
		Other:  other,
		ByPath: out,
	}
}

func binaryDownloadOS(name string) string {
	switch {
	case strings.HasPrefix(name, "kit-darwin-"):
		return "mac"
	case strings.HasPrefix(name, "kit-linux-"):
		return "linux"
	default:
		return "other"
	}
}

func runtimeDownloadOS(osName string) string {
	switch osName {
	case "darwin":
		return "mac"
	case "linux":
		return "linux"
	default:
		return "other"
	}
}

func (s *Server) loadStats() {
	if s.config.StatsFile == "" {
		return
	}
	content, err := os.ReadFile(s.config.StatsFile)
	if err != nil {
		return
	}
	var data statsFileData
	if err := json.Unmarshal(content, &data); err != nil {
		return
	}
	if data.Counters != nil {
		s.counters = data.Counters
	}
	if data.OSCounters != nil {
		s.osCounters = data.OSCounters
	}
}

func (s *Server) saveStatsLocked() {
	if s.config.StatsFile == "" {
		return
	}
	dir := filepath.Dir(s.config.StatsFile)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
	}
	data := statsFileData{
		Counters:   s.counters,
		OSCounters: s.osCounters,
	}
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	tmp := s.config.StatsFile + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, s.config.StatsFile)
}

func serveChecksum(w http.ResponseWriter, path string) {
	sum, err := kitruntime.FileSHA256(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(sum + "  " + filepath.Base(path) + "\n"))
}

func cachedRuntimeVersions(cacheDir string, name string) []string {
	versions := map[string]bool{}
	root := filepath.Join(cacheDir, name)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		for _, seed := range kitruntime.Specs()[name].SeedVersions {
			if strings.Contains(entry.Name(), seed) {
				versions[seed] = true
			}
		}
		return nil
	})
	out := make([]string, 0, len(versions))
	for value := range versions {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func findCachedRuntimeFile(cacheDir string, name string, runtimeVersion string, osName string, arch string) string {
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
			file := entry.Name()
			if strings.Contains(file, runtimeVersion) && strings.Contains(file, osName) && archMatches(file, arch) {
				found = path
				return nil
			}
			if strings.Contains(file, runtimeVersion) && found == "" {
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

func archMatches(file string, arch string) bool {
	if strings.Contains(file, arch) {
		return true
	}
	return arch == "amd64" && strings.Contains(file, "x64")
}

func defaultRuntimeCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".kit-server", "cache", "runtimes")
	}
	return filepath.Join(home, ".kit-server", "cache", "runtimes")
}

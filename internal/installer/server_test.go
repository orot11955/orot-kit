package installer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallServerVersionAndBinary(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "dist")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(binDir, "kit-linux-amd64")
	if err := os.WriteFile(binary, []byte("kit"), 0o755); err != nil {
		t.Fatal(err)
	}

	handler := NewServerWithConfig(Config{
		BinDir:          binDir,
		RuntimeCacheDir: filepath.Join(root, "runtimes"),
		BaseURL:         "http://example.test",
	})

	response := request(handler, "/version")
	if response.Code != http.StatusOK {
		t.Fatalf("/version status = %d", response.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	binaries, ok := body["binaries"].([]any)
	if !ok || len(binaries) != 1 {
		t.Fatalf("binaries = %#v", body["binaries"])
	}

	binResponse := request(handler, "/bin/kit-linux-amd64")
	if binResponse.Code != http.StatusOK {
		t.Fatalf("/bin status = %d", binResponse.Code)
	}
	statsResponse := request(handler, "/stats")
	var stats struct {
		Downloads DownloadStats `json:"downloads"`
	}
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.Downloads.Linux != 1 || stats.Downloads.Mac != 0 || stats.Downloads.Total != 1 {
		t.Fatalf("download stats = %#v", stats.Downloads)
	}
	if stats.Downloads.ByPath["/bin/kit-linux-amd64"] != 1 {
		t.Fatalf("download path stats = %#v", stats.Downloads.ByPath)
	}

	checksumResponse := request(handler, "/bin/kit-linux-amd64/checksum")
	if checksumResponse.Code != http.StatusOK {
		t.Fatalf("/bin checksum status = %d", checksumResponse.Code)
	}
}

func TestInstallServerRuntimeMetadataAndStats(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "runtimes")
	runtimeDir := filepath.Join(cacheDir, "node", "linux-amd64")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(runtimeDir, "node-v22.3.0-linux-x64.tar.gz")
	if err := os.WriteFile(archive, []byte("runtime"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := NewServerWithConfig(Config{
		BinDir:          filepath.Join(root, "dist"),
		RuntimeCacheDir: cacheDir,
		BaseURL:         "http://example.test",
	})

	metaResponse := request(handler, "/runtime/node/22.3.0/linux/amd64?meta=1")
	if metaResponse.Code != http.StatusOK {
		t.Fatalf("runtime meta status = %d", metaResponse.Code)
	}
	var meta map[string]any
	if err := json.NewDecoder(metaResponse.Body).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	if meta["cached"] != true || meta["version"] != "22.3.0" {
		t.Fatalf("runtime meta = %#v", meta)
	}

	headResponse := requestWithMethod(handler, http.MethodHead, "/runtime/node/22.3.0/linux/amd64")
	if headResponse.Code != http.StatusOK {
		t.Fatalf("runtime head status = %d", headResponse.Code)
	}

	fileResponse := request(handler, "/runtime/node/22.3.0/linux/amd64")
	if fileResponse.Code != http.StatusOK {
		t.Fatalf("runtime file status = %d", fileResponse.Code)
	}

	statsResponse := request(handler, "/stats")
	var stats struct {
		Downloads DownloadStats `json:"downloads"`
	}
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.Downloads.Linux != 1 || stats.Downloads.Total != 1 {
		t.Fatalf("download stats = %#v", stats.Downloads)
	}
	if stats.Downloads.ByPath["/runtime/node/22.3.0/linux/amd64"] != 1 {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestInstallServerPageAssetsAndMacLinuxStats(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "dist")
	assetsDir := filepath.Join(root, "assets")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kit-darwin-arm64", "kit-linux-amd64"} {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"orot.png", "mac.png", "linux.png", "favicon.ico"} {
		if err := os.WriteFile(filepath.Join(assetsDir, name), []byte("image"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	handler := NewServerWithConfig(Config{
		BinDir:    binDir,
		AssetsDir: assetsDir,
		BaseURL:   "http://kit.local",
	})

	pageResponse := requestWithHost(handler, "/", "kit.local")
	if pageResponse.Code != http.StatusOK {
		t.Fatalf("page status = %d", pageResponse.Code)
	}
	page := pageResponse.Body.String()
	for _, want := range []string{
		"orot-kit",
		"curl -fsSL http://kit.local/install.sh | sh",
		"/assets/orot.png",
		"/assets/mac.png",
		"/assets/linux.png",
		"지원 OS",
		"supported-os-list",
		"macOS",
		"Linux",
		"kit git diff",
		"kit runtime serve",
		"make serve",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q", want)
		}
	}
	for _, unexpected := range []string{"직접 다운로드", `class="chip"`, `href="/bin/kit-darwin-arm64"`} {
		if strings.Contains(page, unexpected) {
			t.Fatalf("page should not contain %q", unexpected)
		}
	}

	if assetResponse := request(handler, "/assets/orot.png"); assetResponse.Code != http.StatusOK {
		t.Fatalf("asset status = %d", assetResponse.Code)
	}
	if faviconResponse := request(handler, "/favicon.ico"); faviconResponse.Code != http.StatusOK {
		t.Fatalf("favicon status = %d", faviconResponse.Code)
	}

	_ = request(handler, "/bin/kit-darwin-arm64")
	_ = request(handler, "/bin/kit-linux-amd64")
	statsResponse := request(handler, "/stats")
	var stats struct {
		Downloads DownloadStats `json:"downloads"`
	}
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.Downloads.Mac != 1 || stats.Downloads.Linux != 1 || stats.Downloads.Total != 2 {
		t.Fatalf("stats = %#v", stats.Downloads)
	}
}

func TestInstallServerPersistsStatsFile(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "dist")
	statsFile := filepath.Join(root, "download-stats.json")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "kit-linux-amd64"), []byte("kit"), 0o755); err != nil {
		t.Fatal(err)
	}

	config := Config{
		BinDir:    binDir,
		BaseURL:   "http://kit.local",
		StatsFile: statsFile,
	}
	first := NewServerWithConfig(config)
	if response := request(first, "/bin/kit-linux-amd64"); response.Code != http.StatusOK {
		t.Fatalf("download status = %d", response.Code)
	}

	second := NewServerWithConfig(config)
	statsResponse := request(second, "/stats")
	var stats struct {
		Downloads DownloadStats `json:"downloads"`
	}
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.Downloads.Linux != 1 || stats.Downloads.Total != 1 {
		t.Fatalf("persisted stats = %#v", stats.Downloads)
	}
}

func TestInstallServerHeadDoesNotCountDownload(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "dist")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "kit-linux-amd64"), []byte("kit"), 0o755); err != nil {
		t.Fatal(err)
	}

	handler := NewServerWithConfig(Config{
		BinDir:  binDir,
		BaseURL: "http://kit.local",
	})

	if response := requestWithMethod(handler, http.MethodHead, "/bin/kit-linux-amd64"); response.Code != http.StatusOK {
		t.Fatalf("head status = %d", response.Code)
	}
	statsResponse := request(handler, "/stats")
	var stats struct {
		Downloads DownloadStats `json:"downloads"`
	}
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.Downloads.Total != 0 || stats.Downloads.Linux != 0 {
		t.Fatalf("HEAD should not count as download: %#v", stats.Downloads)
	}
}

func TestInstallScriptUsesRequestURL(t *testing.T) {
	handler := NewServerWithConfig(Config{BaseURL: "http://fallback.local"})
	response := requestWithHost(handler, "/install.sh", "kit.local")
	body := response.Body.String()
	if !strings.Contains(body, `base="${KIT_BASE_URL:-http://kit.local}"`) {
		t.Fatalf("install script missing base URL: %s", body)
	}
}

func TestInstallScriptDoesNotCountAsDownload(t *testing.T) {
	handler := NewServerWithConfig(Config{BaseURL: "http://kit.local"})
	if response := request(handler, "/install.sh"); response.Code != http.StatusOK {
		t.Fatalf("install script status = %d", response.Code)
	}
	statsResponse := request(handler, "/stats")
	var stats struct {
		Downloads DownloadStats `json:"downloads"`
	}
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.Downloads.Total != 0 || len(stats.Downloads.ByPath) != 0 {
		t.Fatalf("install script should not count as download: %#v", stats.Downloads)
	}
}

func TestInstallServerUsesForwardedURL(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "dist")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "kit-linux-amd64"), []byte("kit"), 0o755); err != nil {
		t.Fatal(err)
	}

	handler := NewServerWithConfig(Config{
		BinDir:  binDir,
		BaseURL: "http://fallback.local",
	})

	pageResponse := requestWith(handler, "/", func(request *http.Request) {
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "downloads.orot.dev")
	})
	page := pageResponse.Body.String()
	if !strings.Contains(page, "curl -fsSL https://downloads.orot.dev/install.sh | sh") {
		t.Fatalf("page did not use forwarded URL: %s", page)
	}

	scriptResponse := requestWith(handler, "/install.sh", func(request *http.Request) {
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "downloads.orot.dev")
	})
	if body := scriptResponse.Body.String(); !strings.Contains(body, `base="${KIT_BASE_URL:-https://downloads.orot.dev}"`) {
		t.Fatalf("install script did not use forwarded URL: %s", body)
	}

	versionResponse := requestWith(handler, "/version", func(request *http.Request) {
		request.Header.Set("X-Forwarded-Proto", "https")
		request.Header.Set("X-Forwarded-Host", "downloads.orot.dev")
	})
	var versionBody map[string]any
	if err := json.NewDecoder(versionResponse.Body).Decode(&versionBody); err != nil {
		t.Fatal(err)
	}
	if versionBody["base_url"] != "https://downloads.orot.dev" {
		t.Fatalf("version base_url = %#v", versionBody["base_url"])
	}
	binaries, ok := versionBody["binaries"].([]any)
	if !ok || len(binaries) != 1 {
		t.Fatalf("binaries = %#v", versionBody["binaries"])
	}
	binary, ok := binaries[0].(map[string]any)
	if !ok || binary["download_url"] != "https://downloads.orot.dev/bin/kit-linux-amd64" {
		t.Fatalf("binary metadata = %#v", binaries[0])
	}
}

func request(handler http.Handler, path string) *httptest.ResponseRecorder {
	return requestWith(handler, path, nil)
}

func requestWithMethod(handler http.Handler, method string, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	handler.ServeHTTP(recorder, request)
	return recorder
}

func requestWithHost(handler http.Handler, path string, host string) *httptest.ResponseRecorder {
	return requestWith(handler, path, func(request *http.Request) {
		request.Host = host
	})
}

func requestWith(handler http.Handler, path string, configure func(*http.Request)) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if configure != nil {
		configure(request)
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}

package installer

import (
	"html"
	"strconv"
	"strings"

	"github.com/orot-dev/orot-kit/pkg/version"
)

type PageData struct {
	Config         Config
	Stats          DownloadStats
	BaseURL        string
	InstallCommand string
}

func NewPageData(config Config, stats DownloadStats) PageData {
	config = config.withDefaults()
	baseURL := strings.TrimRight(config.BaseURL, "/")
	return PageData{
		Config:         config,
		Stats:          stats,
		BaseURL:        baseURL,
		InstallCommand: "curl -fsSL " + baseURL + "/install.sh | sh",
	}
}

func renderPage(data PageData) string {
	baseURL := html.EscapeString(data.BaseURL)
	install := html.EscapeString(data.InstallCommand)
	installAlt := html.EscapeString("curl -fsSL " + data.BaseURL + "/install.sh | KIT_INSTALL_DIR=~/bin sh")
	uninstall := html.EscapeString("curl -fsSL " + data.BaseURL + "/uninstall.sh | sh")
	macCount := strconv.FormatInt(data.Stats.Mac, 10)
	linuxCount := strconv.FormatInt(data.Stats.Linux, 10)
	versionText := html.EscapeString(version.Version)

	return `<!doctype html>
<html lang="ko">

<head>
  <meta charset="utf-8">
  <title>orot-kit docs</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="macOS와 Linux에서 자주 쓰는 OS, 개발, 서버 관리 유틸을 한 줄로 묶은 개인 터미널 툴킷">
  <link rel="icon" href="/favicon.ico">

  <style>
    *,
    *::before,
    *::after {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }

    :root {
      --bg: #0a0c10;
      --surface: #111318;
      --surface2: #181b22;
      --border: #21262d;
      --border2: #30363d;
      --fg: #cdd9e5;
      --fg2: #768390;
      --fg3: #4d5562;
      --accent: #3dd68c;
      --accent2: #26a65b;
      --blue: #539bf5;
      --yellow: #c69026;
      --red: #e5534b;
      --radius: 10px;
      --radius-sm: 6px;
      --mono: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    }

    html {
      scroll-behavior: smooth;
    }

    body {
      background:
        radial-gradient(circle at 48% 0%, rgba(61, 214, 140, 0.05), transparent 34rem),
        var(--bg);
      color: var(--fg);
      font-family: -apple-system, BlinkMacSystemFont, "Inter", "Segoe UI", system-ui, sans-serif;
      font-size: 15px;
      line-height: 1.7;
      -webkit-font-smoothing: antialiased;
    }

    a {
      color: inherit;
      text-decoration: none;
    }

    .layout {
      display: grid;
      grid-template-columns: 220px minmax(0, 860px);
      column-gap: 1.75rem;
      width: fit-content;
      max-width: calc(100vw - 4rem);
      min-height: 100vh;
      margin: 0 auto;
    }

    .sidebar {
      position: sticky;
      top: 2rem;
      height: calc(100vh - 4rem);
      overflow-y: auto;
      background: transparent;
      padding: 1rem 0;
      display: flex;
      flex-direction: column;
      scrollbar-width: thin;
      scrollbar-color: var(--border2) transparent;
    }

    .sidebar::before {
      content: "";
      position: absolute;
      top: 1rem;
      bottom: 1rem;
      right: -0.75rem;
      width: 1px;
      background: linear-gradient(to bottom, transparent, color-mix(in srgb, var(--border2) 50%, transparent), transparent);
      pointer-events: none;
    }

    .sidebar-logo {
      padding: 0 0.65rem 1.15rem 0.5rem;
      margin-bottom: 0.9rem;
      text-align: right;
    }

    .logo-img {
      display: block;
      width: 32px;
      height: 32px;
      object-fit: contain;
      margin-left: auto;
      margin-bottom: 0.45rem;
      border-radius: 8px;
    }

    .sidebar-logo .ver {
      font-family: var(--mono);
      font-size: 0.72rem;
      line-height: 1.2;
      color: var(--fg3);
    }

    .sidebar-os-stats {
      display: grid;
      gap: 0.35rem;
      margin-top: 0.8rem;
      justify-items: end;
    }

    .os-stat {
      display: inline-grid;
      grid-template-columns: 1.05rem auto;
      align-items: center;
      gap: 0.45rem;
      min-width: 4.3rem;
      padding: 0.28rem 0.5rem;
      border: 1px solid var(--border2);
      border-radius: var(--radius-sm);
      background: var(--surface);
      box-shadow: inset 0 1px 0 color-mix(in srgb, var(--fg) 4%, transparent);
      font-family: var(--mono);
      font-size: 0.75rem;
      line-height: 1.2;
      color: var(--fg2);
    }

    .os-logo {
      display: block;
      width: 1.05rem;
      height: 1.05rem;
      object-fit: contain;
      opacity: 0.9;
      filter: grayscale(1) brightness(1.45);
    }

    .os-count {
      min-width: 1.5rem;
      text-align: right;
      color: var(--accent);
      font-variant-numeric: tabular-nums;
    }

    .sidebar-section {
      padding: 0 0.75rem;
      margin-bottom: 0.25rem;
    }

    .sidebar-section-title {
      font-size: 0.68rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: color-mix(in srgb, var(--fg3) 70%, transparent);
      padding: 0 0.65rem 0 0.5rem;
      margin: 1.15rem 0 0.35rem;
      text-align: right;
    }

    .sidebar-section-title:first-child {
      margin-top: 0;
    }

    .sidebar nav {
      position: relative;
    }

    .sidebar nav a {
      display: block;
      position: relative;
      padding: 0.28rem 0.65rem 0.28rem 0.5rem;
      border-radius: var(--radius-sm);
      font-size: 0.82rem;
      font-weight: 500;
      color: var(--fg3);
      text-align: right;
      text-decoration: none;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      opacity: 0.42;
      transform: translateX(0) scale(1);
      transform-origin: right center;
      transition:
        opacity 0.18s ease,
        color 0.18s ease,
        font-size 0.18s ease,
        transform 0.18s ease,
        background 0.18s ease,
        text-shadow 0.18s ease;
    }

    .sidebar nav a:hover {
      opacity: 0.82;
      color: var(--fg);
      background: color-mix(in srgb, var(--surface2) 35%, transparent);
    }

    .sidebar nav a.active {
      opacity: 1;
      color: var(--fg);
      font-size: 1.02rem;
      font-weight: 750;
      transform: translateX(-0.15rem);
      background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--accent) 12%, transparent));
      text-shadow:
        0 0 18px color-mix(in srgb, var(--accent) 30%, transparent),
        0 0 1px color-mix(in srgb, var(--fg) 50%, transparent);
    }

    .sidebar nav:has(a.active) a:not(.active) {
      opacity: 0.28;
      color: color-mix(in srgb, var(--fg3) 78%, transparent);
    }

    .sidebar nav:has(a.active) a:not(.active):hover {
      opacity: 0.75;
      color: var(--fg2);
    }

    .main {
      min-width: 0;
      width: 860px;
      max-width: 100%;
      padding: 3.5rem 0 6rem;
    }

    .hero {
      margin-bottom: 3rem;
    }

    .hero h1 {
      font-size: 2.8rem;
      font-weight: 800;
      color: var(--accent);
      letter-spacing: -0.04em;
      line-height: 1.1;
    }

    .hero .tagline {
      color: var(--fg2);
      font-size: 1rem;
      margin-top: 0.5rem;
    }

    .install-box {
      background: var(--surface);
      border: 1px solid var(--accent2);
      border-radius: var(--radius);
      padding: 1.1rem 1.4rem;
      margin: 1rem 0 0.6rem;
      font-family: var(--mono);
      font-size: 0.92rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .install-box .prompt {
      color: var(--fg3);
      user-select: none;
    }

    .install-box .cmd {
      color: var(--fg);
      flex: 1;
      word-break: break-all;
    }

    .install-alt {
      font-size: 0.83rem;
      color: var(--fg2);
      margin-top: 0.4rem;
    }

    .install-alt code {
      background: var(--surface2);
      border: 1px solid var(--border2);
      padding: 0.1rem 0.4rem;
      border-radius: 4px;
      font-family: var(--mono);
      font-size: 0.82rem;
      color: var(--fg);
    }

    .supported-os-list {
      display: grid;
      gap: 0.35rem;
      margin: 0.75rem 0 1.2rem;
      color: var(--fg2);
      font-family: var(--mono);
      font-size: 0.84rem;
    }

    .supported-os-item {
      display: grid;
      grid-template-columns: 5.5rem 1fr;
      column-gap: 1rem;
      align-items: baseline;
      padding: 0.12rem 0;
      border-bottom: 1px solid color-mix(in srgb, var(--border) 72%, transparent);
    }

    .supported-os-name {
      color: var(--fg);
      font-weight: 600;
    }

    .supported-os-arch {
      color: var(--fg2);
    }

    .main h2 {
      scroll-margin-top: 2rem;
      font-size: 1.25rem;
      font-weight: 700;
      color: var(--fg);
      margin-top: 3rem;
      padding-bottom: 0.5rem;
      border-bottom: 1px solid var(--border);
      letter-spacing: -0.01em;
    }

    .main h2:first-of-type {
      margin-top: 0;
    }

    .main h3 {
      font-size: 0.72rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.07em;
      color: var(--fg3);
      margin-top: 1.8rem;
      margin-bottom: 0.5rem;
    }

    .main p {
      color: var(--fg2);
      font-size: 0.9rem;
      margin: 0.5rem 0;
    }

    pre {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 1rem 1.2rem;
      overflow-x: auto;
      margin: 0.5rem 0 0.9rem;
      position: relative;
    }

    pre code {
      font-family: var(--mono);
      font-size: 0.83rem;
      line-height: 1.7;
      color: var(--fg);
      background: transparent;
      padding: 0;
      border: none;
      border-radius: 0;
    }

    pre code .c {
      color: var(--fg3);
    }

    :not(pre)>code {
      font-family: var(--mono);
      font-size: 0.82em;
      background: var(--surface2);
      border: 1px solid var(--border2);
      padding: 0.1em 0.4em;
      border-radius: 4px;
      color: var(--fg);
    }

    .table-wrap {
      overflow-x: auto;
      margin: 0.5rem 0 1.2rem;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.85rem;
    }

    th {
      color: var(--fg3);
      font-weight: 600;
      font-size: 0.72rem;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      padding: 0.5rem 0.8rem;
      border-bottom: 1px solid var(--border2);
      text-align: left;
      white-space: nowrap;
    }

    td {
      padding: 0.55rem 0.8rem;
      border-bottom: 1px solid var(--border);
      color: var(--fg2);
      vertical-align: top;
    }

    td:first-child {
      white-space: nowrap;
    }

    tr:last-child td {
      border-bottom: none;
    }

    .callout {
      border-radius: var(--radius-sm);
      padding: 0.65rem 1rem;
      font-size: 0.85rem;
      margin: 0.6rem 0;
      display: flex;
      gap: 0.5rem;
      align-items: flex-start;
    }

    .callout-icon {
      flex-shrink: 0;
    }

    .callout.tip {
      background: color-mix(in srgb, var(--accent) 8%, transparent);
      border: 1px solid color-mix(in srgb, var(--accent) 25%, transparent);
      color: var(--fg2);
    }

    .callout.warn {
      background: color-mix(in srgb, var(--red) 8%, transparent);
      border: 1px solid color-mix(in srgb, var(--red) 25%, transparent);
      color: var(--fg2);
    }

    .callout.info {
      background: color-mix(in srgb, var(--blue) 8%, transparent);
      border: 1px solid color-mix(in srgb, var(--blue) 25%, transparent);
      color: var(--fg2);
    }

    .callout.tip .callout-icon {
      color: var(--accent);
    }

    .callout.warn .callout-icon {
      color: var(--red);
    }

    .callout.info .callout-icon {
      color: var(--blue);
    }

    footer {
      margin-top: 5rem;
      padding-top: 1.5rem;
      border-top: 1px solid var(--border);
      font-size: 0.8rem;
      color: var(--fg3);
    }

    footer code {
      font-family: var(--mono);
      font-size: 0.8em;
      color: var(--fg2);
    }

    @media (max-width: 1100px) {
      .layout {
        grid-template-columns: 190px minmax(0, 760px);
        column-gap: 1.5rem;
        max-width: calc(100vw - 2rem);
      }

      .main {
        width: 760px;
      }

      .sidebar nav a.active {
        font-size: 0.96rem;
      }
    }

    @media (max-width: 900px) {
      .layout {
        display: block;
        width: auto;
        max-width: none;
        margin: 0;
      }

      .sidebar {
        display: none;
      }

      .main {
        width: auto;
        max-width: none;
        padding: 2rem 1.25rem 4rem;
      }

      .hero h1 {
        font-size: 2rem;
      }
    }

    @media (prefers-reduced-motion: reduce) {
      html {
        scroll-behavior: auto;
      }

      .sidebar nav a {
        transition: none;
      }
    }
  </style>
</head>

<body>
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <img class="logo-img" src="/assets/orot.png" alt="orot-kit logo">
        <div class="ver">v` + versionText + `</div>

        <div class="sidebar-os-stats" aria-label="download counts">
          <div class="os-stat" title="macOS downloads">
            <img class="os-logo" src="/assets/mac.png" alt="macOS">
            <span class="os-count">` + macCount + `</span>
          </div>
          <div class="os-stat" title="Linux downloads">
            <img class="os-logo" src="/assets/linux.png" alt="Linux">
            <span class="os-count">` + linuxCount + `</span>
          </div>
        </div>
      </div>

      <div class="sidebar-section">
        <nav>
          <div class="sidebar-section-title">시작하기</div>
          <a href="#install">설치</a>
          <a href="#update">업데이트</a>
          <a href="#uninstall">제거</a>
          <a href="#quickstart">빠른 시작</a>
          <a href="#changes">변경 사항</a>

          <div class="sidebar-section-title">기능</div>
          <a href="#files">Files</a>
          <a href="#archive">Archive</a>
          <a href="#network">Network</a>
          <a href="#resource">Resource</a>
          <a href="#git">Git & Diff</a>
          <a href="#service">Service</a>
          <a href="#ssh-transfer">SSH & Transfer</a>
          <a href="#firewall">Firewall</a>
          <a href="#secret">Secret</a>

          <div class="sidebar-section-title">기타</div>
          <a href="#flags">공통 플래그</a>
          <a href="#api">서버 API</a>
        </nav>
      </div>
    </aside>

    <main class="main">
      <div class="hero">
        <h1>orot-kit</h1>
        <p class="tagline">macOS와 Linux에서 자주 쓰는 OS·개발·서버 관리 유틸을 하나의 CLI로.</p>
      </div>

      <h2 id="install">설치</h2>
      <p>아래 한 줄로 OS와 아키텍처를 자동 감지해 <code>$HOME/.local/bin/kit</code>에 설치한다.</p>

      <div class="install-box">
        <span class="prompt">$</span>
        <span class="cmd">` + install + `</span>
      </div>
      <div class="install-alt">
        설치 위치 변경 &nbsp;→&nbsp;
        <code>` + installAlt + `</code>
      </div>

      <h2 id="update">업데이트</h2>
      <p>설치 서버의 현재 OS/Arch 바이너리로 실행 중인 <code>kit</code>을 교체한다. 업데이트 다운로드는 서버 다운로드 카운트에 포함하지 않는다.</p>
      <pre><code>kit update                  <span class="c"># 현재 kit 바이너리 업데이트</span>
kit update --dry-run        <span class="c"># 다운로드 URL과 교체 경로만 확인</span>
kit update --base-url ` + baseURL + `</code></pre>

      <h3>지원 OS</h3>
      <div class="supported-os-list" aria-label="supported operating systems">
        <div class="supported-os-item">
          <span class="supported-os-name">macOS</span>
          <span class="supported-os-arch">arm64, amd64</span>
        </div>
        <div class="supported-os-item">
          <span class="supported-os-name">Linux</span>
          <span class="supported-os-arch">arm64, amd64</span>
        </div>
      </div>

      <h2 id="uninstall">제거</h2>
      <p>설치된 <code>kit</code> 바이너리와 <code>~/.kit</code>, <code>~/.kit-server</code> 상태 파일을 함께 정리한다.</p>
      <pre><code>kit uninstall         <span class="c"># kit 패키지 제거</span></code></pre>
      <div class="install-alt">
        CLI에서 먼저 확인 &nbsp;→&nbsp;
        <code>kit uninstall --dry-run</code>
      </div>

      <h2 id="quickstart">빠른 시작</h2>
      <pre><code>kit --help            <span class="c"># 대표 명령어 목록</span>
kit -v                <span class="c"># 빌드 버전 확인</span>
kit version           <span class="c"># 빌드 버전 확인</span>
kit update --dry-run  <span class="c"># 업데이트 다운로드 URL 확인</span>
kit info              <span class="c"># OS, Arch, Go, 설치 경로</span>
kit resource          <span class="c"># 서버 리소스 요약</span>
kit network           <span class="c"># 네트워크 요약</span>
kit git status        <span class="c"># 안전한 Git 상태 확인</span></code></pre>

      <h3>출력 형식</h3>
      <pre><code>Title
  Kit Info

Result
  Version: 0.1.0-dev
  OS: linux
  Arch: amd64</code></pre>
      <p>일반 명령은 제목, 실행 명령, 요약, 결과, 힌트를 같은 구조로 출력한다. 스크립트에서 쓰려면 <code>--json</code>을 붙인다.</p>

      <h2 id="changes">변경 사항</h2>
      <ul>
        <li>Runtime Manager 기능과 Node·Go·Python·Java 런타임 단축 명령을 제거했다.</li>
        <li>Docker 관리 기능을 제거하고 서비스 관리는 systemctl·Homebrew services 중심으로 단순화했다.</li>
        <li>설치 서버에서 런타임 캐시 API와 runtime-cache-dir 옵션을 제거했다.</li>
      </ul>

      <h2 id="files">Files</h2>

      <h3>탐색·검색·용량</h3>
      <pre><code>kit ls .                      <span class="c"># ls -al .</span>
kit ls ..                     <span class="c"># ls -al ..</span>
kit ls ../..                  <span class="c"># ls -al ../..</span>
kit ls ./src
kit tree . --depth 2

kit find TODO --root .
kit find "*.go" --root . --type file
kit find nginx /etc

kit size .
kit size ./dist</code></pre>
      <div class="callout info">
        <span class="callout-icon">i</span>
        <span><code>kit find</code>는 패턴에 와일드카드가 없으면 자동으로 <code>*pattern*</code> 형태로 찾는다.</span>
      </div>

      <h2 id="archive">Archive</h2>

      <h3>압축·해제</h3>
      <pre><code>kit archive README.md --format tar.gz --output readme.tar.gz
kit archive ./dist --format zip --output dist.zip
kit archive ./logs --format tar.gz --output logs.tar.gz

kit extract readme.tar.gz --dest ./out
kit extract dist.zip ./out</code></pre>
      <p>지원 포맷: <code>tar.gz</code> · <code>tgz</code> · <code>zip</code> · <code>gzip</code></p>

      <h2 id="network">Network</h2>

      <h3>요약·IP·DNS·HTTP</h3>
      <pre><code>kit network
kit network ip
kit network ping example.com --count 4
kit network dig example.com
kit network curl https://example.com --method GET
kit network download ` + baseURL + `/bin/kit-linux-amd64 --output kit --executable</code></pre>

      <h3>포트·패킷 캡처</h3>
      <pre><code>kit network port
kit network port kill 1234 --yes

kit network tcpdump --interface eth0 --port 443 --count 50 --dry-run
kit network tcpdump --interface en0 --host 1.1.1.1 --write capture.pcap</code></pre>
      <div class="callout warn">
        <span class="callout-icon">!</span>
        <span><code>tcpdump</code>는 root 권한이 필요할 수 있다. 실제 실행 전에는 <code>--dry-run</code>으로 명령을 먼저 확인한다.</span>
      </div>

      <h2 id="resource">Resource</h2>

      <h3>서버 리소스·로그</h3>
      <pre><code>kit resource                 <span class="c"># host, uptime, disk, memory, process 요약</span>
kit resource disk            <span class="c"># df -h</span>
kit resource memory          <span class="c"># free -h 또는 vm_stat</span>
kit resource process         <span class="c"># ps aux 상위 항목</span>
kit resource logs nginx
kit resource logs --unit nginx</code></pre>

      <h2 id="git">Git & Diff</h2>

      <h3>조회 전용 Git</h3>
      <pre><code>kit git
kit git status
kit git position
kit git diff
kit git diff --stat
kit git diff --name-only
kit git diff --staged
kit git diff --against main
kit git diff --base origin/main</code></pre>
      <p>Git 기능은 저장소를 변경하지 않는 조회 명령에 집중한다.</p>

      <h3>파일 간 코드 비교</h3>
      <pre><code>kit diff README.md
kit diff old.go new.go
kit diff old.go new.go --context 5</code></pre>

      <h2 id="service">Service</h2>

      <h3>서비스 alias</h3>
      <pre><code>kit service add nginx --type systemctl --name nginx
kit service nginx status
kit service nginx logs
kit service nginx restart</code></pre>

      <h2 id="ssh-transfer">SSH & Transfer</h2>

      <h3>SSH alias·키</h3>
      <pre><code>kit ssh add prod --host example.com --user deploy --port 22
kit ssh prod
kit ssh keygen ~/.ssh/id_ed25519
kit ssh copy deploy@example.com</code></pre>

      <h3>scp·rsync</h3>
      <pre><code>kit send --server prod --local ./dist --remote /srv/app
kit receive --server prod --remote /var/log/app.log --local ./logs/
kit sync --server prod --local ./dist --remote /srv/app --delete</code></pre>

      <h2 id="firewall">Firewall</h2>

      <h3>상태·포트 변경</h3>
      <pre><code>kit fw status
kit fw list
kit fw open 8080
kit fw open 5353 --proto udp
kit fw close 8080 --yes</code></pre>
      <p>Linux는 <code>ufw</code> 또는 <code>firewall-cmd</code>, macOS는 <code>pfctl</code> 상태 조회를 사용한다.</p>

      <h2 id="secret">Secret</h2>

      <h3>secret 생성</h3>
      <pre><code>kit secret password --length 32
kit secret password --length 48 --symbols=false
kit secret token --length 48
kit secret api-key --prefix orot --length 32
kit secret jwt --format env --key JWT_SECRET --no-print
kit secret hex --length 32
kit secret base64 --length 32
kit secret env --key API_TOKEN --format base64
kit secret uuid</code></pre>

      <h2 id="flags">공통 플래그</h2>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>플래그</th>
              <th>설명</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><code>-v</code>, <code>--version</code></td>
              <td>루트에서 kit 버전 출력</td>
            </tr>
            <tr>
              <td><code>--dry-run</code></td>
              <td>외부 명령을 실행하지 않고 preview만 출력</td>
            </tr>
            <tr>
              <td><code>--json</code></td>
              <td>스크립트와 파이프라인에서 쓰기 좋은 JSON 출력</td>
            </tr>
            <tr>
              <td><code>--verbose</code></td>
              <td>상세한 실행 결과 출력</td>
            </tr>
            <tr>
              <td><code>--yes</code></td>
              <td>확인 프롬프트를 건너뛰고 실행</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2 id="api">서버 API</h2>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Endpoint</th>
              <th>내용</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><a href="/install.sh"><code>/install.sh</code></a></td>
              <td>OS와 아키텍처를 감지하는 설치 스크립트</td>
            </tr>
            <tr>
              <td><a href="/uninstall.sh"><code>/uninstall.sh</code></a></td>
              <td>kit 바이너리와 로컬 상태 파일을 제거하는 스크립트</td>
            </tr>
            <tr>
              <td><a href="/version"><code>/version</code></a></td>
              <td>버전, 빌드 정보, 사용 가능한 바이너리 목록</td>
            </tr>
            <tr>
              <td><a href="/stats"><code>/stats</code></a></td>
              <td>macOS/Linux 다운로드 카운트와 path별 카운트. <code>?update=1</code> 바이너리 다운로드는 제외</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p>런타임 캐시 배포 기능은 제거되어 <code>/runtime</code> 계열 엔드포인트를 제공하지 않는다.</p>

      <footer>
        orot-kit &nbsp;·&nbsp; <code>` + baseURL + `</code><br>
        설치: <code>` + install + `</code><br>
        제거: <code>` + uninstall + `</code>
      </footer>
    </main>
  </div>

  <script>
    const links = [...document.querySelectorAll('.sidebar nav a[href^="#"]')];
    const headings = links
      .map((link) => document.querySelector(link.getAttribute('href')))
      .filter(Boolean);

    function setActive(id) {
      links.forEach((link) => {
        const isActive = link.getAttribute('href') === ` + "`#${id}`" + `;
        link.classList.toggle('active', isActive);
      });
    }

    const observer = new IntersectionObserver((entries) => {
      const visible = entries
        .filter((entry) => entry.isIntersecting)
        .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];

      if (visible) {
        setActive(visible.target.id);
      }
    }, {
      root: null,
      threshold: [0.1, 0.25, 0.5, 0.75],
      rootMargin: '-15% 0px -70% 0px'
    });

    headings.forEach((heading) => observer.observe(heading));

    if (headings.length > 0) {
      const initial = headings.find((heading) => {
        const rect = heading.getBoundingClientRect();
        return rect.top >= 0;
      }) || headings[0];

      setActive(initial.id);
    }
  </script>
</body>

</html>
`
}

func renderInstallScript(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return `#!/usr/bin/env sh
set -eu

base="${KIT_BASE_URL:-` + baseURL + `}"
install_dir="${KIT_INSTALL_DIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin|linux) ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

name="kit-$os-$arch"
url="$base/bin/$name"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

mkdir -p "$install_dir"
echo "Downloading $url"
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"
mv "$tmp" "$install_dir/kit"
echo "Installed kit to $install_dir/kit"
echo "Make sure $install_dir is in PATH."
`
}

func renderUninstallScript() string {
	return `#!/usr/bin/env sh
set -u

install_dir="${KIT_INSTALL_DIR:-$HOME/.local/bin}"

remove_path() {
  path="$1"
  kind="$2"
  if [ -z "$path" ]; then
    return
  fi
  if [ ! -e "$path" ] && [ ! -L "$path" ]; then
    echo "Skipped $kind: $path"
    return
  fi
  if rm -rf "$path" 2>/dev/null; then
    echo "Removed $kind: $path"
  else
    echo "Could not remove $kind: $path" >&2
  fi
}

remove_path "${KIT_BIN:-$install_dir/kit}" "binary"
remove_path "$HOME/.local/bin/kit" "binary"
remove_path "$HOME/bin/kit" "binary"

if command -v kit >/dev/null 2>&1; then
  found_kit="$(command -v kit)"
  case "$found_kit" in
    */*) remove_path "$found_kit" "binary" ;;
  esac
fi

remove_path "$HOME/.kit" "state"
remove_path "$HOME/.kit-server" "state"

echo "Uninstall complete."
echo "Remove ~/.kit/shims from PATH in your shell profile if you added it manually."
`
}

const PageHTML = `deprecated: use renderPage`

const InstallScript = `deprecated: use renderInstallScript`

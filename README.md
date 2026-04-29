# orot-kit

`orot-kit`은 macOS와 Linux에서 자주 쓰는 개발, 서버 운영, 파일, 네트워크 명령을 `kit` 하나로 묶는 개인용 터미널 툴킷입니다.

핵심 방향은 단순한 래퍼가 아니라 작업 단위 CLI입니다. 외우기 어려운 명령은 질문형 빌더로 만들고, 실제 실행되는 원본 명령은 항상 출력해서 도구를 쓰면서도 CLI를 학습할 수 있게 합니다.

## 지원 OS

| OS | Architecture |
| --- | --- |
| macOS | arm64, amd64 |
| Linux | arm64, amd64 |

다운로드는 `curl` 기반으로만 처리합니다. 설치 스크립트와 `kit network download` 모두 `curl`을 사용합니다.

## 빠른 시작

```bash
go run . -v
go run . version
go run . info
go run . --dry-run update
go run . --dry-run find nginx --root .
go run . --dry-run archive README.md --format tar.gz --output readme.tar.gz
go run . --dry-run network download http://localhost:8080/bin/kit-linux-amd64 --output kit --executable
go run . resource
go run . network
```

버전 확인:

```bash
kit -v
kit --version
kit version
```

전역 플래그:

```bash
kit --dry-run   # 실제 실행 없이 원본 명령만 확인
kit --json      # JSON 출력
kit --verbose   # 상세 출력
kit --yes       # 안전 확인을 건너뛰고 실행
```

## 설치

릴리즈 사이트 또는 로컬 설치 서버가 떠 있는 경우:

```bash
curl -fsSL http://localhost:8080/install.sh | sh
```

설치 위치를 바꾸려면:

```bash
curl -fsSL http://localhost:8080/install.sh | KIT_INSTALL_DIR=~/bin sh
```

파일만 받을 때는 `kit network download`를 사용합니다.

```bash
kit network download http://localhost:8080/bin/kit-linux-amd64 --output kit --executable
kit network download http://localhost:8080/bin/kit-linux-amd64 --output kit --sha256 <sha256>
```

## 업데이트

설치 서버의 현재 OS/Arch 바이너리로 실행 중인 `kit`을 교체합니다. 업데이트용 바이너리 다운로드는 서버 다운로드 카운트에 포함하지 않습니다.

```bash
kit update
kit update --dry-run
kit update --base-url http://localhost:8080
```

기본 업데이트 서버는 `KIT_BASE_URL`, `KIT_INSTALL_BASE_URL`, `~/.kit/config.yaml`의 `server.install_base_url`, `http://localhost:8080` 순서로 결정합니다. 현재 실행 파일이 아닌 다른 경로를 교체하려면 `--bin`을 사용합니다.

```bash
kit update --bin ~/.local/bin/kit
```

## 제거

CLI에서 먼저 삭제 대상을 확인할 수 있습니다.

```bash
kit uninstall --dry-run
kit uninstall --yes
```

기본 제거 대상:

- 현재 실행 중인 `kit` 바이너리
- `~/.local/bin/kit`
- `~/bin/kit`
- `/usr/local/bin/kit`
- `~/.kit`
- `~/.kit-server`

설정이나 서버 캐시를 남기려면:

```bash
kit uninstall --keep-config --keep-server --yes
```

설치 서버가 떠 있는 경우 원격 스크립트로도 제거할 수 있습니다.

```bash
curl -fsSL http://localhost:8080/uninstall.sh | sh
```

## 빌드

```bash
make build       # 현재 OS/Arch용 bin/kit 생성
make dist        # macOS/Linux amd64/arm64 배포 바이너리 생성
make check       # gofmt, go vet, go test
```

개별 검증:

```bash
go test ./...
go vet ./...
go build ./...
```

## 주요 기능

## 변경 사항

- Runtime Manager 기능을 제거했습니다. `runtime` 명령과 `node`, `go`, `python`, `java` 런타임 단축 명령은 더 이상 제공하지 않습니다.
- Docker 관리 기능을 제거했습니다. 서비스 관리는 `systemctl`과 Homebrew services 중심으로 단순화했습니다.
- 설치 서버에서 런타임 캐시 API를 제거했습니다. `/runtime` 계열 엔드포인트와 `--runtime-cache-dir` 옵션은 더 이상 사용하지 않습니다.

### 파일 탐색

```bash
kit ls .
kit ls ..
kit ls ../..
kit ls ./src
kit tree . --depth 2
kit size .
kit find nginx --root .
kit find "*.go" --root . --type file
```

`kit find`는 패턴에 와일드카드가 없으면 자동으로 `*pattern*` 형태로 검색합니다.

### 압축과 해제

```bash
kit archive ./dist --format tar.gz --output dist.tar.gz
kit archive ./dist --format zip --output dist.zip
kit archive ./logs --format tar.gz --output logs.tar.gz
kit extract dist.tar.gz --dest ./out
kit extract dist.zip ./out
```

지원 포맷: `tar.gz`, `tgz`, `zip`, `gzip`.

### 네트워크

```bash
kit network
kit network ip
kit network ping example.com --count 4
kit network dig example.com
kit network curl https://example.com --method GET
kit network download https://example.com/file.tar.gz --output file.tar.gz
kit network port
kit network port kill 1234 --yes
kit network tcpdump --interface eth0 --port 443 --count 50 --dry-run
```

`tcpdump`, `port kill`처럼 권한이나 위험이 있는 작업은 실행 전에 확인 절차를 둡니다.

### 리소스 점검

```bash
kit resource
kit resource disk
kit resource memory
kit resource process
kit resource logs
kit resource logs nginx
```

OS에서 사용할 수 있는 `uname`, `uptime`, `df`, `free`, `ps`, `journalctl` 등을 조합해 요약합니다.

### Git과 Diff

```bash
kit git status
kit git position
kit git diff
kit git diff --stat
kit git diff --name-only
kit git diff --staged
kit git diff --against main
kit git diff --base origin/main
kit diff README.md
kit diff old.go new.go
kit diff old.go new.go --context 5
```

Git 기능은 저장소를 변경하지 않는 조회 중심 명령으로 구성합니다.

### 서비스

```bash
kit service list
kit service nginx status
kit service nginx logs --tail 200
kit service nginx restart
kit service add nginx --type systemctl --name nginx
```

서비스 alias는 `~/.kit/config.yaml`에 저장되며, `systemctl`과 Homebrew services 흐름을 지원합니다.

### SSH와 전송

```bash
kit ssh add orbit --host 10.0.0.10 --user deploy --port 2222 --identity ~/.ssh/orbit
kit ssh keygen ~/.ssh/orbit
kit ssh copy deploy@example.com

kit send --server orbit --local ./dist --remote /srv/app
kit receive --server orbit --remote /srv/app/logs --local ./logs
kit sync --server orbit --local ./dist --remote /srv/app
```

등록된 SSH host 정보는 `send`, `receive`, `sync`에서 재사용됩니다.

### Firewall

```bash
kit fw status
kit fw list
kit fw open 8080 --dry-run
kit fw close 8080 --dry-run
```

Linux에서는 `ufw` 또는 `firewall-cmd`를 감지해 명령을 구성합니다. macOS의 `pfctl` 변경은 자동화하지 않고 안내 오류를 반환합니다.

### Secret

```bash
kit secret password --length 32
kit secret api-key --prefix orot --length 32 --no-print
kit secret jwt --format env --key JWT_SECRET --no-print
kit secret uuid
kit secret env DATABASE_URL --length 48
```

`--no-print`를 사용하면 민감한 값을 화면에 출력하지 않습니다.

## 설정 파일

기본 설정 경로는 `~/.kit/config.yaml`입니다.

예시:

```yaml
language: ko

output:
  show_command: true
  format: text

server:
  install_base_url: http://localhost:8080

ssh:
  hosts:
    orbit:
      host: 10.0.0.10
      user: deploy
      port: 2222
      identity_file: ~/.ssh/orbit

services:
  nginx:
    type: systemctl
    name: nginx
```

## 설치 서버와 문서 사이트

배포 바이너리와 설치 페이지를 로컬 또는 서버에서 제공합니다.

```bash
make serve
make serve-status
make serve-stop
```

직접 실행:

```bash
kit install-server \
  --addr :8080 \
  --bin-dir dist \
  --assets-dir assets \
  --stats-file ~/.kit-server/download-stats.json \
  --base-url http://localhost:8080
```

제공 엔드포인트:

| Path | 설명 |
| --- | --- |
| `/` | 명령어 문서와 설치 안내 페이지 |
| `/install.sh` | OS/Arch를 감지하는 curl 설치 스크립트 |
| `/uninstall.sh` | kit 바이너리와 로컬 상태 파일을 제거하는 스크립트 |
| `/bin/<kit binary>` | 배포 바이너리 |
| `/bin/<kit binary>/checksum` | 바이너리 SHA256 |
| `/version` | 버전, 빌드 정보, 바이너리 메타데이터 |
| `/stats` | 실제 GET 다운로드 집계 |
| `/healthz` | 상태 확인 |

다운로드 통계는 실제 바이너리 파일 `GET` 요청만 집계합니다. `HEAD`, `/install.sh`, `/version`, `/stats` 조회와 `kit update`가 사용하는 `/bin/<kit binary>?update=1` 요청은 다운로드 수에 포함하지 않습니다.

런타임 캐시 배포 기능은 제거되어 `/runtime` 계열 엔드포인트를 제공하지 않습니다.

## 개발 메모

- 외부 명령을 실행하는 기능은 결과에 원본 명령을 함께 출력합니다.
- 위험한 작업은 기본적으로 확인을 요구하고, 자동화가 필요할 때만 `--yes`를 사용합니다.
- 질문형 빌더는 인자가 부족할 때 필요한 값을 순서대로 묻습니다.
- 네트워크 다운로드는 `curl`을 표준 경로로 사용합니다.
- 테스트 캐시는 Makefile 기본값처럼 `/tmp/orot-kit-gocache`, `/tmp/orot-kit-gomodcache`를 쓰면 샌드박스 환경에서도 안정적으로 동작합니다.

## QA 체크

릴리즈 전 최소 확인:

```bash
make check
make dist
make serve SERVE_ADDR=:8090 SERVE_BASE_URL=http://localhost:8090
curl -sS http://localhost:8090/healthz
curl -sS http://localhost:8090/version
curl -sS http://localhost:8090/install.sh
curl -sS http://localhost:8090/uninstall.sh
curl -sS -I http://localhost:8090/bin/kit-linux-amd64
curl -sS http://localhost:8090/stats
make serve-stop
```

CLI smoke test:

```bash
bin/kit --help
bin/kit -v
bin/kit update --dry-run
bin/kit --dry-run find nginx --root . --type file
bin/kit --dry-run archive README.md --format tar.gz --output readme.tar.gz
bin/kit --dry-run network download http://localhost:8080/bin/kit-linux-amd64 --output kit --executable
bin/kit uninstall --dry-run
```

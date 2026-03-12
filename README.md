# connect-dbms

[English README](./README.en.md)

여러 RDBMS에 하나의 바이너리로 접속할 수 있는 Go 기반 TUI + CLI 데이터베이스 클라이언트입니다.

## 개요

`connect-dbms`는 터미널에서 바로 실행하는 데이터베이스 클라이언트입니다. 인자 없이 실행하면 TUI가 열리고, 서브커맨드를 사용하면 스크립트, 자동화, 에이전트 환경에서 비대화형으로 사용할 수 있습니다.

주요 방향은 다음과 같습니다.

- 기본 빌드에 포함되는 드라이버는 pure Go 중심이라 `CGO_ENABLED=0` 빌드가 가능합니다.
- TUI와 CLI가 같은 내부 패키지를 공유합니다.
- 설정은 JSON 파일로 저장되며 사람이 직접 읽고 수정할 수 있습니다.
- DB 에러는 구조화된 포맷으로 출력되어 문제 추적이 쉽습니다.

## 현재 구현된 기능

현재 코드 기준으로 사용 가능한 핵심 기능입니다.

- 여러 DB 드라이버 지원: MySQL, MariaDB, PostgreSQL, Oracle, SQLite
- 기본 TUI 실행: 저장 세션 선택, 빠른 연결, SQL 실행, 결과 조회
- Query history: 최근 실행 SQL 자동 저장 및 검색
- TUI export: 현재 결과를 파일로 저장
- Multiple query tabs: F6으로 새 탭, F7/F8로 탭 이동
- Autocomplete: Ctrl+Space/F9로 table-name 자동완성
- SQL preview highlighting: preview 패널에서 SQL 키워드 하이라이트
- SSH tunnel support: 세션에 SSH jump host 정보를 넣어 터널링 연결 가능
- 테이블 브라우저: 테이블 목록 확인, 컬럼 정보 확인, 선택 테이블용 `SELECT * ... LIMIT 100` 초안 생성
- 비대화형 SQL 실행: `query` 커맨드로 `table`, `json`, `csv`, `tsv` 출력
- 세션 관리: `config add`, `edit`, `remove`, `list`, `show`, `path`
- 연결 확인: `connect`
- 지원 드라이버 목록 확인: `drivers`
- 버전 및 빌드 정보 확인: `version`
- 드라이버별, 네트워크별 구조화 에러 포맷

## 지원 데이터베이스

| Driver | 상태 | 비고 |
| --- | --- | --- |
| `mysql` | Working | `github.com/go-sql-driver/mysql` |
| `mariadb` | Working | MySQL 드라이버 alias |
| `postgres` | Working | `pgx/v5` 기반 |
| `postgresql` | Working | PostgreSQL alias |
| `oracle` | Working | `go-ora` 기반 pure Go |
| `sqlite` | Working | `modernc.org/sqlite`, no CGO |
| `tibero` | Stub / Optional | build tag 필요, ODBC 기반 |
| `cubrid` | Stub / Optional | build tag 필요, ODBC 기반 |

## 왜 이 프로젝트인가

여러 종류의 DB를 다룰 때 도구를 매번 바꾸지 않고, 하나의 실행 파일과 하나의 설정 파일로 일관되게 연결, 조회, 테스트하려는 목적에 맞춰져 있습니다.

## 빠른 시작

권장 환경:

- Go 1.24 이상
- macOS, Linux, Windows, Termux 중 하나
- 기본 드라이버 사용 시 추가 클라이언트 라이브러리 없이 시작 가능

### 1. 빌드

```bash
go build -ldflags "-s -w" -o connect-dbms .
```

또는 Makefile을 사용할 수 있습니다.

```bash
make build
```

### 2. 드라이버 확인

```bash
./connect-dbms drivers
```

### 3. 세션 추가

PostgreSQL 예시:

```bash
./connect-dbms config add \
  --name local-pg \
  --driver postgres \
  --host 127.0.0.1 \
  --port 5432 \
  --user postgres \
  --database mydb
```

SQLite 예시:

```bash
./connect-dbms config add \
  --name local-sqlite \
  --driver sqlite \
  --database ./data.db
```

DSN 직접 지정 예시:

```bash
./connect-dbms config add \
  --name prod-mysql \
  --driver mysql \
  --dsn "user:pass@tcp(db.example.com:3306)/app?parseTime=true"
```

### 4. 연결 테스트

```bash
./connect-dbms connect local-pg
```

또는 세션 없이 직접:

```bash
./connect-dbms connect --driver sqlite --dsn ./data.db
```

### 5. SQL 실행

```bash
./connect-dbms query --profile local-pg --sql "SELECT now()" --format table
```

파일 입력:

```bash
./connect-dbms query --profile local-pg --file query.sql --format json
```

파이프 입력:

```bash
echo "SELECT 1" | ./connect-dbms query --profile local-pg --format csv
```

### 6. TUI 실행

```bash
./connect-dbms
```

인자 없이 실행하면 Alt Screen 기반 TUI가 열립니다.

## TUI 사용법

TUI는 크게 세 가지 흐름으로 구성됩니다.

### 연결 화면

- 왼쪽: 저장된 세션 목록
- 오른쪽: 빠른 연결 폼
- `Enter`: 연결
- `Tab`: 왼쪽, 오른쪽 패널 전환
- `Delete`: 저장 세션 삭제

### 쿼리 화면

- 상단: SQL 에디터
- 하단: 결과 테이블 또는 실행 상태
- `F5` 또는 `Ctrl+E`: SQL 실행
- `Ctrl+H`: 저장된 query history 검색 및 불러오기
- `Ctrl+S`: 현재 결과 export
- `Ctrl+Space` 또는 `F9`: 자동완성 열기
- `F6`: 새 query tab 열기
- `F7`, `F8`: 다음, 이전 query tab 이동
- `Tab`: 에디터, 결과 패널 전환
- `PgUp`, `PgDn`: 결과 스크롤

### 테이블 화면

- `Ctrl+T`: 테이블 브라우저 열기
- 테이블 목록과 컬럼 정보를 함께 확인
- `Enter`: 선택한 테이블 기준 조회 SQL 초안 생성
- `Esc`: 쿼리 화면으로 돌아가기

### 공통 키

- `F1`: 도움말
- `Ctrl+N`: 새 연결
- `Ctrl+Q`: 종료

## CLI 사용법

### 기본 도움말

```bash
./connect-dbms --help
```

### 주요 커맨드

```bash
./connect-dbms
./connect-dbms config
./connect-dbms config list --json
./connect-dbms config show local-pg
./connect-dbms connect local-pg
./connect-dbms query --profile local-pg --sql "SELECT * FROM users"
./connect-dbms version
./connect-dbms drivers
```

참고:

- `connect-dbms config`를 인자 없이 실행하면 세션 관리 전용 TUI가 열립니다.
- `connect-dbms query`는 `--profile` 또는 `--driver` + `--dsn` 조합으로 실행합니다.
- config TUI의 드라이버 순서는 `mysql`, `mariadb`, `oracle`, `postgresql`, `tibero`, `cubrid`, `sqlite`입니다.
- config TUI에서 `sqlite`를 선택하면 `Database` 대신 `File Path` 의미로 사용하며, 예시는 `./data.db`, `/tmp/app.db` 같은 경로입니다.
- `sqlite`에서는 `Host`, `Port`, `User`, `Password` 값이 사용되지 않습니다.
- pooling 설정은 `max_open_conns`, `max_idle_conns`, `conn_max_lifetime_seconds` 필드와 CLI 플래그로 지정할 수 있습니다.
- SSH 설정은 세션의 `ssh_host`, `ssh_port`, `ssh_user`, `ssh_password`, `ssh_key_path` 필드 또는 config TUI/CLI로 지정할 수 있습니다.

### `query` 출력 포맷

- `table`: 사람이 읽기 쉬운 표 형식
- `json`: 자동화 친화적 JSON 배열
- `csv`: 쉼표 구분
- `tsv`: 탭 구분

## 설정 파일

기본 위치:

```text
~/.config/connect-dbms/config.json
```

다른 경로를 쓰려면 전역 플래그 `--config`를 사용할 수 있습니다.

최초 실행 시 설정 파일이 없으면 빈 `config.json`을 자동 생성합니다.

예시:

```json
{
  "profiles": [
    {
      "name": "my-postgres",
      "driver": "postgres",
      "host": "localhost",
      "port": 5432,
      "user": "admin",
      "password": "secret",
      "database": "mydb"
    },
    {
      "name": "raw-dsn-example",
      "driver": "mysql",
      "dsn": "user:pass@tcp(host:3306)/db?parseTime=true"
    }
  ]
}
```

중요:

- `dsn`이 있으면 `host`, `port`, `user`, `password`, `database`보다 우선합니다.
- 비밀번호는 평문 JSON으로 저장됩니다.

## 빌드

### 개발 빌드

```bash
make build
```

### 크로스 컴파일

```bash
make build-all
```

출력 파일은 `dist/` 아래에 생성됩니다.

### 버전 정보 주입

```bash
make build VERSION=1.0.0
```

### GoReleaser

```bash
goreleaser release --snapshot --clean
```

기본판(pure-Go) 설정 파일은 `.goreleaser.yaml`입니다.

확장판(ODBC full) 릴리즈는 아래 설정을 사용합니다.

```bash
goreleaser release --snapshot --clean --config .goreleaser-odbc.yaml
```

파일:
- 기본판: `.goreleaser.yaml`
- 확장판: `.goreleaser-odbc.yaml`

릴리즈 성격:
- 기본판: pure-Go 중심, 바로 실행 가능한 기본 DB 지원
- 확장판: macOS/Linux 전용, Tibero/Cubrid 포함 ODBC 기능 활성화

GitHub Actions:
- `.github/workflows/release.yml`에서 태그 푸시(`v*`) 시 기본판과 확장판 릴리즈를 분리 배포합니다.

### 선택 빌드 태그

```bash
make build-odbc
```

ODBC 기반 통합이 필요한 경우에만 사용하는 선택 빌드입니다.

추가 요구 사항:

- `tibero`, `cubrid`는 ODBC 드라이버와 ODBC 개발 헤더가 필요합니다.
- macOS, Linux에서는 보통 `unixODBC` 같은 시스템 라이브러리 설치가 필요합니다.
- 이 경로는 `CGO_ENABLED=0` 대상이 아닙니다.
- 확장판 바이너리라도 대상 머신에는 벤더 ODBC 드라이버가 실제로 설치되어 있어야 합니다.

macOS + Homebrew 예시:

```bash
brew install unixodbc
make build-odbc
make test-odbc
```

Linux(Fedora, RHEL 계열) 예시:

```bash
sudo dnf install unixODBC unixODBC-devel -y
make build-odbc
make test-odbc
```

설명:

- `unixODBC`는 런타임 라이브러리입니다.
- `unixODBC-devel`은 `sql.h` 같은 헤더를 제공하므로 빌드에 필요합니다.
- 실제 연결에는 DB 벤더의 ODBC 드라이버도 별도로 설치되어 있어야 합니다.
```

## 아키텍처 요약

프로젝트는 Cobra 기반 CLI와 Bubble Tea 기반 TUI가 `internal/` 패키지를 공유하는 구조입니다.

```text
main.go
cmd/
  root.go
  config.go
  connect.go
  query.go
  tui.go
  version.go
internal/
  db/
  tui/
  profile/
  export/
  dberr/
  history/
```

핵심 패키지:

- `internal/db`: 드라이버 레지스트리, 공통 쿼리 실행, 메타데이터 조회 인터페이스
- `internal/tui`: 화면 라우팅, 연결, 쿼리, 테이블, 도움말 UI
- `internal/profile`: JSON 설정 파일 저장소
- `internal/export`: `table`, `json`, `csv`, `tsv` 출력 포맷
- `internal/history`: SQLite 기반 query history 저장소
- `internal/dberr`: 구조화 DB 에러 래핑

## 에러 출력 형식

DB 관련 실패는 사람이 읽기 쉬운 멀티라인 포맷으로 stderr에 출력됩니다.

예시:

```text
[ERROR] 2026-03-12 13:50:50.827 KST
  Driver : postgres
  Phase  : ping
  Host   : localhost
  Code   : PG-CONN
  Message: connection refused
  Raw    : <original error>
  Trace  :
    <stack frames>
```

대표 코드 규칙:

- MySQL: `MYSQL-{errno}`
- PostgreSQL: `PG-{SQLSTATE}`
- Oracle: `ORA-{code}`
- SQLite: `SQLITE_{NAME}({code})`
- 네트워크: `CONN-REFUSED`, `CONN-TIMEOUT`, `CONN-DNS`, `NET-*`

## 현재 한계와 예정 작업

현재 저장소 상태에서 아직 비어 있거나 미완성인 영역입니다.

- Tibero, Cubrid 메타데이터 보강: 부분 구현 상태
- autocomplete 고도화: column/context-aware 추천 강화
- syntax highlighting 고도화: editor 본문 렌더링 정밀도 향상
- 테스트 범위 확대: 추가 필요

## 테스트

현재는 history, pooling helper, query export helper, TUI 설정 일부에 대한 테스트가 들어 있습니다. Makefile에는 아래 타깃이 준비되어 있습니다.

```bash
make test
```

## 개발 시 참고

- Go module 이름은 `connect-dbms`가 아니라 `oslo`입니다.
- import path는 `oslo/internal/...`를 사용합니다.
- SQLite는 `modernc.org/sqlite`를 사용하므로 CGO가 필요 없습니다.
- TUI는 Alt Screen을 사용하므로 전체 터미널을 점유합니다.

## 라이선스

현재 저장소에는 별도의 라이선스 파일이 보이지 않습니다. 공개 배포 전에는 라이선스 정책을 명확히 정하는 것을 권장합니다.

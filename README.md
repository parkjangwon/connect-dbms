# connect-dbms

[English README](./README.en.md)

여러 RDBMS에 하나의 바이너리로 접속할 수 있는 터미널 기반 DB 클라이언트입니다.  
인자 없이 실행하면 TUI가 열리고, 서브커맨드로는 스크립트용 CLI처럼 사용할 수 있습니다.

## 무엇이 되나

- TUI 접속/조회
- 저장 세션 관리
- SQL 실행
- query history 검색
- 결과 export
- 여러 query tab
- SSH tunnel 기반 접속

## 지원 데이터베이스

| Driver | 상태 | 비고 |
| --- | --- | --- |
| `mysql` | 지원 | MariaDB 포함 |
| `mariadb` | 지원 | MySQL alias |
| `postgres` | 지원 | PostgreSQL |
| `postgresql` | 지원 | PostgreSQL alias |
| `oracle` | 지원 | pure Go |
| `sqlite` | 지원 | 파일 기반 / 메모리 DB |
| `tibero` | 확장판에서 지원 | ODBC 필요 |
| `cubrid` | 확장판에서 지원 | ODBC 필요 |

## 배포판 종류

### 기본판

- pure-Go 중심
- 바로 쓰기 쉬운 기본 배포판
- MySQL, MariaDB, PostgreSQL, Oracle, SQLite 대상

### 확장판

- macOS/Linux용 ODBC 포함 배포판
- Tibero, Cubrid까지 포함
- 대상 머신에 ODBC 드라이버는 별도로 설치되어 있어야 함

## 빠른 시작

### 1. 실행

```bash
./connect-dbms
```

원라인 설치/업데이트:

```bash
curl -fsSL https://raw.githubusercontent.com/parkjangwon/connect-dbms/master/install.sh | sh
```

확장판 설치:

```bash
curl -fsSL https://raw.githubusercontent.com/parkjangwon/connect-dbms/master/install.sh | CONNECT_DBMS_CHANNEL=odbc sh
```

삭제:

```bash
curl -fsSL https://raw.githubusercontent.com/parkjangwon/connect-dbms/master/install.sh | sh -s -- uninstall
```

### 2. 세션 추가

PostgreSQL:

```bash
./connect-dbms config add \
  --name local-pg \
  --driver postgresql \
  --host 127.0.0.1 \
  --port 5432 \
  --user postgres \
  --database mydb
```

SQLite:

```bash
./connect-dbms config add \
  --name local-sqlite \
  --driver sqlite \
  --database ./data.db
```

SSH tunnel 포함 예시:

```bash
./connect-dbms config add \
  --name prod-pg \
  --driver postgresql \
  --host 10.0.0.12 \
  --port 5432 \
  --user app \
  --database appdb \
  --ssh-host bastion.example.com \
  --ssh-user ec2-user \
  --ssh-key-path ~/.ssh/id_rsa
```

### 3. 연결 확인

```bash
./connect-dbms connect local-pg
```

### 4. SQL 실행

```bash
./connect-dbms query --profile local-pg --sql "SELECT now()" --format table
```

## 설정 파일

- 위치: `~/.config/connect-dbms/config.json`
- 최초 실행 시 파일이 없으면 자동 생성
- 비밀번호는 평문 JSON으로 저장

SQLite 세션은 `database` 필드에 DB 파일 경로를 넣으면 됩니다.

## TUI 키

### 공통

- `F1`: 도움말
- `Ctrl+Q`: 종료
- `Ctrl+N`: 새 연결

### 쿼리 화면

- `F5` 또는 `Ctrl+E`: SQL 실행
- `Ctrl+H`: query history
- `Ctrl+S`: 결과 export
- `Ctrl+Space` 또는 `F9`: 자동완성
- `F6`: 새 탭
- `F7`, `F8`: 탭 이동
- `Ctrl+T`: 테이블 브라우저

## 빌드

기본 빌드:

```bash
make build
```

ODBC 확장 빌드:

```bash
make build-odbc
```

ODBC 테스트:

```bash
make test-odbc
```

## 릴리즈

- 기본판 설정: `.goreleaser.yaml`
- 확장판 설정: `.goreleaser-odbc.yaml`
- GitHub Actions: `.github/workflows/release.yml`

## 참고

- 자동완성과 하이라이팅은 현재 실사용 가능한 1차 버전입니다.
- Tibero/Cubrid는 확장판 + 대상 머신의 ODBC 드라이버 설치가 필요합니다.

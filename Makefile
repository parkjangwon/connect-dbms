VERSION ?= 1.0.3
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BASE_LDFLAGS := -s -w \
	-X oslo/cmd.Version=$(VERSION) \
	-X oslo/cmd.BuildDate=$(BUILD_DATE) \
	-X oslo/cmd.GitCommit=$(GIT_COMMIT)
LDFLAGS := $(BASE_LDFLAGS) -X oslo/cmd.Edition=basic
ODBC_LDFLAGS_RELEASE := $(BASE_LDFLAGS) -X oslo/cmd.Edition=odbc-full
UNAME_S := $(shell uname -s)
ODBC_TAGS := tibero cubrid

ifeq ($(UNAME_S),Darwin)
ODBC_PREFIX ?= $(shell brew --prefix unixodbc 2>/dev/null)
ODBC_CFLAGS ?= $(if $(ODBC_PREFIX),-I$(ODBC_PREFIX)/include,)
ODBC_LDFLAGS ?= $(if $(ODBC_PREFIX),-L$(ODBC_PREFIX)/lib,)
else
ODBC_CFLAGS ?= $(shell pkg-config --cflags odbc 2>/dev/null)
ODBC_LDFLAGS ?= $(filter-out -lodbc,$(shell pkg-config --libs odbc 2>/dev/null))
endif

ODBC_ENV = CGO_CFLAGS="$(ODBC_CFLAGS)" CGO_LDFLAGS="$(ODBC_LDFLAGS)"

.PHONY: build build-all build-odbc clean test test-odbc test-tibero test-cubrid release-basic release-odbc

build:
	go build -ldflags "$(LDFLAGS)" -o connect-dbms .

build-all: build-linux build-macos build-windows

build-odbc:
	$(ODBC_ENV) go build -tags "$(ODBC_TAGS)" -ldflags "$(ODBC_LDFLAGS_RELEASE)" -o connect-dbms .

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/connect-dbms-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/connect-dbms-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "$(LDFLAGS)" -o dist/connect-dbms-linux-armv7 .

build-macos:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/connect-dbms-darwin-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/connect-dbms-darwin-amd64 .

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/connect-dbms-windows-amd64.exe .

clean:
	rm -f connect-dbms
	rm -rf dist/

test:
	go test ./...

test-odbc: test-tibero test-cubrid

test-tibero:
	$(ODBC_ENV) go test -tags tibero ./internal/db

test-cubrid:
	$(ODBC_ENV) go test -tags cubrid ./internal/db

release-basic:
	goreleaser release --clean

release-odbc:
	goreleaser release --clean --config .goreleaser-odbc.yaml

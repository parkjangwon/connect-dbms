# Local Patches

This repository uses a local `replace` for `github.com/alexbrainman/odbc`.

Applied Darwin-specific adjustments:

- Removed the hardcoded Homebrew Intel path `/usr/local/opt/unixodbc/...`
  from `api/api_unix.go`
- Removed duplicate Darwin `-lodbc` linkage from `api/zapi_unix.go`

Reason:

- avoid Apple Silicon linker warnings
- let the parent build supply `CGO_CFLAGS` / `CGO_LDFLAGS`

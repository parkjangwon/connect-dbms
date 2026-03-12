package dberr

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBError wraps a database error with structured info for issue tracking.
type DBError struct {
	Driver    string
	Phase     string // "open", "ping", "query", "exec"
	Code      string // driver-specific error code
	Message   string // human-readable message
	Detail    string // extra detail (e.g. PG detail, hint)
	RawError  error  // original unwrapped error
	Stack     string // stack trace
	Timestamp time.Time
	Host      string // connection target
}

func (e *DBError) Error() string {
	return e.Message
}

func (e *DBError) Unwrap() error {
	return e.RawError
}

// Format produces a multi-line error report for terminal output.
func (e *DBError) Format() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[ERROR] %s\n", e.Timestamp.Format("2006-01-02 15:04:05.000 MST")))
	sb.WriteString(fmt.Sprintf("  Driver : %s\n", e.Driver))
	sb.WriteString(fmt.Sprintf("  Phase  : %s\n", e.Phase))
	if e.Host != "" {
		sb.WriteString(fmt.Sprintf("  Host   : %s\n", e.Host))
	}
	if e.Code != "" {
		sb.WriteString(fmt.Sprintf("  Code   : %s\n", e.Code))
	}
	sb.WriteString(fmt.Sprintf("  Message: %s\n", e.Message))
	if e.Detail != "" {
		sb.WriteString(fmt.Sprintf("  Detail : %s\n", e.Detail))
	}
	sb.WriteString(fmt.Sprintf("  Raw    : %s\n", e.RawError.Error()))
	sb.WriteString(fmt.Sprintf("  Trace  :\n%s", indentLines(e.Stack, "    ")))

	return sb.String()
}

// Wrap creates a DBError from a raw error, extracting driver-specific info.
func Wrap(driver, phase, host string, err error) *DBError {
	if err == nil {
		return nil
	}

	dbe := &DBError{
		Driver:    driver,
		Phase:     phase,
		Host:      host,
		RawError:  err,
		Timestamp: time.Now(),
		Stack:     captureStack(3),
	}

	// Try to extract driver-specific error code and detail
	switch {
	case extractMySQL(err, dbe):
	case extractPostgres(err, dbe):
	case extractOracle(err, dbe):
	case extractNet(err, dbe):
	default:
		dbe.Code = classifyGeneric(err)
		dbe.Message = err.Error()
	}

	return dbe
}

// --- MySQL / MariaDB ---

func extractMySQL(err error, dbe *DBError) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		dbe.Code = fmt.Sprintf("MYSQL-%d", mysqlErr.Number)
		dbe.Message = mysqlErr.Message
		return true
	}
	// go-sql-driver connection errors often contain "dial tcp" or similar
	if dbe.Driver == "mysql" || dbe.Driver == "mariadb" {
		msg := err.Error()
		if strings.Contains(msg, "dial tcp") || strings.Contains(msg, "connection refused") {
			dbe.Code = "MYSQL-CONN"
			dbe.Message = msg
			return true
		}
		if strings.Contains(msg, "Access denied") {
			dbe.Code = "MYSQL-1045"
			dbe.Message = msg
			return true
		}
	}
	return false
}

// --- PostgreSQL ---

func extractPostgres(err error, dbe *DBError) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		dbe.Code = fmt.Sprintf("PG-%s", pgErr.Code)
		dbe.Message = pgErr.Message
		var details []string
		if pgErr.Detail != "" {
			details = append(details, "Detail: "+pgErr.Detail)
		}
		if pgErr.Hint != "" {
			details = append(details, "Hint: "+pgErr.Hint)
		}
		if pgErr.SchemaName != "" {
			details = append(details, "Schema: "+pgErr.SchemaName)
		}
		if pgErr.TableName != "" {
			details = append(details, "Table: "+pgErr.TableName)
		}
		if pgErr.ColumnName != "" {
			details = append(details, "Column: "+pgErr.ColumnName)
		}
		if pgErr.Where != "" {
			details = append(details, "Where: "+pgErr.Where)
		}
		dbe.Detail = strings.Join(details, "; ")
		return true
	}
	var pgConnErr *pgconn.ConnectError
	if errors.As(err, &pgConnErr) {
		dbe.Code = "PG-CONN"
		dbe.Message = pgConnErr.Error()
		return true
	}
	if dbe.Driver == "postgres" || dbe.Driver == "postgresql" {
		msg := err.Error()
		if strings.Contains(msg, "connect") || strings.Contains(msg, "dial") {
			dbe.Code = "PG-CONN"
			dbe.Message = msg
			return true
		}
	}
	return false
}

// --- Oracle ---

func extractOracle(err error, dbe *DBError) bool {
	if dbe.Driver != "oracle" {
		return false
	}
	msg := err.Error()
	// go-ora errors typically contain "ORA-XXXXX"
	if idx := strings.Index(msg, "ORA-"); idx >= 0 {
		// Extract ORA-NNNNN
		end := idx + 4
		for end < len(msg) && msg[end] >= '0' && msg[end] <= '9' {
			end++
		}
		dbe.Code = msg[idx:end]
		dbe.Message = msg
		return true
	}
	// TNS errors
	if strings.Contains(msg, "TNS") {
		dbe.Code = "ORA-TNS"
		dbe.Message = msg
		return true
	}
	if strings.Contains(msg, "dial") || strings.Contains(msg, "connect") || strings.Contains(msg, "refused") {
		dbe.Code = "ORA-CONN"
		dbe.Message = msg
		return true
	}
	return false
}

// --- SQLite ---

// sqliteCodeMap maps numeric SQLite error codes to names.
var sqliteCodeMap = map[string]string{
	"1": "SQLITE_ERROR", "2": "SQLITE_INTERNAL", "3": "SQLITE_PERM",
	"4": "SQLITE_ABORT", "5": "SQLITE_BUSY", "6": "SQLITE_LOCKED",
	"7": "SQLITE_NOMEM", "8": "SQLITE_READONLY", "9": "SQLITE_INTERRUPT",
	"10": "SQLITE_IOERR", "11": "SQLITE_CORRUPT", "12": "SQLITE_NOTFOUND",
	"13": "SQLITE_FULL", "14": "SQLITE_CANTOPEN", "15": "SQLITE_PROTOCOL",
	"17": "SQLITE_SCHEMA", "18": "SQLITE_TOOBIG", "19": "SQLITE_CONSTRAINT",
	"20": "SQLITE_MISMATCH", "21": "SQLITE_MISUSE", "23": "SQLITE_AUTH",
	"26": "SQLITE_NOTADB",
}

func classifySQLite(err error) (string, bool) {
	msg := err.Error()
	// modernc.org/sqlite: "SQL logic error: ... (1)"
	if idx := strings.LastIndex(msg, "("); idx > 0 && strings.HasSuffix(msg, ")") {
		code := msg[idx+1 : len(msg)-1]
		if name, ok := sqliteCodeMap[code]; ok {
			return fmt.Sprintf("%s(%s)", name, code), true
		}
	}
	// Also check for SQLITE_ prefix directly
	if strings.Contains(msg, "SQLITE_") {
		idx := strings.Index(msg, "SQLITE_")
		end := idx
		for end < len(msg) && msg[end] != ':' && msg[end] != ' ' && msg[end] != '(' {
			end++
		}
		return msg[idx:end], true
	}
	return "", false
}

// --- Network / Generic ---

func extractNet(err error, dbe *DBError) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		dbe.Code = "NET-" + strings.ToUpper(netErr.Op)
		dbe.Message = netErr.Error()
		if netErr.Addr != nil {
			dbe.Detail = "Address: " + netErr.Addr.String()
		}
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		dbe.Code = "NET-DNS"
		dbe.Message = dnsErr.Error()
		return true
	}
	return false
}

func classifyGeneric(err error) string {
	if code, ok := classifySQLite(err); ok {
		return code
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"):
		return "CONN-REFUSED"
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return "CONN-TIMEOUT"
	case strings.Contains(msg, "no such host"):
		return "CONN-DNS"
	case strings.Contains(msg, "EOF"):
		return "CONN-EOF"
	case strings.Contains(msg, "reset by peer"):
		return "CONN-RESET"
	case strings.Contains(msg, "authentication") || strings.Contains(msg, "password") || strings.Contains(msg, "Access denied"):
		return "AUTH-FAIL"
	default:
		return "UNKNOWN"
	}
}

func captureStack(skip int) string {
	var sb strings.Builder
	pcs := make([]uintptr, 20)
	n := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		// Skip runtime internals
		if strings.Contains(frame.Function, "runtime.") && !strings.Contains(frame.Function, "runtime/debug") {
			if !more {
				break
			}
			continue
		}
		sb.WriteString(fmt.Sprintf("%s\n  %s:%d\n", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
	return sb.String()
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	var sb strings.Builder
	for _, line := range lines {
		if line == "" {
			continue
		}
		sb.WriteString(prefix)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

//go:build tibero

package db

import "testing"

func TestTiberoOpenUsesRegisteredODBCDriver(t *testing.T) {
	drv, err := Get("tibero")
	if err != nil {
		t.Fatalf("Get(tibero) error = %v", err)
	}

	conn, err := drv.Open(ConnConfig{DSN: "Driver={Tibero 6 ODBC Driver};SERVER=127.0.0.1;PORT=8629;DB=test;UID=test;PWD=test"})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	conn.Close()
}

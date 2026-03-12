//go:build cubrid

package db

import (
	"reflect"
	"testing"
)

func TestCubridOpenUsesRegisteredODBCDriver(t *testing.T) {
	drv, err := Get("cubrid")
	if err != nil {
		t.Fatalf("Get(cubrid) error = %v", err)
	}

	conn, err := drv.Open(ConnConfig{DSN: "Driver={CUBRID Driver};SERVER=127.0.0.1;PORT=33000;DB=test;UID=test;PWD=test"})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	conn.Close()
}

func TestParseCubridForeignKeys(t *testing.T) {
	createSQL := "CREATE TABLE [child] ([id] INTEGER NOT NULL, [parent_id] INTEGER, CONSTRAINT [pk_child] PRIMARY KEY ([id]), CONSTRAINT [fk_child_parent] FOREIGN KEY ([parent_id]) REFERENCES [parent] ([id]), CONSTRAINT [fk_child_pair] FOREIGN KEY ([parent_id], [id]) REFERENCES [parent_pair] ([parent_id], [child_id])) REUSE_OID"

	got := parseCubridForeignKeys(createSQL)
	want := []FKInfo{
		{
			Name:       "fk_child_parent",
			Columns:    []string{"parent_id"},
			RefTable:   "parent",
			RefColumns: []string{"id"},
		},
		{
			Name:       "fk_child_pair",
			Columns:    []string{"parent_id", "id"},
			RefTable:   "parent_pair",
			RefColumns: []string{"parent_id", "child_id"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCubridForeignKeys() = %#v, want %#v", got, want)
	}
}

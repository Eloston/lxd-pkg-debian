package sqlite3

import (
	"database/sql/driver"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Exercise the volatile VFS implementation.
func Test_VolatileVFS(t *testing.T) {
	fs := RegisterVolatileFileSystem("volatile")
	defer UnregisterVolatileFileSystem(fs)

	// Open a connection using the volatile VFS as backend.
	drv := &SQLiteDriver{}
	conni, err := drv.Open("file:test.db?vfs=volatile")
	if err != nil {
		t.Fatal("failed to open connection with volatile VFS", err)
	}
	conn := conni.(*SQLiteConn)

	// Set WAL journaling.
	pragmaWAL(t, conn)

	// Create a test table and insert a few rows into it.
	if _, err := conn.Exec("CREATE TABLE test (n INT)", nil); err != nil {
		t.Fatal("failed to create table on volatile VFS", err)
	}
	tx, err := conn.Begin()
	if err != nil {
		t.Fatal("failed to begin transaction on volatile VFS", err)
	}

	for i := 0; i < 100; i++ {
		_, err = conn.Exec("INSERT INTO test(n) VALUES(?)", []driver.Value{int64(i)})
		if err != nil {
			t.Fatal("failed to insert value on volatile VFS", err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal("failed to commit transaction on volatile VFS", err)
	}

	// Assert that the rows are actually there.
	assertTestTableRows(t, conn, 100)

	// Take a full checkpoint of the volatile database.
	size, ckpt, err := conn.WalCheckpoint("main", WalCheckpointTruncate)
	if err != nil {
		t.Fatal("failed to perform WAL checkpoint on volatile VFS", err)
	}
	if size != 0 {
		t.Fatalf("expected size to be %d, got %d", 0, size)
	}
	if ckpt != 0 {
		t.Fatalf("expected ckpt to be %d, got %d", 0, ckpt)
	}

	// Close the connection to the volatile database.
	if err := conn.Close(); err != nil {
		t.Fatal("failed to close connection on volatile VFS", err)
	}

	// Dump the content of the volatile file system and check that the
	// database data are still intact when queried with a regular
	// connection.
	dir, err := ioutil.TempDir("", "go-sqlite3-volatile-vfs-")
	if err != nil {
		t.Fatal("failed to create temporary directory for VFS dump", err)
	}
	defer os.RemoveAll(dir)
	if err := fs.Dump(dir); err != nil {
		t.Fatal("failed to dump volatile VFS", err)
	}
	conni, err = drv.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal("failed to open connection to dumped volatile database", err)
	}
	conn = conni.(*SQLiteConn)
	assertTestTableRows(t, conn, 100)
	if err := conn.Close(); err != nil {
		t.Fatal("failed to close connection to dumped volatile database", err)
	}
}

func assertTestTableRows(t *testing.T, conn *SQLiteConn, n int) {
	rows, err := conn.Query("SELECT n FROM test", nil)
	if err != nil {
		t.Fatal("failed to query test table", err)
	}
	for i := 0; i < n; i++ {
		values := make([]driver.Value, 1)
		if err := rows.Next(values); err != nil {
			t.Fatal("failed to fetch test table row", err)
		}
		n, ok := values[0].(int64)
		if !ok {
			t.Fatal("expected int64 row value")
		}
		if int(n) != i {
			t.Fatalf("expected row value to be %d, got %d", i, n)
		}
	}
	if err := rows.Close(); err != nil {
		t.Fatal("failed to close test table result set", err)
	}
}

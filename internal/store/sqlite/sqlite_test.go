package sqlite

import (
	"context"
	"testing"
)

func TestNew_CreatesTables(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer db.Close()

	// Verify tables exist by querying sqlite_master
	ctx := context.Background()
	rows, err := db.db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query sqlite_master error: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		tables = append(tables, name)
	}

	expected := []string{"accounts", "attachments", "email_labels", "emails", "emails_fts", "labels", "sync_state"}
	for _, exp := range expected {
		found := false
		for _, tbl := range tables {
			if tbl == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected table %q not found in %v", exp, tables)
		}
	}
}

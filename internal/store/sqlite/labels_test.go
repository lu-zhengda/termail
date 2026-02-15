package sqlite

import (
	"context"
	"testing"

	"github.com/lu-zhengda/termail/internal/domain"
)

func TestUpsertAndListLabels(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	labels := []domain.Label{
		{ID: "lbl-inbox", AccountID: "acc-1", Name: "INBOX", Type: domain.LabelTypeSystem},
		{ID: "lbl-starred", AccountID: "acc-1", Name: "STARRED", Type: domain.LabelTypeSystem},
		{ID: "lbl-custom", AccountID: "acc-1", Name: "Work", Type: domain.LabelTypeUser, Color: "#ff0000"},
	}

	for _, lbl := range labels {
		lbl := lbl
		if err := db.UpsertLabel(ctx, &lbl); err != nil {
			t.Fatalf("UpsertLabel(%s) error: %v", lbl.ID, err)
		}
	}

	got, err := db.ListLabels(ctx, "acc-1")
	if err != nil {
		t.Fatalf("ListLabels() error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListLabels() count = %d, want 3", len(got))
	}

	// Should be ordered by name: INBOX, STARRED, Work
	if got[0].Name != "INBOX" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "INBOX")
	}
	if got[1].Name != "STARRED" {
		t.Errorf("got[1].Name = %q, want %q", got[1].Name, "STARRED")
	}
	if got[2].Name != "Work" {
		t.Errorf("got[2].Name = %q, want %q", got[2].Name, "Work")
	}
	if got[2].Color != "#ff0000" {
		t.Errorf("got[2].Color = %q, want %q", got[2].Color, "#ff0000")
	}

	// Upsert existing label to update it
	updated := &domain.Label{ID: "lbl-custom", AccountID: "acc-1", Name: "Personal", Type: domain.LabelTypeUser, Color: "#00ff00"}
	if err := db.UpsertLabel(ctx, updated); err != nil {
		t.Fatalf("UpsertLabel(update) error: %v", err)
	}

	got, err = db.ListLabels(ctx, "acc-1")
	if err != nil {
		t.Fatalf("ListLabels() after update error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListLabels() count after update = %d, want 3", len(got))
	}

	// Find the updated label
	var found bool
	for _, lbl := range got {
		if lbl.ID == "lbl-custom" {
			found = true
			if lbl.Name != "Personal" {
				t.Errorf("updated label Name = %q, want %q", lbl.Name, "Personal")
			}
			if lbl.Color != "#00ff00" {
				t.Errorf("updated label Color = %q, want %q", lbl.Color, "#00ff00")
			}
		}
	}
	if !found {
		t.Error("updated label lbl-custom not found in list")
	}
}

package domain

import "testing"

func TestThread_MessageCount(t *testing.T) {
	thread := &Thread{Messages: []Email{{ID: "1"}, {ID: "2"}}}
	if got := thread.MessageCount(); got != 2 {
		t.Errorf("MessageCount() = %d, want 2", got)
	}
}

func TestThread_MessageCount_Summary(t *testing.T) {
	thread := &Thread{TotalCount: 5}
	if got := thread.MessageCount(); got != 5 {
		t.Errorf("MessageCount() = %d, want 5 (from TotalCount)", got)
	}
}

func TestThread_IsUnread(t *testing.T) {
	thread := &Thread{Messages: []Email{
		{ID: "1", IsRead: true},
		{ID: "2", IsRead: false},
	}}
	if !thread.IsUnread() {
		t.Error("expected IsUnread() = true when one message is unread")
	}

	allRead := &Thread{Messages: []Email{
		{ID: "1", IsRead: true},
		{ID: "2", IsRead: true},
	}}
	if allRead.IsUnread() {
		t.Error("expected IsUnread() = false when all messages are read")
	}
}

func TestThread_IsUnread_Summary(t *testing.T) {
	thread := &Thread{HasUnread: true}
	if !thread.IsUnread() {
		t.Error("expected IsUnread() = true from HasUnread summary field")
	}

	allRead := &Thread{HasUnread: false}
	if allRead.IsUnread() {
		t.Error("expected IsUnread() = false from HasUnread summary field")
	}
}

package domain

import "testing"

func TestAddress_String(t *testing.T) {
	tests := []struct {
		name string
		addr Address
		want string
	}{
		{"with name", Address{Name: "John", Email: "john@example.com"}, "John <john@example.com>"},
		{"email only", Address{Email: "john@example.com"}, "john@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.String(); got != tt.want {
				t.Errorf("Address.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmail_HasLabel(t *testing.T) {
	e := &Email{Labels: []string{"INBOX", "STARRED"}}
	if !e.HasLabel("INBOX") {
		t.Error("expected HasLabel(INBOX) = true")
	}
	if e.HasLabel("TRASH") {
		t.Error("expected HasLabel(TRASH) = false")
	}
}

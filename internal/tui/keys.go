package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Back          key.Binding
	Compose       key.Binding
	Reply         key.Binding
	ReplyAll      key.Binding
	Forward       key.Binding
	Archive       key.Binding
	Delete        key.Binding
	Star          key.Binding
	Unread        key.Binding
	Label         key.Binding
	Search        key.Binding
	Tab           key.Binding
	Toggle        key.Binding
	SwitchAccount key.Binding
	Help          key.Binding
	Quit          key.Binding
}

var keys = keyMap{
	Up:            key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/\u2191", "up")),
	Down:          key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/\u2193", "down")),
	Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Back:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Compose:       key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compose")),
	Reply:         key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reply")),
	ReplyAll:      key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reply all")),
	Forward:       key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "forward")),
	Archive:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive")),
	Delete:        key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "trash")),
	Star:          key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "star")),
	Unread:        key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unread")),
	Label:         key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "label")),
	Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Tab:           key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	Toggle:        key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "thread/flat")),
	SwitchAccount: key.NewBinding(key.WithKeys("@"), key.WithHelp("@", "account")),
	Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

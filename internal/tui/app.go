package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/termail/internal/domain"
	"github.com/lu-zhengda/termail/internal/provider"
	"github.com/lu-zhengda/termail/internal/store"
)

type pane int

const (
	paneSidebar pane = iota
	paneList
	paneReader
)

type viewMode int

const (
	viewThread viewMode = iota
	viewFlat
)

// --- async result messages ---

type labelsLoadedMsg struct {
	labels []domain.Label
}

type emailsLoadedMsg struct {
	emails []domain.Email
}

type threadsLoadedMsg struct {
	threads []domain.Thread
}

type emailLoadedMsg struct {
	email *domain.Email
}

type threadLoadedMsg struct {
	thread *domain.Thread
}

type searchResultsMsg struct {
	results []domain.Email
}

type emailSentMsg struct{}

type actionDoneMsg struct {
	action string
}

type accountSwitchedMsg struct {
	accountID string
}

type errMsg struct {
	err error
}

// ProviderFactory creates an EmailProvider for the given account ID.
type ProviderFactory func(accountID string) provider.EmailProvider

// --- root model ---

type model struct {
	store           store.Store
	provider        provider.EmailProvider
	providerFactory ProviderFactory
	accountID       string
	accounts        []domain.Account

	sidebar  sidebarModel
	inbox    inboxModel
	reader   readerModel
	composer composerModel
	search   searchModel

	activePane pane
	viewMode   viewMode
	statusBar  statusBar

	width  int
	height int
}

// NewModel creates a new root TUI model.
func NewModel(s store.Store, p provider.EmailProvider, accountID string, accounts []domain.Account, factory ProviderFactory) model {
	inbox := newInbox()
	inbox.focused = true

	sidebar := newSidebar()
	sidebar.accountEmail = accountID

	sb := newStatusBar()
	sb.multiAccount = len(accounts) > 1

	return model{
		store:           s,
		provider:        p,
		providerFactory: factory,
		accountID:       accountID,
		accounts:        accounts,
		activePane:      paneList,
		viewMode:        viewThread,
		sidebar:         sidebar,
		inbox:           inbox,
		reader:          newReader(),
		composer:        newComposer(),
		search:          newSearch(),
		statusBar:       sb,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadLabelsCmd(),
		m.loadMailCmd(domain.LabelInbox),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// --- window resize ---
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.width = msg.Width
		m.resizeSubModels()
		return m, nil

	// --- async result messages ---
	case labelsLoadedMsg:
		m.sidebar.SetLabels(msg.labels)
		m.statusBar.setMessage(fmt.Sprintf("Loaded %d labels", len(msg.labels)))
		return m, nil

	case emailsLoadedMsg:
		m.inbox.SetEmails(msg.emails)
		m.statusBar.setMessage(fmt.Sprintf("Loaded %d emails", len(msg.emails)))
		return m, nil

	case threadsLoadedMsg:
		m.inbox.SetThreads(msg.threads)
		m.statusBar.setMessage(fmt.Sprintf("Loaded %d threads", len(msg.threads)))
		return m, nil

	case emailLoadedMsg:
		if msg.email != nil {
			m.reader.ShowEmail(msg.email)
			m.setFocus(paneReader)
			m.statusBar.readerVisible = true
			m.resizeSubModels()
		}
		return m, nil

	case threadLoadedMsg:
		if msg.thread != nil {
			m.reader.ShowThread(msg.thread)
			m.setFocus(paneReader)
			m.statusBar.readerVisible = true
			m.resizeSubModels()
		}
		return m, nil

	case searchResultsMsg:
		m.search.SetResults(msg.results)
		m.statusBar.setMessage(fmt.Sprintf("Found %d results", len(msg.results)))
		return m, nil

	case emailSentMsg:
		m.composer.Close()
		m.statusBar.setMessage("Email sent")
		m.setFocus(paneList)
		return m, nil

	case actionDoneMsg:
		m.statusBar.setMessage(fmt.Sprintf("Action: %s done", msg.action))
		// Close reader and go back to list after destructive actions.
		if msg.action == "archive" || msg.action == "delete" {
			m.reader.Close()
			m.statusBar.readerVisible = false
			m.setFocus(paneList)
		}
		// Reload current label to reflect changes.
		return m, m.loadMailCmd(m.sidebar.activeLabel)

	case accountSwitchedMsg:
		m.accountID = msg.accountID
		if m.providerFactory != nil {
			m.provider = m.providerFactory(msg.accountID)
		}
		m.sidebar.accountEmail = msg.accountID
		m.sidebar.activeLabel = domain.LabelInbox
		m.sidebar.cursor = 0
		m.inbox.cursor = 0
		m.inbox.offset = 0
		m.reader.Close()
		m.statusBar.readerVisible = false
		m.setFocus(paneList)
		m.statusBar.setMessage(fmt.Sprintf("Switched to %s", msg.accountID))
		return m, tea.Batch(
			m.loadLabelsCmd(),
			m.loadMailCmd(domain.LabelInbox),
		)

	case errMsg:
		m.statusBar.setError(fmt.Sprintf("Error: %v", msg.err))
		return m, nil

	// --- sub-model emitted messages ---
	case labelSelectedMsg:
		m.reader.Close()
		m.statusBar.readerVisible = false
		m.inbox.cursor = 0
		m.inbox.offset = 0
		m.setFocus(paneList)
		m.statusBar.setMessage(fmt.Sprintf("Loading %s...", msg.labelID))
		return m, m.loadMailCmd(msg.labelID)

	case emailSelectedMsg:
		m.statusBar.setMessage("Loading email...")
		return m, tea.Batch(
			m.loadEmailCmd(msg.emailID),
			m.markReadCmd(msg.emailID),
		)

	case threadSelectedMsg:
		m.statusBar.setMessage("Loading thread...")
		return m, tea.Batch(
			m.loadThreadCmd(msg.threadID),
			m.markThreadReadCmd(msg.threadID),
		)

	case emailActionMsg:
		m.statusBar.setMessage(fmt.Sprintf("Performing %s...", msg.action))
		return m, m.performActionCmd(msg.emailID, msg.action)

	case replyMsg:
		m.composer.Reply(msg.email, msg.replyAll)
		m.resizeComposer()
		return m, nil

	case forwardMsg:
		m.composer.Forward(msg.email)
		m.resizeComposer()
		return m, nil

	case closeReaderMsg:
		m.reader.Close()
		m.statusBar.readerVisible = false
		m.setFocus(paneList)
		return m, nil

	case sendMsg:
		m.statusBar.setMessage("Sending email...")
		return m, m.sendEmailCmd(msg.email)

	case cancelComposeMsg:
		m.composer.Close()
		m.setFocus(paneList)
		return m, nil

	case searchQueryMsg:
		m.statusBar.setMessage(fmt.Sprintf("Searching: %s", msg.query))
		return m, m.searchCmd(msg.query)

	case searchResultSelectedMsg:
		m.search.Close()
		m.statusBar.setMessage("Loading email...")
		return m, tea.Batch(
			m.loadEmailCmd(msg.emailID),
			m.markReadCmd(msg.emailID),
		)

	case closeSearchMsg:
		m.search.Close()
		m.setFocus(paneList)
		return m, nil

	// --- key events ---
	case tea.KeyMsg:
		// Composer gets all key events when visible.
		if m.composer.IsVisible() {
			var cmd tea.Cmd
			m.composer, cmd = m.composer.Update(msg)
			return m, cmd
		}

		// Search gets all key events when active.
		if m.search.IsActive() {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

		// Global keys (when no overlay).
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Compose):
			m.composer.Compose()
			m.resizeComposer()
			return m, nil

		case key.Matches(msg, keys.Search):
			m.search.Open()
			m.resizeSearch()
			return m, nil

		case key.Matches(msg, keys.Tab):
			if m.reader.IsVisible() {
				// Toggle between list and reader when reader is open.
				if m.activePane == paneList {
					m.setFocus(paneReader)
				} else {
					m.setFocus(paneList)
				}
			} else {
				// Toggle between sidebar and list.
				if m.activePane == paneSidebar {
					m.setFocus(paneList)
				} else {
					m.setFocus(paneSidebar)
				}
			}
			return m, nil

		case key.Matches(msg, keys.Toggle):
			if m.viewMode == viewThread {
				m.viewMode = viewFlat
				m.inbox.SetViewMode(viewFlat)
				m.statusBar.setMessage("Switched to flat view")
			} else {
				m.viewMode = viewThread
				m.inbox.SetViewMode(viewThread)
				m.statusBar.setMessage("Switched to thread view")
			}
			return m, m.loadMailCmd(m.sidebar.activeLabel)

		case key.Matches(msg, keys.SwitchAccount):
			if len(m.accounts) < 2 {
				m.statusBar.setMessage("Only one account configured")
				return m, nil
			}
			return m, m.switchAccountCmd()
		}

		// Delegate to focused sub-model.
		switch m.activePane {
		case paneSidebar:
			var cmd tea.Cmd
			m.sidebar, cmd = m.sidebar.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

		case paneList:
			var cmd tea.Cmd
			m.inbox, cmd = m.inbox.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

		case paneReader:
			var cmd tea.Cmd
			m.reader, cmd = m.reader.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebarWidth, contentWidth := m.layoutWidths()
	contentHeight := m.height - 3 // reserve space for status bar

	// --- Sidebar ---
	sidebarView := sidebarStyle.
		Width(sidebarWidth).
		Height(contentHeight).
		Render(m.sidebar.View())

	// --- Content area ---
	var contentView string

	switch {
	case m.composer.IsVisible():
		// Composer handles its own border/padding.
		contentView = lipgloss.NewStyle().
			Width(contentWidth).
			Height(contentHeight).
			Render(m.composer.View())

	case m.search.IsActive():
		// Search handles its own styling.
		contentView = lipgloss.NewStyle().
			Width(contentWidth).
			Height(contentHeight).
			Render(m.search.View())

	case m.reader.IsVisible():
		// Split view: list (top half) + reader (bottom half).
		listHeight := contentHeight / 2
		readerHeight := contentHeight - listHeight

		listView := listStyle.
			Width(contentWidth).
			Height(listHeight).
			Render(m.inbox.View())

		readerView := readerStyle.
			Width(contentWidth).
			Height(readerHeight).
			Render(m.reader.View())

		contentView = lipgloss.JoinVertical(lipgloss.Left, listView, readerView)

	default:
		// List takes full content area.
		contentView = listStyle.
			Width(contentWidth).
			Height(contentHeight).
			Render(m.inbox.View())
	}

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentView)
	sb := m.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, main, sb)
}

// --- focus management ---

func (m *model) setFocus(p pane) {
	m.activePane = p
	m.sidebar.focused = (p == paneSidebar)
	m.inbox.focused = (p == paneList)
	m.reader.focused = (p == paneReader)
}

// --- layout helpers ---

func (m model) layoutWidths() (sidebarWidth, contentWidth int) {
	sidebarWidth = m.width / 5
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	contentWidth = m.width - sidebarWidth - 2
	return
}

func (m *model) resizeSubModels() {
	sidebarWidth, contentWidth := m.layoutWidths()
	contentHeight := m.height - 3

	// Pass content area dimensions (subtract border + padding from each style).
	// sidebarStyle: Border(2h + 2v) + Padding(2h + 2v) = 4h, 4v
	m.sidebar.SetSize(sidebarWidth-4, contentHeight-4)

	// listStyle: Border(2h + 2v) + Padding(2h + 0v) = 4h, 2v
	if m.reader.IsVisible() {
		listHeight := contentHeight / 2
		readerHeight := contentHeight - listHeight
		m.inbox.SetSize(contentWidth-4, listHeight-2)
		// readerStyle: Border(2h + 2v) + Padding(4h + 2v) = 6h, 4v
		m.reader.SetSize(contentWidth-6, readerHeight-4)
	} else {
		m.inbox.SetSize(contentWidth-4, contentHeight-2)
	}

	m.resizeComposer()
	m.resizeSearch()
}

func (m *model) resizeComposer() {
	_, contentWidth := m.layoutWidths()
	contentHeight := m.height - 3
	m.composer.SetSize(contentWidth, contentHeight)
}

func (m *model) resizeSearch() {
	_, contentWidth := m.layoutWidths()
	contentHeight := m.height - 3
	m.search.SetSize(contentWidth, contentHeight)
}

// --- async commands ---

func (m model) loadLabelsCmd() tea.Cmd {
	return func() tea.Msg {
		labels, err := m.store.ListLabels(context.Background(), m.accountID)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to load labels: %w", err)}
		}
		return labelsLoadedMsg{labels: labels}
	}
}

func (m model) loadMailCmd(labelID string) tea.Cmd {
	opts := store.ListEmailOptions{
		AccountID: m.accountID,
		LabelID:   labelID,
	}

	if m.viewMode == viewThread {
		return func() tea.Msg {
			threads, err := m.store.ListThreads(context.Background(), opts)
			if err != nil {
				return errMsg{err: fmt.Errorf("failed to load threads: %w", err)}
			}
			return threadsLoadedMsg{threads: threads}
		}
	}

	return func() tea.Msg {
		emails, err := m.store.ListEmails(context.Background(), opts)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to load emails: %w", err)}
		}
		return emailsLoadedMsg{emails: emails}
	}
}

func (m model) loadEmailCmd(emailID string) tea.Cmd {
	return func() tea.Msg {
		email, err := m.store.GetEmail(context.Background(), emailID)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to load email: %w", err)}
		}
		return emailLoadedMsg{email: email}
	}
}

func (m model) loadThreadCmd(threadID string) tea.Cmd {
	return func() tea.Msg {
		thread, err := m.store.GetThread(context.Background(), threadID, m.accountID)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to load thread: %w", err)}
		}
		return threadLoadedMsg{thread: thread}
	}
}

func (m model) markReadCmd(emailID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Update local DB first for immediate UI feedback.
		if err := m.store.SetEmailRead(ctx, emailID, true); err != nil {
			return errMsg{err: fmt.Errorf("failed to mark as read locally: %w", err)}
		}

		// Sync to remote provider.
		if err := m.provider.MarkRead(ctx, emailID, true); err != nil {
			return errMsg{err: fmt.Errorf("failed to mark as read remotely: %w", err)}
		}

		return actionDoneMsg{action: "mark_read"}
	}
}

func (m model) markThreadReadCmd(threadID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Update all messages in thread locally.
		if err := m.store.SetThreadRead(ctx, threadID, true); err != nil {
			return errMsg{err: fmt.Errorf("failed to mark thread as read locally: %w", err)}
		}

		// Get message IDs in the thread to sync to remote.
		thread, err := m.store.GetThread(ctx, threadID, m.accountID)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to load thread for read sync: %w", err)}
		}

		for _, msg := range thread.Messages {
			if !msg.IsRead {
				if err := m.provider.MarkRead(ctx, msg.ID, true); err != nil {
					return errMsg{err: fmt.Errorf("failed to mark message %s as read remotely: %w", msg.ID, err)}
				}
			}
		}

		return actionDoneMsg{action: "mark_read"}
	}
}

func (m model) sendEmailCmd(email *domain.Email) tea.Cmd {
	return func() tea.Msg {
		err := m.provider.SendMessage(context.Background(), email)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to send email: %w", err)}
		}
		return emailSentMsg{}
	}
}

func (m model) searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := m.store.SearchEmails(context.Background(), query, m.accountID)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to search: %w", err)}
		}
		return searchResultsMsg{results: results}
	}
}

func (m model) performActionCmd(emailID, action string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error

		switch action {
		case "archive":
			err = m.provider.ModifyLabels(ctx, emailID, nil, []string{domain.LabelInbox})
		case "delete":
			err = m.provider.TrashMessage(ctx, emailID)
		case "star":
			err = m.provider.ModifyLabels(ctx, emailID, []string{domain.LabelStarred}, nil)
		case "unread":
			if localErr := m.store.SetEmailRead(ctx, emailID, false); localErr != nil {
				return errMsg{err: fmt.Errorf("failed to mark unread locally: %w", localErr)}
			}
			err = m.provider.MarkRead(ctx, emailID, false)
		default:
			return errMsg{err: fmt.Errorf("unknown action: %s", action)}
		}

		if err != nil {
			return errMsg{err: fmt.Errorf("failed to %s: %w", action, err)}
		}
		return actionDoneMsg{action: action}
	}
}

func (m model) switchAccountCmd() tea.Cmd {
	// Cycle to the next account.
	current := m.accountID
	var nextID string
	for i, acc := range m.accounts {
		if acc.ID == current {
			next := (i + 1) % len(m.accounts)
			nextID = m.accounts[next].ID
			break
		}
	}
	if nextID == "" || nextID == current {
		return nil
	}
	return func() tea.Msg {
		return accountSwitchedMsg{accountID: nextID}
	}
}

// Run starts the Bubble Tea TUI application.
func Run(s store.Store, p provider.EmailProvider, accountID string, accounts []domain.Account, factory ProviderFactory) error {
	prog := tea.NewProgram(
		NewModel(s, p, accountID, accounts, factory),
		tea.WithAltScreen(),
	)
	_, err := prog.Run()
	return err
}

package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type screen int

const (
	screenAuth screen = iota
	screenList
	screenDetails
	screenEditor
	screenConfirmDelete
	screenFiles
	screenFileUpload
)

type authMode int

const (
	authLogin authMode = iota
	authRegister
)

type editorMode int

const (
	editorCreate editorMode = iota
	editorEdit
)

type tuiModel struct {
	ctx context.Context
	api Client

	screen screen

	authMode   authMode
	authInputs []textinput.Model
	authFocus  int

	items      []Item
	selected   int
	showSecret bool

	files        []FileSyncInfo
	fileSelected int
	fileInput    textinput.Model

	editorMode   editorMode
	editorID     string
	editorTypeIx int
	editorInputs []textinput.Model
	editorFocus  int

	loading bool
	status  string

	width  int
	height int

	version string
	commit  string
	date    string
}

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)
)

func NewModel(ctx context.Context, api Client, version, commit, date string) tuiModel {
	return tuiModel{
		ctx:        ctx,
		api:        api,
		screen:     screenAuth,
		authMode:   authLogin,
		authInputs: newAuthInputs(authLogin),
		status:     "Введите логин и пароль",
		version:    version,
		commit:     commit,
		date:       date,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := m.updateSystemMsg(msg); handled {
		return model, cmd
	}

	if model, cmd, handled := m.updateResultMsg(msg); handled {
		return model, cmd
	}

	switch m.screen {
	case screenAuth:
		return m.updateAuth(msg)
	case screenList:
		return m.updateList(msg)
	case screenDetails:
		return m.updateDetails(msg)
	case screenEditor:
		return m.updateEditor(msg)
	case screenConfirmDelete:
		return m.updateConfirmDelete(msg)
	case screenFiles:
		return m.updateFiles(msg)
	case screenFileUpload:
		return m.updateFileUpload(msg)
	default:
		return m, nil
	}
}

func (m tuiModel) View() tea.View {
	var body string

	switch m.screen {
	case screenAuth:
		body = m.viewAuth()
	case screenList:
		body = m.viewList()
	case screenDetails:
		body = m.viewDetails()
	case screenEditor:
		body = m.viewEditor()
	case screenConfirmDelete:
		body = m.viewConfirmDelete()
	case screenFiles:
		body = m.viewFiles()
	case screenFileUpload:
		body = m.viewFileUpload()
	default:
		body = "unknown screen"
	}

	view := tea.NewView(appStyle.Render(body))
	view.AltScreen = true
	view.WindowTitle = applicationName
	return view
}

func (m tuiModel) updateSystemMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil, true

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit, true
		}
	}

	return m, nil, false
}

func (m tuiModel) updateResultMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case authOKMsg:
		m.loading = false
		m.applySnapshot(msg.snapshot)
		m.selected = 0
		m.fileSelected = 0
		m.screen = screenList
		m.status = "Вход выполнен, данные синхронизированы"
		return m, nil, true

	case syncedMsg:
		m.finishWithSnapshot(msg.snapshot, m.screen, syncStatusText(msg.snapshot))
		return m, nil, true

	case savedMsg:
		m.finishWithSnapshot(msg.snapshot, screenList, "Запись сохранена")
		return m, nil, true

	case deletedMsg:
		m.finishWithSnapshot(msg.snapshot, screenList, "Запись удалена")
		return m, nil, true

	case fileSyncedMsg:
		m.finishWithSnapshot(msg.snapshot, screenFiles, syncStatusText(msg.snapshot))
		return m, nil, true

	case fileUploadedMsg:
		m.finishWithSnapshot(msg.snapshot, screenFiles, "Файл загружен и список обновлён")
		return m, nil, true

	case fileDownloadedMsg:
		m.finishWithSnapshot(msg.snapshot, screenFiles, "Файл скачан, список обновлён")
		return m, nil, true

	case localFileDeletedMsg:
		m.finishWithSnapshot(msg.snapshot, screenFiles, "Локальный файл удалён, список обновлён")
		return m, nil, true

	case errMsg:
		m.loading = false
		m.status = msg.err.Error()
		return m, nil, true
	}

	return m, nil, false
}

func (m *tuiModel) applySnapshot(snapshot SyncSnapshot) {
	m.items = snapshot.Items
	m.files = snapshot.Files
	clampSelection(m)
	clampFileSelection(m)
}

func (m *tuiModel) finishWithSnapshot(snapshot SyncSnapshot, nextScreen screen, status string) {
	m.loading = false
	m.applySnapshot(snapshot)
	m.screen = nextScreen
	m.status = status
}

func (m *tuiModel) setLoading(status string) {
	m.loading = true
	m.status = status
}

func (m tuiModel) statusView() string {
	if m.loading {
		return subtleStyle.Render("⏳ " + m.status)
	}

	status := strings.ToLower(m.status)
	if strings.Contains(status, "ошиб") || strings.Contains(status, "обяз") || strings.Contains(status, "не ") {
		return errorStyle.Render(m.status)
	}

	return okStyle.Render(m.status)
}

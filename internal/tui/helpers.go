package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type textInputOptions struct {
	placeholder string
	value       string
	width       int
	charLimit   int
	focused     bool
	secret      bool
}

type inputValues []textinput.Model

func newTextInput(opts textInputOptions) textinput.Model {
	input := textinput.New()
	input.Placeholder = opts.placeholder
	input.CharLimit = opts.charLimit
	input.SetWidth(opts.width)
	input.SetValue(opts.value)

	if opts.secret {
		input.EchoMode = textinput.EchoPassword
		input.EchoCharacter = '•'
	}
	if opts.focused {
		input.Focus()
	}

	return input
}

func updateTextInputs(inputs []textinput.Model, msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(inputs))

	for i := range inputs {
		inputs[i], cmds[i] = inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func focusTextInput(inputs []textinput.Model, focusedIndex int) tea.Cmd {
	cmds := make([]tea.Cmd, len(inputs))

	for i := range inputs {
		if i == focusedIndex {
			cmds[i] = inputs[i].Focus()
			continue
		}

		inputs[i].Blur()
	}

	return tea.Batch(cmds...)
}

func newInputValues(inputs []textinput.Model) inputValues {
	return inputValues(inputs)
}

func (values inputValues) raw(index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}

	return values[index].Value()
}

func (values inputValues) trimmed(index int) string {
	return strings.TrimSpace(values.raw(index))
}

func (m tuiModel) currentItem() (Item, bool) {
	if len(m.items) == 0 || m.selected < 0 || m.selected >= len(m.items) {
		return Item{}, false
	}

	return m.items[m.selected], true
}

func (m tuiModel) currentFile() (FileSyncInfo, bool) {
	if len(m.files) == 0 || m.fileSelected < 0 || m.fileSelected >= len(m.files) {
		return FileSyncInfo{}, false
	}

	return m.files[m.fileSelected], true
}

func clampSelection(m *tuiModel) {
	m.selected = clampIndex(m.selected, len(m.items))
}

func clampFileSelection(m *tuiModel) {
	m.fileSelected = clampIndex(m.fileSelected, len(m.files))
}

func clampIndex(index, count int) int {
	if count == 0 || index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}

	return index
}

func nextIndex(index, count int) int {
	if count == 0 {
		return 0
	}

	return (index + 1) % count
}

func prevIndex(index, count int) int {
	if count == 0 {
		return 0
	}
	if index <= 0 {
		return count - 1
	}

	return index - 1
}

func minIndex(index, count int) int {
	if count == 0 || index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}

	return index
}

func maxIndex(index int) int {
	if index < 0 {
		return 0
	}

	return index
}

func typeIndex(t ItemType) int {
	for i, v := range itemTypes {
		if v == t {
			return i
		}
	}

	return 0
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}

	return string(runes[:max-1]) + "…"
}

func mask(s string) string {
	if s == "" {
		return ""
	}

	n := len([]rune(s))
	if n > 12 {
		n = 12
	}

	return strings.Repeat("•", n)
}

func fileStatusView(status FileSyncStatus) string {
	switch status {
	case FileSyncStatusSynced:
		return okStyle.Render("✓ synced")
	case FileSyncStatusLocalOnly:
		return tagStyle.Render("↑ local")
	case FileSyncStatusRemoteOnly:
		return subtleStyle.Render("↓ remote")
	default:
		return string(status)
	}
}

func syncStatusText(snapshot SyncSnapshot) string {
	var synced, local, remote int

	for _, file := range snapshot.Files {
		switch file.Status {
		case FileSyncStatusSynced:
			synced++
		case FileSyncStatusLocalOnly:
			local++
		case FileSyncStatusRemoteOnly:
			remote++
		}
	}

	return fmt.Sprintf(
		"Синхронизация завершена: секретов %d, файлов synced=%d, local_only=%d, remote_only=%d",
		len(snapshot.Items),
		synced,
		local,
		remote,
	)
}

func formatFileTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	return t.Format("02.01 15:04")
}

func formatSize(size int64) string {
	if size <= 0 {
		return "-"
	}

	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

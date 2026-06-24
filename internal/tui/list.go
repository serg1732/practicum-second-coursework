package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m tuiModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "j", "down":
			m.selected = minIndex(m.selected+1, len(m.items))
			return m, nil
		case "k", "up":
			m.selected = maxIndex(m.selected - 1)
			return m, nil
		case "enter", "v":
			return m.openDetails()
		case "n":
			m.openCreateEditor()
			return m, nil
		case "e":
			m.openCurrentItemEditor()
			return m, nil
		case "d":
			m.openDeleteConfirmation()
			return m, nil
		case "f":
			m.screen = screenFiles
			m.status = "Файлы и статус синхронизации"
			return m, nil
		case "s", "r":
			m.setLoading("Обновление состояния...")
			return m, syncCmd(m.ctx, m.api)
		case "l":
			return m.logout()
		}
	}

	return m, nil
}

func (m tuiModel) openDetails() (tea.Model, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}

	m.showSecret = false
	m.screen = screenDetails
	return m, nil
}

func (m *tuiModel) openCurrentItemEditor() {
	if item, ok := m.currentItem(); ok {
		m.openEditEditor(item)
	}
}

func (m *tuiModel) openDeleteConfirmation() {
	if len(m.items) > 0 {
		m.screen = screenConfirmDelete
	}
}

func (m tuiModel) logout() (tea.Model, tea.Cmd) {
	m.items = nil
	m.files = nil
	m.selected = 0
	m.fileSelected = 0
	m.screen = screenAuth
	m.status = "Вы вышли из клиента"
	return m, logoutCmd(m.ctx, m.api)
}

func (m tuiModel) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(applicationName))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("Секреты пользователя"))
	b.WriteString("\n\n")

	if len(m.items) == 0 {
		b.WriteString(subtleStyle.Render("Хранилище пустое. Нажмите n, чтобы добавить запись."))
		b.WriteString("\n\n")
	} else {
		m.writeItems(&b)
	}

	b.WriteString(m.statusView())
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("↑/↓ выбор • enter просмотр • n новая • e редактировать • d удалить • f файлы • s refresh • l logout • q выход"))

	return b.String()
}

func (m tuiModel) writeItems(b *strings.Builder) {
	for i, item := range m.items {
		row := fmt.Sprintf(
			"%-10s %-24s %-24s %s",
			item.Type,
			truncate(item.Title, 24),
			truncate(item.Login, 24),
			item.UpdatedAt.Format("02.01 15:04"),
		)

		if i == m.selected {
			row = selectedStyle.Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	b.WriteString("\n")
}

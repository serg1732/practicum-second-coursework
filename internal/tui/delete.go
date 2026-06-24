package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m tuiModel) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y":
			return m.confirmDelete()
		case "n", "esc":
			m.screen = screenList
			return m, nil
		}
	}

	return m, nil
}

func (m tuiModel) confirmDelete() (tea.Model, tea.Cmd) {
	item, ok := m.currentItem()
	if !ok {
		m.screen = screenList
		return m, nil
	}

	m.setLoading("Удаление...")
	return m, deleteCmd(m.ctx, m.api, item.ID)
}

func (m tuiModel) viewConfirmDelete() string {
	item, ok := m.currentItem()
	if !ok {
		return "Запись не найдена\n\nesc назад"
	}

	var b strings.Builder

	b.WriteString(errorStyle.Render("Удалить запись?"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render("Название:"), item.Title))
	b.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render("Тип:"), item.Type))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("y удалить • n отмена"))

	return b.String()
}

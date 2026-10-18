package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (m tuiModel) updateDetails(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.screen = screenList
			return m, nil
		case "q":
			return m, tea.Quit
		case "p":
			m.showSecret = !m.showSecret
			return m, nil
		case "e":
			m.openCurrentItemEditor()
			return m, nil
		case "d":
			m.screen = screenConfirmDelete
			return m, nil
		}
	}

	return m, nil
}

func (m tuiModel) viewDetails() string {
	item, ok := m.currentItem()
	if !ok {
		return "Запись не найдена\n\nesc назад"
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render(item.Title))
	b.WriteString("\n\n")
	writeField(&b, "ID:       ", item.ID)
	writeField(&b, "Тип:      ", string(item.Type))

	m.writeItemDetails(&b, item)
	writeField(&b, "Updated:  ", item.UpdatedAt.Format(time.RFC3339))

	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("p показать/скрыть секрет • e редактировать • d удалить • esc назад"))

	return b.String()
}

func (m tuiModel) writeItemDetails(b *strings.Builder, item Item) {
	switch item.Type {
	case ItemCard:
		m.writeCardDetails(b, item)
	case ItemText:
		m.writeTextDetails(b, item)
	default:
		m.writeCredsDetails(b, item)
	}
}

func (m tuiModel) writeCardDetails(b *strings.Builder, item Item) {
	card := cardDraftFromDraft(Draft{
		Type:   item.Type,
		Title:  item.Title,
		Login:  item.Login,
		Secret: item.Secret,
		Meta:   item.Meta,
	})

	writeField(b, "Описание:", card.Description)
	writeField(b, "Система:  ", card.PaymentSystem)
	writeField(b, "Номер:    ", card.Number)
	writeField(b, "Владелец: ", card.Holder)
	writeField(b, "CVC:      ", m.maskSecret(card.CVC))
	writeField(b, "End date: ", card.EndDate)
}

func (m tuiModel) writeTextDetails(b *strings.Builder, item Item) {
	writeField(b, "Описание:", item.Meta)
	writeField(b, "Текст:    ", m.maskSecret(item.Secret))
}

func (m tuiModel) writeCredsDetails(b *strings.Builder, item Item) {
	writeField(b, "Логин:    ", item.Login)
	writeField(b, "Секрет:   ", m.maskSecret(item.Secret))
	writeField(b, "Meta:     ", item.Meta)
}

func (m tuiModel) maskSecret(secret string) string {
	if m.showSecret {
		return secret
	}

	return mask(secret)
}

func writeField(b *strings.Builder, label, value string) {
	b.WriteString(fmt.Sprintf("%s %s\n", subtleStyle.Render(label), value))
}

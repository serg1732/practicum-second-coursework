package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m tuiModel) updateAuth(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Quit
		case "ctrl+r":
			m.toggleAuthMode()
			return m, nil
		case "tab", "down":
			m.authFocus = nextIndex(m.authFocus, len(m.authInputs))
			return m, m.focusAuthInput()
		case "shift+tab", "up":
			m.authFocus = prevIndex(m.authFocus, len(m.authInputs))
			return m, m.focusAuthInput()
		case "enter":
			return m.submitAuth()
		}
	}

	return m, updateTextInputs(m.authInputs, msg)
}

func (m *tuiModel) toggleAuthMode() {
	if m.authMode == authLogin {
		m.authMode = authRegister
		m.status = "Режим регистрации"
	} else {
		m.authMode = authLogin
		m.status = "Режим входа"
	}

	m.authInputs = newAuthInputs(m.authMode)
	m.authFocus = 0
}

func (m *tuiModel) focusAuthInput() tea.Cmd {
	return focusTextInput(m.authInputs, m.authFocus)
}

func (m tuiModel) submitAuth() (tea.Model, tea.Cmd) {
	username := strings.TrimSpace(m.authInputs[0].Value())
	password := m.authInputs[1].Value()

	if username == "" || password == "" {
		m.status = "Логин и мастер-пароль обязательны"
		return m, nil
	}

	if m.authMode == authRegister {
		return m.submitRegister(username, password)
	}

	m.setLoading("Аутентификация...")
	return m, loginCmd(m.ctx, m.api, username, password)
}

func (m tuiModel) submitRegister(username, password string) (tea.Model, tea.Cmd) {
	repeat := m.authInputs[2].Value()
	if password != repeat {
		m.status = "Мастер-пароли не совпадают"
		return m, nil
	}

	m.setLoading("Регистрация...")
	return m, registerCmd(m.ctx, m.api, username, password)
}

func newAuthInputs(mode authMode) []textinput.Model {
	placeholders := []string{"username", "password"}
	if mode == authRegister {
		placeholders = append(placeholders, "repeat password")
	}

	inputs := make([]textinput.Model, len(placeholders))
	for i, placeholder := range placeholders {
		inputs[i] = newTextInput(textInputOptions{
			placeholder: placeholder,
			width:       40,
			charLimit:   128,
			focused:     i == 0,
			secret:      i > 0,
		})
	}

	return inputs
}

func (m tuiModel) viewAuth() string {
	mode := "Вход"
	hint := "ctrl+r регистрация"
	if m.authMode == authRegister {
		mode = "Регистрация"
		hint = "ctrl+r вход"
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render(applicationName))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("Клиент для безопасного хранилища секретов"))
	b.WriteString("\n\n")
	b.WriteString(tagStyle.Render(mode))
	b.WriteString("\n\n")

	for _, input := range m.authInputs {
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.statusView())
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("enter отправить • tab переключить поле • " + hint + " • esc выйти"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("версия " + m.version + ", коммит " + m.commit + ", дата сборки " + m.date))
	return b.String()
}

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (m tuiModel) updateFiles(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			m.screen = screenList
			return m, nil
		case "q":
			return m, tea.Quit
		case "j", "down":
			m.fileSelected = minIndex(m.fileSelected+1, len(m.files))
			return m, nil
		case "k", "up":
			m.fileSelected = maxIndex(m.fileSelected - 1)
			return m, nil
		case "r":
			m.setLoading("Обновление списка файлов...")
			return m, syncCmd(m.ctx, m.api)
		case "s":
			m.setLoading("Синхронизация файлов...")
			return m, syncFilesCmd(m.ctx, m.api)
		case "u":
			m.openFileUpload()
			return m, nil
		case "enter":
			return m.downloadCurrentFile()
		case "d", "delete":
			return m.deleteCurrentLocalFile()
		case "x":
			return m.removeCurrentRemoteFile()
		}
	}

	return m, nil
}

func (m tuiModel) downloadCurrentFile() (tea.Model, tea.Cmd) {
	file, ok := m.currentFile()
	if !ok {
		return m, nil
	}

	m.setLoading("Скачивание файла...")
	return m, downloadFileCmd(m.ctx, m.api, file.Name)
}

func (m tuiModel) deleteCurrentLocalFile() (tea.Model, tea.Cmd) {
	file, ok := m.currentFile()
	if !ok {
		return m, nil
	}

	if file.Status == FileSyncStatusRemoteOnly {
		m.status = "Локального файла нет: запись есть только на сервере"
		return m, nil
	}

	m.setLoading("Удаление локального файла...")
	return m, deleteLocalFileCmd(m.ctx, m.api, file.Name)
}

func (m tuiModel) removeCurrentRemoteFile() (tea.Model, tea.Cmd) {
	file, ok := m.currentFile()
	if !ok {
		return m, nil
	}

	m.setLoading("Удаление файла на сервере...")
	return m, removeFileCmd(m.ctx, m.api, file.Name)
}

func (m tuiModel) viewFiles() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(applicationName))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("Файлы и статус синхронизации"))
	b.WriteString("\n\n")

	if len(m.files) == 0 {
		b.WriteString(subtleStyle.Render("Файлов пока нет. Нажмите u, чтобы загрузить файл."))
		b.WriteString("\n\n")
	} else {
		m.writeFiles(&b)
	}

	b.WriteString(m.statusView())
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("↑/↓ выбор • s синхронизировать • r обновить • u загрузить • enter скачать • d удалить локальный • x удалить на сервере • esc назад"))

	return b.String()
}

func (m tuiModel) writeFiles(b *strings.Builder) {
	b.WriteString(subtleStyle.Render(fmt.Sprintf("%-14s %-32s %-10s %s", "Статус", "Имя", "Размер", "Дата")))
	b.WriteString("\n")

	for i, file := range m.files {
		row := fmt.Sprintf(
			"%-14s %-32s %-10s %s",
			fileStatusView(file.Status),
			truncate(file.Name, 32),
			formatSize(file.Size),
			formatFileTime(displayFileTime(file)),
		)

		if i == m.fileSelected {
			row = selectedStyle.Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	b.WriteString("\n")
}

func displayFileTime(file FileSyncInfo) time.Time {
	if !file.RemoteCreatedAt.IsZero() {
		return file.RemoteCreatedAt
	}

	return file.LocalUpdatedAt
}

func (m *tuiModel) openFileUpload() {
	m.fileInput = newTextInput(textInputOptions{
		placeholder: "Путь к файлу, например ./secret.bin",
		width:       70,
		charLimit:   4096,
		focused:     true,
	})
	m.screen = screenFileUpload
	m.status = "Выберите локальный файл для загрузки"
}

func (m tuiModel) updateFileUpload(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenFiles
			return m, nil
		case "enter", "ctrl+s":
			return m.submitFileUpload()
		}
	}

	var cmd tea.Cmd
	m.fileInput, cmd = m.fileInput.Update(msg)
	return m, cmd
}

func (m tuiModel) submitFileUpload() (tea.Model, tea.Cmd) {
	path := strings.TrimSpace(m.fileInput.Value())
	if path == "" {
		m.status = "Путь к файлу обязателен"
		return m, nil
	}

	m.setLoading("Загрузка файла...")
	return m, uploadFileCmd(m.ctx, m.api, path)
}

func (m tuiModel) viewFileUpload() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Загрузка файла"))
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("Локальный путь"))
	b.WriteString("\n")
	b.WriteString(m.fileInput.View())
	b.WriteString("\n\n")
	b.WriteString(m.statusView())
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("enter загрузить • esc назад"))

	return b.String()
}

package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type editorInputSpec struct {
	label       string
	placeholder string
	value       string
	echoSecret  bool
}

func (m *tuiModel) openCreateEditor() {
	m.editorMode = editorCreate
	m.editorID = ""
	m.editorTypeIx = typeIndex(ItemText)
	m.editorInputs = newEditorInputsForDraft(ItemText, Draft{Type: ItemText})
	m.editorFocus = 0
	m.screen = screenEditor
	m.status = "Новая запись"
}

func (m *tuiModel) openEditEditor(item Item) {
	m.editorMode = editorEdit
	m.editorID = item.ID
	m.editorTypeIx = typeIndex(item.Type)
	m.editorInputs = newEditorInputsForItem(item)
	m.editorFocus = 0
	m.screen = screenEditor
	m.status = "Редактирование записи"
}

func (m tuiModel) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenList
			return m, nil
		case "tab", "down":
			m.editorFocus = nextIndex(m.editorFocus, len(m.editorInputs))
			return m, m.focusEditorInput()
		case "shift+tab", "up":
			m.editorFocus = prevIndex(m.editorFocus, len(m.editorInputs))
			return m, m.focusEditorInput()
		case "ctrl+t":
			m.nextEditorType()
			return m, m.focusEditorInput()
		case "ctrl+s":
			return m.submitEditor()
		}
	}

	return m, updateTextInputs(m.editorInputs, msg)
}

func (m *tuiModel) focusEditorInput() tea.Cmd {
	return focusTextInput(m.editorInputs, m.editorFocus)
}

func (m *tuiModel) nextEditorType() {
	draft := m.editorDraftFromInputs()
	m.editorTypeIx = nextIndex(m.editorTypeIx, len(itemTypes))

	itemType := itemTypes[m.editorTypeIx]
	m.editorInputs = newEditorInputsForDraft(itemType, draft)
	m.editorFocus = 0
	m.status = fmt.Sprintf("Тип изменён на %s", itemType)
}

func (m tuiModel) submitEditor() (tea.Model, tea.Cmd) {
	draft := m.editorDraftFromInputs()
	if err := validateDraft(draft); err != nil {
		m.status = err.Error()
		return m, nil
	}

	m.setLoading("Сохранение...")
	return m, saveCmd(m.ctx, m.api, m.editorID, draft)
}

func (m tuiModel) editorDraftFromInputs() Draft {
	itemType := itemTypes[m.editorTypeIx]
	values := newInputValues(m.editorInputs)

	switch itemType {
	case ItemText:
		return Draft{
			Type:   itemType,
			Title:  values.trimmed(0),
			Secret: values.raw(1),
			Meta:   values.trimmed(2),
		}
	case ItemCard:
		return cardDraftFromInputs(values, values.trimmed(0))
	case ItemBinary:
		path := values.trimmed(0)
		return Draft{
			Type:   itemType,
			Title:  safeFileName(path),
			Secret: path,
		}
	case ItemCreds:
		return Draft{
			Type:   itemType,
			Title:  values.trimmed(0),
			Login:  values.trimmed(1),
			Secret: values.raw(2),
			Meta:   values.trimmed(3),
		}
	default:
		return Draft{Type: itemType}
	}
}

func cardDraftFromInputs(values inputValues, title string) Draft {
	card := cardDraft{
		Description:   values.trimmed(1),
		PaymentSystem: values.trimmed(2),
		Number:        values.trimmed(3),
		Holder:        values.trimmed(4),
		CVC:           values.trimmed(5),
		EndDate:       values.trimmed(6),
	}

	return Draft{
		Type:   ItemCard,
		Title:  title,
		Login:  card.Number,
		Secret: card.CVC,
		Meta:   cardDraftToJSON(card),
	}
}

func validateDraft(draft Draft) error {
	switch draft.Type {
	case ItemText:
		return validateTextDraft(draft)
	case ItemCard:
		return validateCardItemDraft(draft)
	case ItemBinary:
		return validateBinaryDraft(draft)
	case ItemCreds:
		return validateCredsDraft(draft)
	default:
		return fmt.Errorf("unsupported item type %q", draft.Type)
	}
}

func validateTextDraft(draft Draft) error {
	if draft.Title == "" {
		return errors.New("для text название обязательно")
	}
	if strings.TrimSpace(draft.Secret) == "" {
		return errors.New("для text содержимое обязательно")
	}

	return nil
}

func validateCardItemDraft(draft Draft) error {
	if draft.Title == "" {
		return errors.New("для card название обязательно")
	}

	_, err := parseCardDraft(draft)
	return err
}

func validateBinaryDraft(draft Draft) error {
	if strings.TrimSpace(draft.Secret) == "" {
		return errors.New("для binary путь к файлу обязателен")
	}

	return nil
}

func validateCredsDraft(draft Draft) error {
	if draft.Title == "" {
		return errors.New("название обязательно")
	}
	if draft.Login == "" {
		return errors.New("логин обязателен")
	}
	if strings.TrimSpace(draft.Secret) == "" {
		return errors.New("секретное значение обязательно")
	}

	return nil
}

func newEditorInputsForItem(item Item) []textinput.Model {
	draft := Draft{
		Type:   item.Type,
		Title:  item.Title,
		Login:  item.Login,
		Secret: item.Secret,
		Meta:   item.Meta,
	}

	return newEditorInputsForDraft(item.Type, draft)
}

func newEditorInputsForDraft(itemType ItemType, draft Draft) []textinput.Model {
	specs := editorInputSpecs(itemType, draft)
	inputs := make([]textinput.Model, len(specs))

	for i, spec := range specs {
		inputs[i] = newTextInput(textInputOptions{
			placeholder: spec.placeholder,
			value:       spec.value,
			width:       58,
			charLimit:   4096,
			focused:     i == 0,
			secret:      spec.echoSecret,
		})
	}

	return inputs
}

func editorInputSpecs(itemType ItemType, draft Draft) []editorInputSpec {
	switch itemType {
	case ItemText:
		return textEditorInputSpecs(draft)
	case ItemCard:
		return cardEditorInputSpecs(draft)
	case ItemBinary:
		return binaryEditorInputSpecs(draft)
	case ItemCreds:
		return credsEditorInputSpecs(draft)
	default:
		return []editorInputSpec{{label: "Название", placeholder: "Название", value: draft.Title}}
	}
}

func textEditorInputSpecs(draft Draft) []editorInputSpec {
	return []editorInputSpec{
		{label: "Название", placeholder: "Название, например Заметка", value: draft.Title},
		{label: "Текст", placeholder: "Секретный текст", value: draft.Secret, echoSecret: true},
		{label: "Описание", placeholder: "Meta / комментарий", value: draftDescription(draft)},
	}
}

func cardEditorInputSpecs(draft Draft) []editorInputSpec {
	card := cardDraftFromDraft(draft)
	return []editorInputSpec{
		{label: "Название", placeholder: "Название, например Основная карта", value: draft.Title},
		{label: "Описание", placeholder: "Описание / комментарий", value: card.Description},
		{label: "Платёжная система", placeholder: "payment_system, например VISA", value: card.PaymentSystem},
		{label: "Номер / login", placeholder: "number / login", value: card.Number},
		{label: "Владелец", placeholder: "holder", value: card.Holder},
		{label: "CVC / secret", placeholder: "cvc / secret", value: card.CVC, echoSecret: true},
		{label: "Срок действия", placeholder: "end_date, например 12/29 или 12/2030", value: card.EndDate},
	}
}

func binaryEditorInputSpecs(draft Draft) []editorInputSpec {
	path := draft.Secret
	if path == "" {
		path = draft.Title
	}

	return []editorInputSpec{
		{label: "Путь к файлу", placeholder: "Путь к файлу, например ./secret.bin", value: path},
	}
}

func credsEditorInputSpecs(draft Draft) []editorInputSpec {
	return []editorInputSpec{
		{label: "Название", placeholder: "Название, например GitHub", value: draft.Title},
		{label: "Логин", placeholder: "Логин / username", value: draft.Login},
		{label: "Секрет", placeholder: "Пароль / token / secret", value: draft.Secret, echoSecret: true},
		{label: "Описание", placeholder: "Meta / комментарий", value: draftDescription(draft)},
	}
}

func draftDescription(draft Draft) string {
	if draft.Type == ItemCard {
		return cardDraftFromDraft(draft).Description
	}

	return draft.Meta
}

func (m tuiModel) viewEditor() string {
	mode := "Создание записи"
	if m.editorMode == editorEdit {
		mode = "Редактирование записи"
	}

	itemType := itemTypes[m.editorTypeIx]
	labels := editorInputSpecs(itemType, m.editorDraftFromInputs())

	var b strings.Builder

	b.WriteString(titleStyle.Render(mode))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s %s\n\n", subtleStyle.Render("Тип:"), tagStyle.Render(string(itemType))))

	for i, input := range m.editorInputs {
		b.WriteString(subtleStyle.Render(labels[i].label))
		b.WriteString("\n")
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}

	if itemType == ItemCard {
		b.WriteString(subtleStyle.Render("Для card JSON будет собран автоматически из payment_system, number/login, holder, cvc/secret, end_date."))
		b.WriteString("\n\n")
	}

	b.WriteString(m.statusView())
	b.WriteString("\n\n")
	b.WriteString(subtleStyle.Render("ctrl+s сохранить • ctrl+t сменить тип • tab поле • esc назад"))

	return b.String()
}

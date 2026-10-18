package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	keepermodels "github.com/serg1732/practicum-second-coursework/internal/models"
)

func syncResultToItems(result keepermodels.SyncResult) []Item {
	items := make([]Item, 0, len(result.Text)+len(result.Card)+len(result.Creds))
	items = append(items, syncedTextToItems(result.Text)...)
	items = append(items, syncedCardToItems(result.Card)...)
	items = append(items, syncedCredsToItems(result.Creds)...)

	return items
}

func syncedTextToItems(texts []keepermodels.SyncedText) []Item {
	items := make([]Item, 0, len(texts))

	for _, text := range texts {
		items = append(items, Item{
			ID:        itemID(ItemText, text.Name),
			Type:      ItemText,
			Title:     text.Name,
			Secret:    text.Data,
			Meta:      text.Description,
			UpdatedAt: text.UpdatedAt,
		})
	}

	return items
}

func syncedCredsToItems(creds []keepermodels.SyncedCreds) []Item {
	items := make([]Item, 0, len(creds))

	for _, credential := range creds {
		items = append(items, Item{
			ID:        itemID(ItemCreds, credential.Name),
			Type:      ItemCreds,
			Title:     credential.Name,
			Login:     credential.Login,
			Secret:    credential.Password,
			Meta:      credential.Description,
			UpdatedAt: credential.UpdatedAt,
		})
	}

	return items
}

func syncedCardToItems(cards []keepermodels.SyncedCard) []Item {
	items := make([]Item, 0, len(cards))

	for _, card := range cards {
		cardMeta := cardDraft{
			Description:   card.Description,
			PaymentSystem: card.PaymentSystem,
			Number:        card.Number,
			Holder:        card.Holder,
			CVC:           strconv.Itoa(card.CVC),
			EndDate:       card.EndDate.Format(layoutDate),
		}

		items = append(items, Item{
			ID:        itemID(ItemCard, card.Name),
			Type:      ItemCard,
			Title:     card.Name,
			Login:     card.Number,
			Secret:    strconv.Itoa(card.CVC),
			Meta:      cardDraftToJSON(cardMeta),
			UpdatedAt: card.UpdatedAt,
		})
	}

	return items
}

func remoteFilesFromBinary(files []keepermodels.SyncedBinary) map[string]time.Time {
	remote := make(map[string]time.Time, len(files))

	for _, file := range files {
		name := safeFileName(file.Name)
		if name == "" {
			continue
		}

		remote[name] = file.CreatedAt
	}

	return remote
}

type cardDraft struct {
	Description   string `json:"description"`
	PaymentSystem string `json:"payment_system"`
	Number        string `json:"number"`
	Holder        string `json:"holder"`
	CVC           string `json:"cvc"`
	EndDate       string `json:"end_date"`
}

func parseCardDraft(draft Draft) (cardDraft, error) {
	card, err := parseCardDraftMeta(draft.Meta)
	if err != nil {
		return cardDraft{}, fmt.Errorf("для card поле Meta должно быть JSON: %w", err)
	}

	if card.Description == "" {
		card.Description = draft.Title
	}
	fillEmptyCardRequiredFields(&card, draft)
	if err = validateCardDraft(card); err != nil {
		return cardDraft{}, err
	}

	card.EndDate, err = normalizeCardEndDate(card.EndDate)
	if err != nil {
		return cardDraft{}, err
	}

	return card, nil
}

func cardDraftFromDraft(draft Draft) cardDraft {
	card, err := parseCardDraftMeta(draft.Meta)
	if err != nil {
		card.Description = draft.Meta
	}

	if card.Description == "" && draft.Type != ItemCard {
		card.Description = draft.Meta
	}

	fillEmptyCardRequiredFields(&card, draft)
	return card
}

func fillEmptyCardRequiredFields(card *cardDraft, draft Draft) {
	if card.Number == "" {
		card.Number = draft.Login
	}
	if card.CVC == "" {
		card.CVC = draft.Secret
	}
}

func validateCardDraft(card cardDraft) error {
	if card.PaymentSystem == "" || card.Number == "" || card.Holder == "" || card.CVC == "" || card.EndDate == "" {
		return errors.New("для card нужны payment_system, number/login, holder, cvc/secret, end_date")
	}

	return nil
}

func parseCardDraftMeta(meta string) (cardDraft, error) {
	if strings.TrimSpace(meta) == "" {
		return cardDraft{}, nil
	}

	var card cardDraft
	if err := json.Unmarshal([]byte(meta), &card); err != nil {
		return cardDraft{}, err
	}

	return card, nil
}

func normalizeCardEndDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("для card end_date обязателен")
	}

	dateValue := strings.NewReplacer("/", ".", "-", ".").Replace(value)
	if parsed, err := time.Parse(layoutDate, dateValue); err == nil {
		return parsed.Format(layoutDate), nil
	}

	parts := strings.Split(dateValue, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("для card неверный end_date %q: используйте MM/YY, MM/YYYY или DD.MM.YYYY", value)
	}

	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return "", fmt.Errorf("для card неверный месяц end_date %q", value)
	}

	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("для card неверный год end_date %q", value)
	}
	if year >= 0 && year < 100 {
		year += 2000
	}
	if year < 2000 || year > 2099 {
		return "", fmt.Errorf("для card неверный год end_date %q", value)
	}

	lastDayOfMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	return lastDayOfMonth.Format(layoutDate), nil
}

func cardDraftToJSON(card cardDraft) string {
	data, err := json.Marshal(card)
	if err != nil {
		return "{}"
	}

	return string(data)
}

func itemID(itemType ItemType, name string) string {
	return string(itemType) + ":" + name
}

func splitItemID(id string) (kind string, name string) {
	kind, name, ok := strings.Cut(id, ":")
	if !ok {
		return "", ""
	}

	return kind, name
}

package models

import "time"

// SyncResult - результат синхронизации для TUI.
type SyncResult struct {
	Text   []SyncedText
	Card   []SyncedCard
	Creds  []SyncedCreds
	Binary []SyncedBinary
}

type SyncedText struct {
	Name        string
	Description string
	Data        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SyncedCard struct {
	Name          string
	Description   string
	PaymentSystem string
	Number        string
	Holder        string
	CVC           int
	EndDate       time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SyncedCreds struct {
	Name        string
	Description string
	Login       string
	Password    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SyncedBinary struct {
	Name      string
	CreatedAt time.Time
}

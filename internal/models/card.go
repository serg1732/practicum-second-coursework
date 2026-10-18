package models

import "time"

type Card struct {
	Name          string    `json:"Name"`
	Description   string    `json:"Description"`
	PaymentSystem string    `json:"PaymentSystem"`
	Number        string    `json:"Number"`
	Holder        string    `json:"Holder"`
	EndDate       time.Time `json:"EndDate"`
	CVC           int       `json:"CVC"`
}

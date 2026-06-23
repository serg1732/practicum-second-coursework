package models

import (
	"time"
)

type Token struct {
	AccessToken string
	UserID      int64
	Iat         time.Time
	Exp         time.Time
}

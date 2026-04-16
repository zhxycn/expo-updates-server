package model

import "github.com/uptrace/bun"

type User struct {
	bun.BaseModel `bun:"table:user"`

	ID           string `json:"id" bun:",pk"`
	Username     string `json:"username" bun:",unique,notnull"`
	Email        string `json:"email" bun:",unique,notnull"`
	PasswordHash string `json:"-" bun:",notnull"`
	CreatedAt    string `json:"createdAt"`
}

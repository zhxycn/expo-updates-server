package model

import "github.com/uptrace/bun"

type Project struct {
	bun.BaseModel `bun:"table:project"`

	ID        string `json:"id" bun:",pk"`
	Name      string `json:"name" bun:",notnull"`
	Slug      string `json:"slug" bun:",notnull"`
	CreatedAt string `json:"createdAt"`
}

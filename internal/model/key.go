package model

import "github.com/uptrace/bun"

type Key struct {
	bun.BaseModel `bun:"table:key"`

	ID        string `json:"id" bun:",pk"`
	ProjectID string `json:"projectId" bun:",notnull"`
	CreatedBy string `json:"createdBy" bun:",notnull"`
	Name      string `json:"name" bun:",notnull"`
	KeyHash   string `json:"-" bun:",notnull"`
	KeyPrefix string `json:"keyPrefix" bun:",notnull"`
	CreatedAt string `json:"createdAt"`
}

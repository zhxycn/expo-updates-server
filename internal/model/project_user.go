package model

import "github.com/uptrace/bun"

type Role string

const (
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

type ProjectUser struct {
	bun.BaseModel `bun:"table:project_user"`

	ID        string `json:"id" bun:",pk"`
	ProjectID string `json:"projectId" bun:",notnull"`
	UserID    string `json:"userId" bun:",notnull"`
	Role      Role   `json:"role" bun:",notnull"`
	CreatedAt string `json:"createdAt"`
}

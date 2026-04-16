package database

import (
	"context"
	"time"

	"github.com/google/uuid"

	"expo-updates-server/internal/model"
)

func (d *Database) CreateUser(ctx context.Context, username, email, password string) (*model.User, error) {
	hash, err := d.hash.Hash(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	_, err = d.client.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (d *Database) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	user := new(model.User)

	err := d.client.NewSelect().Model(user).Where("username = ? OR email = ?", login, login).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (d *Database) CheckPassword(user *model.User, password string) bool {
	return d.hash.Check(user.PasswordHash, password)
}

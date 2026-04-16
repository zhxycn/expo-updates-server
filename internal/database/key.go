package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"expo-updates-server/internal/model"

	"github.com/google/uuid"
)

func (d *Database) CreateKey(ctx context.Context, projectId, userId, name string) (*model.Key, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}

	plain := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plain))

	key := &model.Key{
		ID:        uuid.Must(uuid.NewV7()).String(),
		ProjectID: projectId,
		CreatedBy: userId,
		Name:      name,
		KeyHash:   hex.EncodeToString(hash[:]),
		KeyPrefix: plain[:8],
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	_, err := d.client.NewInsert().Model(key).Exec(ctx)
	if err != nil {
		return nil, "", err
	}

	return key, plain, nil
}

func (d *Database) VerifyKey(ctx context.Context, plain string) (*model.Key, error) {
	hash := sha256.Sum256([]byte(plain))
	keyHash := hex.EncodeToString(hash[:])

	key := new(model.Key)

	err := d.client.NewSelect().Model(key).Where("key_hash = ?", keyHash).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (d *Database) ListProjectKeys(ctx context.Context, projectId string) ([]model.Key, error) {
	keys := make([]model.Key, 0)

	err := d.client.NewSelect().Model(&keys).
		Where("project_id = ?", projectId).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (d *Database) UpdateKey(ctx context.Context, keyId, projectId, name string) error {
	_, err := d.client.NewUpdate().Model((*model.Key)(nil)).
		Set("name = ?", name).
		Where("id = ? AND project_id = ?", keyId, projectId).
		Exec(ctx)
	return err
}

func (d *Database) DeleteKey(ctx context.Context, keyId, projectId string) error {
	_, err := d.client.NewDelete().Model((*model.Key)(nil)).
		Where("id = ? AND project_id = ?", keyId, projectId).
		Exec(ctx)
	return err
}

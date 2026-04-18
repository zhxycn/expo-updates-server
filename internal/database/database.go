package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"expo-updates-server/internal/crypto"
)

type Database struct {
	client *bun.DB
	hash   crypto.Password
	models []interface{}
}

func NewDatabase(path string, hash crypto.Password) (*Database, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")

	return &Database{
		client: bun.NewDB(db, sqlitedialect.New()),
		hash:   hash,
	}, nil
}

func (d *Database) ModelRegister(models ...interface{}) {
	d.models = append(d.models, models...)
}

func (d *Database) Migrate(ctx context.Context) error {
	for _, m := range d.models {
		_, err := d.client.NewCreateTable().
			Model(m).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

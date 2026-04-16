package main

import (
	"context"
	"fmt"
	"log"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	_ "modernc.org/sqlite"

	"expo-updates-server/internal/config"
	"expo-updates-server/internal/crypto"
	"expo-updates-server/internal/database"
	"expo-updates-server/internal/handler"
	"expo-updates-server/internal/middleware"
	"expo-updates-server/internal/model"
	"expo-updates-server/internal/service"
	"expo-updates-server/internal/signing"
	"expo-updates-server/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	var store storage.Storage
	switch cfg.StorageType {
	case "s3":
		store, err = storage.NewS3Storage(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3Region, cfg.S3AccessKey, cfg.S3SecretKey)
		if err != nil {
			log.Fatal(err)
		}
	default:
		store = storage.NewLocalStorage(cfg.StorageDir)
	}

	signer, err := signing.NewSigner(cfg.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	hash := crypto.NewPassword(crypto.DefaultArgon2())

	db, err := database.NewDatabase(cfg.DatabasePath, hash)
	if err != nil {
		log.Fatal(err)
	}

	db.ModelRegister(
		(*model.User)(nil),
		(*model.Project)(nil),
		(*model.ProjectUser)(nil),
		(*model.Key)(nil),
	)

	if err := db.Migrate(context.Background()); err != nil {
		log.Fatal(err)
	}

	jwt := &middleware.JWT{Secret: []byte(cfg.JWTSecret)}

	svc := service.NewUpdateService(cfg, store)

	h := handler.NewHandler(cfg, svc, db, signer, jwt)

	e := echo.New()
	e.Use(echomw.RequestLogger())
	e.Use(echomw.Recover())

	h.Register(e)

	log.Fatal(e.Start(fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)))
}

package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"expo-updates-server/internal/config"
	"expo-updates-server/internal/handler"
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
		log.Fatal("S3 storage not supported yet")
	default:
		store = storage.NewLocalStorage(cfg.StorageDir)
	}

	signer, err := signing.NewSigner(cfg.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	svc := service.NewUpdateService(cfg, store)

	h := handler.NewHandler(cfg, svc, signer)

	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	h.Register(e)

	log.Fatal(e.Start(fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)))
}

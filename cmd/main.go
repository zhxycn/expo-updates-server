package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	_ "go.uber.org/automaxprocs"
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
		store = storage.NewLocalStorage(filepath.Join(cfg.DataDir, "updates"))
	}
	store = storage.NewCachedStorage(store)

	signer, err := signing.NewSigner(cfg.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	hash := crypto.NewPassword(crypto.DefaultArgon2())

	db, err := database.NewDatabase(filepath.Join(cfg.DataDir, "ota.db"), hash)
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

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      e,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	log.Printf("Server is running at %s", server.Addr)

	<-ctx.Done()

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		log.Fatal(err)
	}
}

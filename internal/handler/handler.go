package handler

import (
	"github.com/labstack/echo/v5"

	"expo-updates-server/internal/config"
	"expo-updates-server/internal/service"
	"expo-updates-server/internal/signing"
)

type Handler struct {
	cfg    *config.Config
	svc    *service.UpdateService
	signer *signing.Signer
}

func NewHandler(cfg *config.Config, svc *service.UpdateService, signer *signing.Signer) *Handler {
	return &Handler{
		cfg:    cfg,
		svc:    svc,
		signer: signer,
	}
}

func (h *Handler) Register(e *echo.Echo) {
	client := e.Group("/api/:project")
	client.GET("/manifest", h.GetManifest)
	client.GET("/assets", h.GetAssets)
	client.POST("/publish", h.Publish)
}

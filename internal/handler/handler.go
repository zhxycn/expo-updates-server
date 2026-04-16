package handler

import (
	"github.com/labstack/echo/v5"

	"expo-updates-server/internal/config"
	"expo-updates-server/internal/database"
	"expo-updates-server/internal/middleware"
	"expo-updates-server/internal/service"
	"expo-updates-server/internal/signing"
)

type Handler struct {
	cfg    *config.Config
	svc    *service.UpdateService
	db     *database.Database
	signer *signing.Signer
	jwt    *middleware.JWT
}

func NewHandler(cfg *config.Config, svc *service.UpdateService, db *database.Database, signer *signing.Signer, jwt *middleware.JWT) *Handler {
	return &Handler{
		cfg:    cfg,
		svc:    svc,
		db:     db,
		signer: signer,
		jwt:    jwt,
	}
}

func (h *Handler) Register(e *echo.Echo) {
	auth := e.Group("/api/auth")
	auth.POST("/register", h.UserRegister)
	auth.POST("/login", h.UserLogin)

	projects := e.Group("/api/projects", h.jwt.Auth())
	projects.POST("", h.CreateProject)
	projects.GET("", h.ListProjects)
	projects.POST("/:id/users", h.AddProjectUser)
	projects.DELETE("/:id/users/:userId", h.DeleteProjectUser)
	projects.POST("/:id/keys", h.CreateProjectKey)
	projects.GET("/:id/keys", h.ListProjectKeys)
	projects.PATCH("/:id/keys/:keyId", h.UpdateProjectKey)
	projects.DELETE("/:id/keys/:keyId", h.DeleteProjectKey)

	updates := e.Group("/api/updates/:project")
	updates.GET("/manifest", h.GetManifest)
	updates.GET("/assets", h.GetAssets)
	updates.POST("/publish", h.Publish)
}

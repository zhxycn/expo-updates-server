package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"expo-updates-server/internal/model"
)

func (h *Handler) CreateProject(c *echo.Context) error {
	userId := c.Get("UserID").(string)

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Name == "" || req.Slug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing parameters.",
		})
	}

	project, err := h.db.CreateProject(c.Request().Context(), req.Name, req.Slug, userId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, project)
}

func (h *Handler) ListProjects(c *echo.Context) error {
	userId := c.Get("UserID").(string)

	projects, err := h.db.ListProjectsByUser(c.Request().Context(), userId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, projects)
}

func (h *Handler) AddProjectUser(c *echo.Context) error {
	userId := c.Get("UserID").(string)
	projectId := c.Param("id")

	member, err := h.db.GetProjectUser(c.Request().Context(), projectId, userId)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Permission denied.",
		})
	}
	if member.Role != model.RoleOwner {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only owner can manage project members.",
		})
	}

	var req struct {
		UserID string     `json:"userId"`
		Role   model.Role `json:"role"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	added, err := h.db.AddProjectUser(c.Request().Context(), projectId, req.UserID, req.Role)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "User already exists.",
		})
	}

	return c.JSON(http.StatusCreated, added)
}

func (h *Handler) DeleteProjectUser(c *echo.Context) error {
	userId := c.Get("UserID").(string)
	projectId := c.Param("id")
	targetUserId := c.Param("userId")

	if userId == targetUserId {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot remove yourself.",
		})
	}

	member, err := h.db.GetProjectUser(c.Request().Context(), projectId, userId)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Permission denied.",
		})
	}
	if member.Role != model.RoleOwner {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only owner can manage project members.",
		})
	}

	if err := h.db.DeleteProjectUser(c.Request().Context(), projectId, targetUserId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) CreateProjectKey(c *echo.Context) error {
	userId := c.Get("UserID").(string)
	projectId := c.Param("id")

	member, err := h.db.GetProjectUser(c.Request().Context(), projectId, userId)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Permission denied.",
		})
	}
	if member.Role != model.RoleOwner {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only owner can manage project keys.",
		})
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Name == "" {
		req.Name = "key-" + uuid.Must(uuid.NewV7()).String()[:8]
	}

	key, plain, err := h.db.CreateKey(c.Request().Context(), projectId, userId, req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"key":    key,
		"secret": plain,
	})
}

func (h *Handler) ListProjectKeys(c *echo.Context) error {
	userId := c.Get("UserID").(string)
	projectId := c.Param("id")

	member, err := h.db.GetProjectUser(c.Request().Context(), projectId, userId)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Permission denied.",
		})
	}
	if member.Role != model.RoleOwner {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only owner can manage project keys.",
		})
	}

	keys, err := h.db.ListProjectKeys(c.Request().Context(), projectId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, keys)
}

func (h *Handler) UpdateProjectKey(c *echo.Context) error {
	userId := c.Get("UserID").(string)
	projectId := c.Param("id")
	keyId := c.Param("keyId")

	member, err := h.db.GetProjectUser(c.Request().Context(), projectId, userId)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Permission denied.",
		})
	}
	if member.Role != model.RoleOwner {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only owner can manage project keys.",
		})
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing parameters.",
		})
	}

	if err := h.db.UpdateKey(c.Request().Context(), keyId, projectId, req.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) DeleteProjectKey(c *echo.Context) error {
	userId := c.Get("UserID").(string)
	projectId := c.Param("id")
	keyId := c.Param("keyId")

	member, err := h.db.GetProjectUser(c.Request().Context(), projectId, userId)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Permission denied.",
		})
	}
	if member.Role != model.RoleOwner {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only owner can manage project keys.",
		})
	}

	if err := h.db.DeleteKey(c.Request().Context(), keyId, projectId); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) UserRegister(c *echo.Context) error {
	var req RegisterRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing parameters.",
		})
	}

	user, err := h.db.CreateUser(c.Request().Context(), req.Username, req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "User already exists.",
		})
	}

	token, err := h.jwt.Generate(user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"user":  user,
		"token": token,
	})
}

func (h *Handler) UserLogin(c *echo.Context) error {
	var req LoginRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Login == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing parameters.",
		})
	}

	user, err := h.db.GetUserByLogin(c.Request().Context(), req.Login)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid credentials.",
		})
	}

	if !h.db.CheckPassword(user, req.Password) {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid credentials.",
		})
	}

	token, err := h.jwt.Generate(user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user":  user,
		"token": token,
	})
}

package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

func (h *Handler) Publish(c *echo.Context) error {
	project := c.Param("project")

	auth := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Missing API key.",
		})
	}

	key, err := h.db.VerifyKey(c.Request().Context(), strings.TrimPrefix(auth, "Bearer "))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid API key.",
		})
	}

	if key.ProjectID != project {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid API key.",
		})
	}

	runtimeVersion := c.FormValue("runtimeVersion")
	if runtimeVersion == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No runtimeVersion provided.",
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to parse multipart form: " + err.Error(),
		})
	}

	files := make(map[string][]byte)
	for fieldName, fileHeaders := range form.File {
		file, err := fileHeaders[0].Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		files[fieldName] = data
	}

	updateId, err := h.svc.PublishUpdate(c.Request().Context(), project, runtimeVersion, files)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"updateId": updateId,
		"message":  "Update published successfully",
	})
}

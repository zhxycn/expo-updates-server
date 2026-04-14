package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

func (h *Handler) Publish(c *echo.Context) error {
	project := c.Param("project")
	runtimeVersion := c.FormValue("runtimeVersion")

	if runtimeVersion == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No runtimeVersion provided.",
		})
	}

	updateID := strconv.FormatInt(time.Now().Unix(), 10)

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

	err = h.svc.PublishUpdate(c.Request().Context(), project, runtimeVersion, updateID, files)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"updateID": updateID,
		"message":  "Update published successfully",
	})
}

package handler

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v5"
)

func (h *Handler) GetAssets(c *echo.Context) error {
	project := c.Param("project")
	asset := c.QueryParam("asset")
	platform := c.QueryParam("platform")
	runtimeVersion := c.QueryParam("runtimeVersion")

	if asset == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No asset name provided.",
		})
	}

	if platform != "ios" && platform != "android" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": `No platform provided. Expected "ios" or "android".`,
		})
	}

	if runtimeVersion == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No runtimeVersion provided.",
		})
	}

	reader, err := h.svc.GetAssetReader(c.Request().Context(), project, runtimeVersion, asset)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Asset \"%s\" does not exist.", asset),
		})
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	if strings.Contains(asset, "bundles/") {
		contentType = "application/javascript"
	} else if ext := filepath.Ext(asset); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			contentType = t
		}
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	return c.Stream(http.StatusOK, contentType, reader)
}

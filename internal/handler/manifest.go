package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/labstack/echo/v5"

	"expo-updates-server/internal/model"
	"expo-updates-server/internal/signing"
)

func (h *Handler) GetManifest(c *echo.Context) error {
	project := c.Param("project")
	protocolVersion, _ := strconv.Atoi(c.Request().Header.Get("expo-protocol-version"))
	platform := c.Request().Header.Get("expo-platform")
	runtimeVersion := c.Request().Header.Get("expo-runtime-version")
	currentUpdateId := c.Request().Header.Get("expo-current-update-id")
	embeddedUpdateId := c.Request().Header.Get("expo-embedded-update-id")
	expectSignature := c.Request().Header.Get("expo-expect-signature")

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

	result, err := h.svc.ResolveManifest(c.Request().Context(), model.ResolveParams{
		Project:          project,
		RuntimeVersion:   runtimeVersion,
		Platform:         platform,
		ProtocolVersion:  protocolVersion,
		CurrentUpdateID:  currentUpdateId,
		EmbeddedUpdateID: embeddedUpdateId,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	if result.Directive != nil {
		return h.writeDirectiveResponse(c, protocolVersion, result.Directive, expectSignature)
	}

	return h.writeManifestResponse(c, protocolVersion, result.Manifest, expectSignature)
}

func (h *Handler) writeManifestResponse(c *echo.Context, protocolVersion int, manifest *model.Manifest, expectSignature string) error {
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	var signatureHeader string
	if expectSignature != "" {
		if h.signer == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Code signing requested but no key supplied when starting server.",
			})
		}

		sig, err := h.signer.Sign(manifestJSON)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		signatureHeader = signing.FormatSignatureHeader(sig)
	}

	assetRequestHeaders := make(map[string]map[string]string)
	for _, asset := range manifest.Assets {
		assetRequestHeaders[asset.Key] = map[string]string{}
	}
	assetRequestHeaders[manifest.LaunchAsset.Key] = map[string]string{}

	extensionsJSON, err := json.Marshal(model.Extensions{
		AssetRequestHeaders: assetRequestHeaders,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	manifestHeader := make(textproto.MIMEHeader)
	manifestHeader.Set("Content-Disposition", `form-data; name="manifest"`)
	manifestHeader.Set("Content-Type", "application/json; charset=utf-8")
	if signatureHeader != "" {
		manifestHeader.Set("expo-signature", signatureHeader)
	}
	manifestPart, err := writer.CreatePart(manifestHeader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	manifestPart.Write(manifestJSON)

	extHeader := make(textproto.MIMEHeader)
	extHeader.Set("Content-Disposition", `form-data; name="extensions"`)
	extHeader.Set("Content-Type", "application/json")
	extPart, err := writer.CreatePart(extHeader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	extPart.Write(extensionsJSON)

	writer.Close()

	c.Response().Header().Set("expo-protocol-version", strconv.Itoa(protocolVersion))
	c.Response().Header().Set("expo-sfv-version", "0")
	c.Response().Header().Set("cache-control", "public, s-maxage=5, max-age=0")

	return c.Blob(http.StatusOK, "multipart/mixed; boundary="+writer.Boundary(), body.Bytes())
}

func (h *Handler) writeDirectiveResponse(c *echo.Context, protocolVersion int, directive *model.Directive, expectSignature string) error {
	directiveJSON, err := json.Marshal(directive)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	var signatureHeader string
	if expectSignature != "" {
		if h.signer == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Code signing requested but no key supplied when starting server.",
			})
		}

		sig, err := h.signer.Sign(directiveJSON)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		signatureHeader = signing.FormatSignatureHeader(sig)
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	dirHeader := make(textproto.MIMEHeader)
	dirHeader.Set("Content-Disposition", `form-data; name="directive"`)
	dirHeader.Set("Content-Type", "application/json; charset=utf-8")
	if signatureHeader != "" {
		dirHeader.Set("expo-signature", signatureHeader)
	}
	dirPart, err := writer.CreatePart(dirHeader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	dirPart.Write(directiveJSON)

	writer.Close()

	c.Response().Header().Set("expo-protocol-version", strconv.Itoa(protocolVersion))
	c.Response().Header().Set("expo-sfv-version", "0")
	c.Response().Header().Set("cache-control", "public, s-maxage=5, max-age=0")

	return c.Blob(http.StatusOK, "multipart/mixed; boundary="+writer.Boundary(), body.Bytes())
}

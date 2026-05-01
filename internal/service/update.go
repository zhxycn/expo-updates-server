package service

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"

	"expo-updates-server/internal/config"
	"expo-updates-server/internal/model"
	"expo-updates-server/internal/storage"
)

type UpdateService struct {
	storage storage.Storage
	cfg     *config.Config
}

func NewUpdateService(cfg *config.Config, store storage.Storage) *UpdateService {
	return &UpdateService{
		storage: store,
		cfg:     cfg,
	}
}

func (s *UpdateService) PublishUpdate(ctx context.Context, project, runtimeVersion string, files map[string][]byte) (string, error) {
	metadataRaw, ok := files["metadata.json"]
	if !ok {
		return "", errors.New("no metadata.json")
	}

	var metadata model.ExportMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return "", err
	}

	for platform, meta := range metadata.FileMetadata {
		var index model.PlatformIndex

		bundlePath := strings.ReplaceAll(meta.Bundle, "\\", "/")
		bundleData, ok := files[bundlePath]
		if !ok {
			return "", fmt.Errorf("bundle %s not found", bundlePath)
		}

		index.Bundle = model.AssetIndex{
			Path:        bundlePath,
			Hash:        computeHash(bundleData),
			Key:         computeKey(bundleData),
			ContentType: "application/javascript",
		}

		index.Assets = make([]model.AssetIndex, 0, len(meta.Assets))

		for _, am := range meta.Assets {
			assetPath := strings.ReplaceAll(am.Path, "\\", "/")
			assetData, ok := files[assetPath]
			if !ok {
				return "", fmt.Errorf("asset %s not found", assetPath)
			}

			index.Assets = append(index.Assets, model.AssetIndex{
				Path:          assetPath,
				Hash:          computeHash(assetData),
				Key:           computeKey(assetData),
				ContentType:   mime.TypeByExtension("." + am.Ext),
				FileExtension: "." + am.Ext,
			})
		}

		indexJSON, err := json.Marshal(index)
		if err != nil {
			return "", err
		}

		files[fmt.Sprintf("index.%s.json", platform)] = indexJSON
	}

	return s.storage.PutUpdate(ctx, project, runtimeVersion, files)
}

func (s *UpdateService) readAsset(ctx context.Context, project, runtimeVersion, updateId, path string) ([]byte, error) {
	reader, err := s.storage.GetAsset(ctx, project, runtimeVersion, updateId, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func computeKey(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])
}

func hashToUUID(hash string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hash[:8], hash[8:12], hash[12:16], hash[16:20], hash[20:32])
}

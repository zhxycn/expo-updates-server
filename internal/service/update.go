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
	"strconv"
	"strings"
	"time"

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

func (s *UpdateService) GetLatestUpdate(ctx context.Context, project, runtimeVersion, platform string) (*model.Manifest, error) {
	updateID, err := s.storage.GetLatestUpdateID(ctx, project, runtimeVersion)
	if err != nil {
		return nil, err
	}

	indexData, err := s.readAsset(ctx, project, runtimeVersion, updateID, fmt.Sprintf("index.%s.json", platform))
	if err != nil {
		return nil, err
	}

	var index model.PlatformIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, err
	}

	metadata, err := s.storage.GetMetadata(ctx, project, runtimeVersion, updateID)
	if err != nil {
		return nil, err
	}

	expoConfig, err := s.storage.GetExpoConfig(ctx, project, runtimeVersion, updateID)
	if err != nil {
		return nil, err
	}

	assets := make([]model.Asset, 0, len(index.Assets))
	for _, ai := range index.Assets {
		assets = append(assets, model.Asset{
			Hash:          ai.Hash,
			Key:           ai.Key,
			ContentType:   ai.ContentType,
			FileExtension: ai.FileExtension,
			URL: fmt.Sprintf("%s/api/updates/%s/assets?asset=%s&runtimeVersion=%s&platform=%s",
				s.cfg.Hostname, project, ai.Path, runtimeVersion, platform),
		})
	}

	launchAsset := model.Asset{
		Hash:        index.Bundle.Hash,
		Key:         index.Bundle.Key,
		ContentType: "application/javascript",
		URL: fmt.Sprintf("%s/api/updates/%s/assets?asset=%s&runtimeVersion=%s&platform=%s",
			s.cfg.Hostname, project, index.Bundle.Path, runtimeVersion, platform),
	}

	metadataHash := sha256.Sum256(metadata)
	manifestID := hashToUUID(hex.EncodeToString(metadataHash[:]))

	var expoConfigJSON map[string]any
	_ = json.Unmarshal(expoConfig, &expoConfigJSON)

	createdAt, _ := strconv.ParseInt(updateID, 10, 64)

	return &model.Manifest{
		ID:             manifestID,
		CreatedAt:      time.Unix(createdAt, 0),
		RuntimeVersion: runtimeVersion,
		LaunchAsset:    launchAsset,
		Assets:         assets,
		Metadata:       map[string]string{},
		Extra: map[string]any{
			"expoClient": expoConfigJSON,
		},
	}, nil
}

func (s *UpdateService) CheckRollback(ctx context.Context, project, runtimeVersion string) (bool, *model.Directive, error) {
	updateID, err := s.storage.GetLatestUpdateID(ctx, project, runtimeVersion)
	if err != nil {
		return false, nil, err
	}

	isRollback, err := s.storage.IsRollback(ctx, project, runtimeVersion, updateID)
	if err != nil {
		return false, nil, err
	}

	if !isRollback {
		return false, nil, nil
	}

	createdAt, _ := strconv.ParseInt(updateID, 10, 64)

	return true, &model.Directive{
		Type: "rollBackToEmbedded",
		Parameters: map[string]any{
			"commitTime": time.Unix(createdAt, 0).UTC().Format(time.RFC3339),
		},
	}, nil
}

func (s *UpdateService) IsUpToDate(ctx context.Context, project, runtimeVersion, currentUpdateID string) (bool, error) {
	updateID, err := s.storage.GetLatestUpdateID(ctx, project, runtimeVersion)
	if err != nil {
		return false, err
	}

	metadata, err := s.storage.GetMetadata(ctx, project, runtimeVersion, updateID)
	if err != nil {
		return false, err
	}

	metadataHash := sha256.Sum256(metadata)
	manifestID := hashToUUID(hex.EncodeToString(metadataHash[:]))

	if currentUpdateID == manifestID {
		return true, nil
	}
	return false, nil
}

func (s *UpdateService) GetAssetReader(ctx context.Context, project, runtimeVersion, assetPath string) (io.ReadCloser, error) {
	updateID, err := s.storage.GetLatestUpdateID(ctx, project, runtimeVersion)
	if err != nil {
		return nil, err
	}

	return s.storage.GetAsset(ctx, project, runtimeVersion, updateID, assetPath)
}

func (s *UpdateService) PublishUpdate(ctx context.Context, project, runtimeVersion, updateID string, files map[string][]byte) error {
	metadataRaw, ok := files["metadata.json"]
	if !ok {
		return errors.New("no metadata.json")
	}

	var metadata model.ExportMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return err
	}

	for platform, meta := range metadata.FileMetadata {
		var index model.PlatformIndex

		bundlePath := strings.ReplaceAll(meta.Bundle, "\\", "/")
		bundleData, ok := files[bundlePath]
		if !ok {
			return fmt.Errorf("bundle %s not found", bundlePath)
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
				return fmt.Errorf("asset %s not found", assetPath)
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
			return err
		}

		files[fmt.Sprintf("index.%s.json", platform)] = indexJSON
	}

	return s.storage.PutUpdate(ctx, project, runtimeVersion, updateID, files)
}

func (s *UpdateService) readAsset(ctx context.Context, project, runtimeVersion, updateID, path string) ([]byte, error) {
	reader, err := s.storage.GetAsset(ctx, project, runtimeVersion, updateID, path)
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

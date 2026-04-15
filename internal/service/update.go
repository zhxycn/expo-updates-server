package service

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strconv"
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

	metadata, err := s.storage.GetMetadata(ctx, project, runtimeVersion, updateID)
	if err != nil {
		return nil, err
	}

	expoConfig, err := s.storage.GetExpoConfig(ctx, project, runtimeVersion, updateID)
	if err != nil {
		return nil, err
	}

	var metadataJSON model.ExportMetadata
	if err = json.Unmarshal(metadata, &metadataJSON); err != nil {
		return nil, err
	}

	platformMetadata, ok := metadataJSON.FileMetadata[platform]
	if !ok {
		return nil, fmt.Errorf("no metadata for platform %s", platform)
	}

	assets := make([]model.Asset, 0, len(platformMetadata.Assets))
	for _, am := range platformMetadata.Assets {
		data, err := s.readAsset(ctx, project, runtimeVersion, updateID, filepath.ToSlash(am.Path))
		if err != nil {
			return nil, err
		}

		assets = append(assets, model.Asset{
			Hash:          computeHash(data),
			Key:           computeKey(data),
			ContentType:   mime.TypeByExtension("." + am.Ext),
			FileExtension: "." + am.Ext,
			URL: fmt.Sprintf("%s/api/%s/assets?asset=%s&runtimeVersion=%s&platform=%s",
				s.cfg.Hostname, project, filepath.ToSlash(am.Path), runtimeVersion, platform),
		})
	}

	bundleData, err := s.readAsset(ctx, project, runtimeVersion, updateID, platformMetadata.Bundle)
	if err != nil {
		return nil, err
	}

	launchAsset := model.Asset{
		Hash:        computeHash(bundleData),
		Key:         computeKey(bundleData),
		ContentType: "application/javascript",
		URL: fmt.Sprintf("%s/api/%s/assets?asset=%s&runtimeVersion=%s&platform=%s",
			s.cfg.Hostname, project, filepath.ToSlash(platformMetadata.Bundle), runtimeVersion, platform),
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

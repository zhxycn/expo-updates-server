package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"expo-updates-server/internal/model"
)

func (s *UpdateService) ResolveManifest(ctx context.Context, p model.ResolveParams) (*model.ManifestResult, error) {
	updateId, err := s.storage.GetLatestUpdateID(ctx, p.Project, p.RuntimeVersion)
	if err != nil {
		return nil, err
	}

	isRollback, err := s.storage.IsRollback(ctx, p.Project, p.RuntimeVersion, updateId)
	if err != nil {
		return nil, err
	}

	if isRollback {
		if p.ProtocolVersion == 0 {
			return nil, fmt.Errorf("rollbacks not supported on protocol version 0")
		}

		if p.CurrentUpdateID == p.EmbeddedUpdateID {
			return &model.ManifestResult{
				Directive: &model.Directive{Type: "noUpdateAvailable"},
			}, nil
		}

		createdAt, _ := strconv.ParseInt(updateId, 10, 64)
		return &model.ManifestResult{
			Directive: &model.Directive{
				Type: "rollBackToEmbedded",
				Parameters: map[string]any{
					"commitTime": time.Unix(createdAt, 0).UTC().Format(time.RFC3339),
				},
			},
		}, nil
	}

	metadata, err := s.storage.GetMetadata(ctx, p.Project, p.RuntimeVersion, updateId)
	if err != nil {
		return nil, err
	}

	metadataHash := sha256.Sum256(metadata)
	manifestId := hashToUUID(hex.EncodeToString(metadataHash[:]))

	if p.CurrentUpdateID != "" && p.ProtocolVersion == 1 && p.CurrentUpdateID == manifestId {
		return &model.ManifestResult{
			Directive: &model.Directive{Type: "noUpdateAvailable"},
		}, nil
	}

	indexData, err := s.readAsset(ctx, p.Project, p.RuntimeVersion, updateId, fmt.Sprintf("index.%s.json", p.Platform))
	if err != nil {
		return nil, err
	}

	var index model.PlatformIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, err
	}

	expoConfig, err := s.storage.GetExpoConfig(ctx, p.Project, p.RuntimeVersion, updateId)
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
				s.cfg.Hostname, p.Project, ai.Path, p.RuntimeVersion, p.Platform),
		})
	}

	launchAsset := model.Asset{
		Hash:        index.Bundle.Hash,
		Key:         index.Bundle.Key,
		ContentType: "application/javascript",
		URL: fmt.Sprintf("%s/api/updates/%s/assets?asset=%s&runtimeVersion=%s&platform=%s",
			s.cfg.Hostname, p.Project, index.Bundle.Path, p.RuntimeVersion, p.Platform),
	}

	var expoConfigJSON map[string]any
	_ = json.Unmarshal(expoConfig, &expoConfigJSON)

	createdAt, _ := strconv.ParseInt(updateId, 10, 64)

	return &model.ManifestResult{
		Manifest: &model.Manifest{
			ID:             manifestId,
			CreatedAt:      time.Unix(createdAt, 0),
			RuntimeVersion: p.RuntimeVersion,
			LaunchAsset:    launchAsset,
			Assets:         assets,
			Metadata:       map[string]string{},
			Extra: map[string]any{
				"expoClient": expoConfigJSON,
			},
		},
	}, nil
}

func (s *UpdateService) GetAssetReader(ctx context.Context, project, runtimeVersion, assetPath string) (io.ReadCloser, error) {
	updateId, err := s.storage.GetLatestUpdateID(ctx, project, runtimeVersion)
	if err != nil {
		return nil, err
	}

	return s.storage.GetAsset(ctx, project, runtimeVersion, updateId, assetPath)
}

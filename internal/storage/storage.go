package storage

import (
	"context"
	"io"
)

type Storage interface {
	GetLatestUpdateID(ctx context.Context, project, runtimeVersion string) (string, error)
	GetMetadata(ctx context.Context, project, runtimeVersion, updateID string) ([]byte, error)
	GetExpoConfig(ctx context.Context, project, runtimeVersion, updateID string) ([]byte, error)
	GetAsset(ctx context.Context, project, runtimeVersion, updateID, assetPath string) (io.ReadCloser, error)
	IsRollback(ctx context.Context, project, runtimeVersion, updateID string) (bool, error)
	PutUpdate(ctx context.Context, project, runtimeVersion, updateID string, files map[string][]byte) error
}

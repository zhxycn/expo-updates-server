package storage

import (
	"context"
	"io"
)

type Storage interface {
	GetLatestUpdateID(ctx context.Context, project, runtimeVersion string) (string, error)
	GetMetadata(ctx context.Context, project, runtimeVersion, updateId string) ([]byte, error)
	GetExpoConfig(ctx context.Context, project, runtimeVersion, updateId string) ([]byte, error)
	GetAsset(ctx context.Context, project, runtimeVersion, updateId, assetPath string) (io.ReadCloser, error)
	IsRollback(ctx context.Context, project, runtimeVersion, updateId string) (bool, error)
	PutUpdate(ctx context.Context, project, runtimeVersion, updateId string, files map[string][]byte) error
}

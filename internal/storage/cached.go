package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"expo-updates-server/internal/cache"
)

type CachedStorage struct {
	inner Storage

	latestID *cache.Cache[string]
	metadata *cache.Cache[[]byte]
	expoConf *cache.Cache[[]byte]
	rollback *cache.Cache[bool]
}

func NewCachedStorage(inner Storage) *CachedStorage {
	return &CachedStorage{
		inner:    inner,
		latestID: cache.New[string](5 * time.Second),
		metadata: cache.New[[]byte](60 * time.Second),
		expoConf: cache.New[[]byte](60 * time.Second),
		rollback: cache.New[bool](10 * time.Second),
	}
}

func (c *CachedStorage) GetLatestUpdateID(ctx context.Context, project, runtimeVersion string) (string, error) {
	key := fmt.Sprintf("%s/%s", project, runtimeVersion)
	if v, ok := c.latestID.Get(key); ok {
		return v, nil
	}

	id, err := c.inner.GetLatestUpdateID(ctx, project, runtimeVersion)
	if err != nil {
		return "", err
	}

	c.latestID.Set(key, id)

	return id, nil
}

func (c *CachedStorage) GetMetadata(ctx context.Context, project, runtimeVersion, updateId string) ([]byte, error) {
	key := fmt.Sprintf("%s/%s/%s", project, runtimeVersion, updateId)
	if v, ok := c.metadata.Get(key); ok {
		return v, nil
	}

	data, err := c.inner.GetMetadata(ctx, project, runtimeVersion, updateId)
	if err != nil {
		return nil, err
	}

	c.metadata.Set(key, data)

	return data, nil
}

func (c *CachedStorage) GetExpoConfig(ctx context.Context, project, runtimeVersion, updateId string) ([]byte, error) {
	key := fmt.Sprintf("%s/%s/%s", project, runtimeVersion, updateId)
	if v, ok := c.expoConf.Get(key); ok {
		return v, nil
	}

	data, err := c.inner.GetExpoConfig(ctx, project, runtimeVersion, updateId)
	if err != nil {
		return nil, err
	}

	c.expoConf.Set(key, data)

	return data, nil
}

func (c *CachedStorage) IsRollback(ctx context.Context, project, runtimeVersion, updateId string) (bool, error) {
	key := fmt.Sprintf("%s/%s/%s", project, runtimeVersion, updateId)
	if v, ok := c.rollback.Get(key); ok {
		return v, nil
	}

	isRollback, err := c.inner.IsRollback(ctx, project, runtimeVersion, updateId)
	if err != nil {
		return false, err
	}

	c.rollback.Set(key, isRollback)

	return isRollback, nil
}

func (c *CachedStorage) GetAsset(ctx context.Context, project, runtimeVersion, updateId, assetPath string) (io.ReadCloser, error) {
	return c.inner.GetAsset(ctx, project, runtimeVersion, updateId, assetPath)
}

func (c *CachedStorage) PutUpdate(ctx context.Context, project, runtimeVersion, updateId string, files map[string][]byte) error {
	err := c.inner.PutUpdate(ctx, project, runtimeVersion, updateId, files)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("%s/%s", project, runtimeVersion)
	c.latestID.DeleteByPrefix(prefix)
	c.metadata.DeleteByPrefix(prefix)
	c.expoConf.DeleteByPrefix(prefix)
	c.rollback.DeleteByPrefix(prefix)

	return nil
}

package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{
		basePath: basePath,
	}
}

func (s *LocalStorage) GetLatestUpdateID(_ context.Context, project, runtimeVersion string) (string, error) {
	dir := filepath.Join(s.basePath, project, runtimeVersion)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	if len(dirs) == 0 {
		return "", fmt.Errorf("no updates for runtime %s", runtimeVersion)
	}

	sort.Slice(dirs, func(i, j int) bool {
		ni, _ := strconv.Atoi(dirs[i])
		nj, _ := strconv.Atoi(dirs[j])
		return ni > nj
	})

	return dirs[0], nil
}

func (s *LocalStorage) GetMetadata(_ context.Context, project, runtimeVersion, updateId string) ([]byte, error) {
	file := filepath.Join(s.updatePath(project, runtimeVersion, updateId), "metadata.json")

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *LocalStorage) GetExpoConfig(_ context.Context, project, runtimeVersion, updateId string) ([]byte, error) {
	file := filepath.Join(s.updatePath(project, runtimeVersion, updateId), "expoConfig.json")

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *LocalStorage) GetAsset(_ context.Context, project, runtimeVersion, updateId, assetPath string) (io.ReadCloser, error) {
	path := filepath.Join(s.updatePath(project, runtimeVersion, updateId), assetPath)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (s *LocalStorage) IsRollback(_ context.Context, project, runtimeVersion, updateId string) (bool, error) {
	path := filepath.Join(s.updatePath(project, runtimeVersion, updateId), "rollback")

	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *LocalStorage) PutUpdate(_ context.Context, project, runtimeVersion, updateId string, files map[string][]byte) error {
	for name, data := range files {
		path := filepath.Join(s.updatePath(project, runtimeVersion, updateId), name)

		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(path, data, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *LocalStorage) updatePath(project, runtimeVersion, updateId string) string {
	return filepath.Join(s.basePath, project, runtimeVersion, updateId)
}

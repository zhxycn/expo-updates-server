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
	"time"
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
	dir, err := safeJoin(s.basePath, project, runtimeVersion)
	if err != nil {
		return "", err
	}

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
	dir, err := s.updatePath(project, runtimeVersion, updateId)
	if err != nil {
		return nil, err
	}

	file, err := safeJoin(dir, "metadata.json")
	if err != nil {
		return nil, err
	}

	return os.ReadFile(file)
}

func (s *LocalStorage) GetExpoConfig(_ context.Context, project, runtimeVersion, updateId string) ([]byte, error) {
	dir, err := s.updatePath(project, runtimeVersion, updateId)
	if err != nil {
		return nil, err
	}

	file, err := safeJoin(dir, "expoConfig.json")
	if err != nil {
		return nil, err
	}

	return os.ReadFile(file)
}

func (s *LocalStorage) GetAsset(_ context.Context, project, runtimeVersion, updateId, assetPath string) (io.ReadCloser, error) {
	dir, err := s.updatePath(project, runtimeVersion, updateId)
	if err != nil {
		return nil, err
	}

	path, err := safeJoin(dir, assetPath)
	if err != nil {
		return nil, err
	}

	return os.Open(path)
}

func (s *LocalStorage) IsRollback(_ context.Context, project, runtimeVersion, updateId string) (bool, error) {
	dir, err := s.updatePath(project, runtimeVersion, updateId)
	if err != nil {
		return false, err
	}

	path, err := safeJoin(dir, "rollback")
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (s *LocalStorage) PutUpdate(_ context.Context, project, runtimeVersion string, files map[string][]byte) (string, error) {
	parent, err := safeJoin(s.basePath, project, runtimeVersion)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", err
	}

	var dir, updateId string
	for {
		updateId = strconv.FormatInt(time.Now().Unix(), 10)

		dir, err = safeJoin(parent, updateId)
		if err != nil {
			return "", err
		}

		err = os.Mkdir(dir, 0755)
		if err == nil {
			break
		}

		if !errors.Is(err, os.ErrExist) {
			return "", err
		}

		time.Sleep(time.Second)
	}

	for name, data := range files {
		path, err := safeJoin(dir, name)
		if err != nil {
			return "", err
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", err
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return "", err
		}
	}

	return updateId, nil
}

func (s *LocalStorage) updatePath(project, runtimeVersion, updateId string) (string, error) {
	return safeJoin(s.basePath, project, runtimeVersion, updateId)
}

package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	client *s3.Client
	bucket string
}

func NewS3Storage(endpoint, bucket, region, accessKey, secretKey string) (*S3Storage, error) {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       region,
		Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		UsePathStyle: true,
	})

	return &S3Storage{client, bucket}, nil
}

func (s *S3Storage) GetLatestUpdateID(ctx context.Context, project, runtimeVersion string) (string, error) {
	prefix := fmt.Sprintf("%s/%s/", project, runtimeVersion)

	prefixes, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return "", err
	}

	var dirs []string
	for _, cp := range prefixes.CommonPrefixes {
		full := strings.TrimPrefix(*cp.Prefix, prefix)
		full = strings.TrimSuffix(full, "/")
		dirs = append(dirs, full)
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

func (s *S3Storage) GetMetadata(ctx context.Context, project, runtimeVersion, updateID string) ([]byte, error) {
	key := fmt.Sprintf("%s/%s/%s/metadata.json", project, runtimeVersion, updateID)

	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()

	data, err := io.ReadAll(object.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *S3Storage) GetExpoConfig(ctx context.Context, project, runtimeVersion, updateID string) ([]byte, error) {
	key := fmt.Sprintf("%s/%s/%s/expoConfig.json", project, runtimeVersion, updateID)

	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()

	data, err := io.ReadAll(object.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *S3Storage) GetAsset(ctx context.Context, project, runtimeVersion, updateID, assetPath string) (io.ReadCloser, error) {
	key := fmt.Sprintf("%s/%s/%s/%s", project, runtimeVersion, updateID, assetPath)

	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return object.Body, nil
}

func (s *S3Storage) IsRollback(ctx context.Context, project, runtimeVersion, updateID string) (bool, error) {
	key := fmt.Sprintf("%s/%s/%s/rollback", project, runtimeVersion, updateID)

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if _, ok := errors.AsType[*types.NotFound](err); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *S3Storage) PutUpdate(ctx context.Context, project, runtimeVersion, updateID string, files map[string][]byte) error {
	for name, data := range files {
		key := fmt.Sprintf("%s/%s/%s/%s", project, runtimeVersion, updateID, name)

		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(data),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

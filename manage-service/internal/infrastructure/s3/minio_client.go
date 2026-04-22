package s3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioProvider struct {
	client *minio.Client
	bucket string
}

func New(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioProvider, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinioProvider{
		client: minioClient,
		bucket: bucket,
	}, nil
}

func (p *MinioProvider) UploadFile(ctx context.Context, key string, content []byte) error {
	// Upload the zip file with FPutObject
	reader := bytes.NewReader(content)
	objectSize := int64(len(content))

	fmt.Printf("[S3] Uploading file to bucket %s with key %s (size: %d)\n", p.bucket, key, objectSize)

	_, err := p.client.PutObject(ctx, p.bucket, key, reader, objectSize, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
		// You can detect content type if needed, but octet-stream is safe
	})
	if err != nil {
		return fmt.Errorf("failed to upload to minio: %w", err)
	}

	fmt.Printf("[S3] ✅ Upload success: %s\n", key)
	return nil
}

func (p *MinioProvider) GetFile(ctx context.Context, key string) ([]byte, error) {
	fmt.Printf("[S3] Fetching file %s from bucket %s\n", key, p.bucket)

	obj, err := p.client.GetObject(ctx, p.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("failed to read object content: %w", err)
	}

	return buf.Bytes(), nil
}

func (p *MinioProvider) DeleteFile(ctx context.Context, key string) error {
	fmt.Printf("[S3] Deleting file %s from bucket %s\n", key, p.bucket)

	err := p.client.RemoveObject(ctx, p.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object from minio: %w", err)
	}

	fmt.Printf("[S3] ✅ Delete success: %s\n", key)
	return nil
}

func (p *MinioProvider) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	objectCh := p.client.ListObjects(ctx, p.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		files = append(files, object.Key)
	}

	return files, nil
}

func (p *MinioProvider) CopyFile(ctx context.Context, src, dst string) error {
	fmt.Printf("[S3] Copying file from %s to %s in bucket %s\n", src, dst, p.bucket)

	srcOpts := minio.CopySrcOptions{
		Bucket: p.bucket,
		Object: src,
	}

	dstOpts := minio.CopyDestOptions{
		Bucket: p.bucket,
		Object: dst,
	}

	_, err := p.client.CopyObject(ctx, dstOpts, srcOpts)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	fmt.Printf("[S3] ✅ Copy success: %s -> %s\n", src, dst)
	return nil
}

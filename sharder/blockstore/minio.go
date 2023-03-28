package blockstore

import (
	"github.com/minio/minio-go"
)

type (
	MinioClient interface {
		// FPutObject creates an object in a bucket, with contents from file at filePath
		FPutObject(bucketName string, hash string, filePath string, options minio.PutObjectOptions) (int64, error)

		// FGetObject downloads contents of an object to a local file.
		FGetObject(bucketName string, objectName string, filePath string, options minio.GetObjectOptions) error

		// StatObject verifies if object exists and you have permission to access.
		StatObject(bucketName string, hash string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)

		// BucketName returns bucket name.
		BucketName() string

		// DeleteLocal returns true if local files need to remove in uploading stage.
		DeleteLocal() bool

		// MakeBucket creates a new bucket with bucketName.
		MakeBucket(bucketName string, location string) error

		// BucketExists verify if bucket exists and you have permission to access it.
		BucketExists(bucketName string) (bool, error)
	}

	minioClient struct {
		*minio.Client
		bucketName  string
		deleteLocal bool
	}

	MinioConfiguration struct {
		StorageServiceURL string
		AccessKeyID       string
		SecretAccessKey   string
		BucketName        string
		BucketLocation    string
		DeleteLocal       bool
		Secure            bool // Secure defines using ssl
	}
)

var (
	// Make sure minioClient implements MinioClient.
	_ MinioClient = (*minioClient)(nil)
)

// CreateMinioClientFromConfig creates MinioClient from passed config.
func CreateMinioClientFromConfig(config MinioConfiguration) (MinioClient, error) {
	mc, err := minio.New(
		config.StorageServiceURL,
		config.AccessKeyID,
		config.SecretAccessKey,
		config.Secure,
	)
	if err != nil {
		return nil, err
	}

	return &minioClient{
		Client:      mc,
		bucketName:  config.BucketName,
		deleteLocal: config.DeleteLocal,
	}, nil
}

// BucketName is a part of MinioClient interface implementation.
func (mc *minioClient) BucketName() string {
	return mc.bucketName
}

// DeleteLocal is a part of MinioClient interface implementation.
func (mc *minioClient) DeleteLocal() bool {
	return mc.deleteLocal
}

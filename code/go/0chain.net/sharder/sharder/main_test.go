package main

import (
	"bytes"
	"testing"

	"0chain.net/core/common"
	"0chain.net/sharder/blockstore"
	"github.com/stretchr/testify/require"
)

func TestMinoConfig(t *testing.T) {

	testCases := []struct {
		name          string
		minioFile     []byte
		err           error
		configuration blockstore.MinioConfiguration
	}{
		{
			name: "Minio config should pass when everything is provided",
			minioFile: []byte(`
storage_service_url: play.min.io
access_key_id: Q3AM3UQ867SPQQA43P2F
secret_access_key: zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG
bucket_name: mytestbucket
bucket_location: us-east-1
`),
			configuration: blockstore.MinioConfiguration{
				StorageServiceURL: "play.min.io",
				AccessKeyID:       "Q3AM3UQ867SPQQA43P2F",
				SecretAccessKey:   "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
				BucketName:        "mytestbucket",
				BucketLocation:    "us-east-1",
			},
		},
		{
			name: "Minio config should fail if storage service url is not provided",
			minioFile: []byte(`
access_key_id: Q3AM3UQ867SPQQA43P2F
secret_access_key: zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG
bucket_name: mytestbucket
bucket_location: us-east-1
`),
			configuration: blockstore.MinioConfiguration{},
			err:           common.NewError("process_minio_config_failed", "Unable to read minio config from minio config file"),
		},
		{
			name: "Minio config should fail if access key id is not provided",
			minioFile: []byte(`
storage_service_url: play.min.io
secret_access_key: zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG
bucket_name: mytestbucket
bucket_location: us-east-1
`),
			configuration: blockstore.MinioConfiguration{},
			err:           common.NewError("process_minio_config_failed", "Unable to read minio config from minio config file"),
		},
		{
			name: "Minio config should fail if secret access key is not provided",
			minioFile: []byte(`
storage_service_url: play.min.io
access_key_id: Q3AM3UQ867SPQQA43P2F
bucket_name: mytestbucket
bucket_location: us-east-1
`),
			configuration: blockstore.MinioConfiguration{},
			err:           common.NewError("process_minio_config_failed", "Unable to read minio config from minio config file"),
		},
		{
			name: "Minio config should fail if bucket name is not provided",
			minioFile: []byte(`
storage_service_url: play.min.io
access_key_id: Q3AM3UQ867SPQQA43P2F
secret_access_key: zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG
bucket_location: us-east-1
`),
			configuration: blockstore.MinioConfiguration{},
			err:           common.NewError("process_minio_config_failed", "Unable to read minio config from minio config file"),
		},
		{
			name: "Minio config should fail if bucket location is not provided",
			minioFile: []byte(`
storage_service_url: play.min.io
access_key_id: Q3AM3UQ867SPQQA43P2F
secret_access_key: zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG
bucket_name: mytestbucket
`),
			configuration: blockstore.MinioConfiguration{},
			err:           common.NewError("process_minio_config_failed", "Unable to read minio config from minio config file"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gotConfiguration, gotErr := processMinioConfig(bytes.NewBuffer(testCase.minioFile))
			require.Equal(t, testCase.configuration, gotConfiguration, "Configuration for minoFile is not set accurately")
			require.ErrorIs(t, gotErr, testCase.err, "Error is not as expected")
		})
	}
}

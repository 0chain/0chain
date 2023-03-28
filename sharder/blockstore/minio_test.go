package blockstore

import (
	"testing"

	"github.com/minio/minio-go"
)

func TestCreateMinioClientFromConfig(t *testing.T) {
	t.Parallel()

	mc := MinioConfiguration{
		StorageServiceURL: "play.min.io",
		AccessKeyID:       "Q3AM3UQ867SPQQA43P2F",
		SecretAccessKey:   "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		BucketName:        "mytestbucket",
		BucketLocation:    "us-east-1",
		DeleteLocal:       false,
		Secure:            false,
	}

	type args struct {
		config MinioConfiguration
	}
	tests := []struct {
		name    string
		args    args
		want    MinioClient
		wantErr bool
	}{
		{
			name: "Test_CreateMinioClientFromConfig_OK",
			args: args{
				config: mc,
			},
			wantErr: false,
		},
		{
			name: "Test_CreateMinioClientFromConfig_Invalid_URL_ERR",
			args: func() args {
				mc := mc
				mc.StorageServiceURL = "invalid#url"
				return args{config: mc}
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := CreateMinioClientFromConfig(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMinioClientFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_minioClient_BucketName(t *testing.T) {
	t.Parallel()

	bucketName := "mytestbucket"

	type fields struct {
		Client      *minio.Client
		bucketName  string
		deleteLocal bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_minioClient_BucketName_OK",
			fields: fields{
				bucketName: bucketName,
			},
			want: bucketName,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &minioClient{
				Client:      tt.fields.Client,
				bucketName:  tt.fields.bucketName,
				deleteLocal: tt.fields.deleteLocal,
			}
			if got := mc.BucketName(); got != tt.want {
				t.Errorf("BucketName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_minioClient_DeleteLocal(t *testing.T) {
	t.Parallel()

	type fields struct {
		Client      *minio.Client
		bucketName  string
		deleteLocal bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Test_minioClient_DeleteLocal_OK",
			fields: fields{
				deleteLocal: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &minioClient{
				Client:      tt.fields.Client,
				bucketName:  tt.fields.bucketName,
				deleteLocal: tt.fields.deleteLocal,
			}
			if got := mc.DeleteLocal(); got != tt.want {
				t.Errorf("DeleteLocal() = %v, want %v", got, tt.want)
			}
		})
	}
}

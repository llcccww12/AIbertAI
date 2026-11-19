package resource

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"aibertai.local/backend/internal/envconfig"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Storage interface {
	Put(context.Context, string, []byte, string) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	Ping(context.Context) error
}
type S3 struct {
	client   *s3.Client
	bucket   string
	mu       sync.Mutex
	checked  time.Time
	checkErr error
}

func NewS3(ctx context.Context) (*S3, error) {
	endpoint := envconfig.Getenv("AIBERTAI_S3_ENDPOINT")
	bucket := envconfig.Getenv("AIBERTAI_S3_BUCKET")
	if bucket == "" {
		bucket = resourceBucket
	}
	region := envconfig.Getenv("AIBERTAI_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}
	if endpoint == "" {
		return nil, errors.New("AIBERTAI_S3_ENDPOINT 不能为空")
	}
	if envconfig.Getenv("AIBERTAI_S3_ACCESS_KEY") == "" || envconfig.Getenv("AIBERTAI_S3_SECRET_KEY") == "" {
		return nil, errors.New("对象存储凭据不能为空")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region), awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(envconfig.Getenv("AIBERTAI_S3_ACCESS_KEY"), envconfig.Getenv("AIBERTAI_S3_SECRET_KEY"), "")))
	if err != nil {
		return nil, err
	}
	return &S3{client: s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = envconfig.Getenv("AIBERTAI_S3_FORCE_PATH_STYLE") != "false"
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	}), bucket: bucket}, nil
}
func (s *S3) Init(ctx context.Context) error {
	if err := s.ensureBucket(ctx, s.bucket); err != nil {
		return err
	}
	if s.bucket != resourceBucket {
		return nil
	}
	ready, err := s.bucketMigrationComplete(ctx)
	if err != nil || ready {
		return err
	}
	previous, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(legacyResourceBucket), MaxKeys: aws.Int32(1)})
	if _, missing := errors.AsType[*types.NoSuchBucket](err); missing {
		return nil
	}
	if err != nil {
		return err
	}
	if len(previous.Contents) != 0 {
		return errors.New("旧资源桶已有数据，请停服执行 migrate-resource-bucket 后再启动")
	}
	return nil
}

func (s *S3) ensureBucket(ctx context.Context, name string) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &name})
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[*types.NotFound](err); !ok {
		return err
	}
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &name})
	return err
}
func (s *S3) Put(ctx context.Context, key string, b []byte, mime string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, Body: bytes.NewReader(b), ContentType: &mime})
	return err
}

func (s *S3) PutReader(ctx context.Context, key string, body io.ReadSeeker, size int64, mime string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket, Key: &key, Body: body, ContentType: &mime, ContentLength: &size,
	})
	return err
}
func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	o, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	return o.Body, nil
}
func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	return err
}
func (s *S3) Ping(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.checked) < 5*time.Second {
		return s.checkErr
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, s.checkErr = s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	s.checked = time.Now()
	return s.checkErr
}

// DeletePrefix removes all objects below an explicit directory prefix.
func (s *S3) DeletePrefix(ctx context.Context, prefix string) error {
	if prefix == "" || !strings.HasSuffix(prefix, "/") {
		return errors.New("对象清理需要明确的目录前缀")
	}
	pages := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: &s.bucket, Prefix: &prefix})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, object := range page.Contents {
			if err := s.Delete(ctx, aws.ToString(object.Key)); err != nil {
				return err
			}
		}
	}
	return nil
}

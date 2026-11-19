package resource

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const resourceBucket = "aibertai-resources"
const legacyResourceBucket = "monkeyai-resources"
const bucketMigrationKey = ".aibertai-migration/resource-bucket-v1"
const bucketMigrationReceipt = "monkeyai-resources -> aibertai-resources v1\n"

func (s *S3) bucketMigrationComplete(ctx context.Context) (bool, error) {
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(resourceBucket), Key: aws.String(bucketMigrationKey),
	})
	if _, missing := errors.AsType[*types.NoSuchKey](err); missing {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer object.Body.Close()
	data, err := io.ReadAll(io.LimitReader(object.Body, int64(len(bucketMigrationReceipt)+1)))
	if err != nil {
		return false, err
	}
	if string(data) != bucketMigrationReceipt {
		return false, errors.New("资源桶迁移记录无效")
	}
	return true, nil
}

// MigrateBucket is an offline operation. Both buckets retain their data.
func MigrateBucket(ctx context.Context, rollback, onlyDefault bool) error {
	s, err := NewS3(ctx)
	if err != nil {
		return err
	}
	if onlyDefault && s.bucket != resourceBucket {
		return nil
	}
	return s.migrateBucket(ctx, rollback)
}

func (s *S3) migrateBucket(ctx context.Context, rollback bool) error {
	if s.bucket != resourceBucket && s.bucket != legacyResourceBucket {
		return errors.New("自定义资源桶不属于旧默认桶迁移")
	}
	if err := s.ensureBucket(ctx, resourceBucket); err != nil {
		return err
	}
	ready, err := s.bucketMigrationComplete(ctx)
	if err != nil || ready && !rollback {
		return err
	}
	source, target := legacyResourceBucket, resourceBucket
	if rollback {
		source, target = target, source
		if err := s.ensureBucket(ctx, target); err != nil {
			return err
		}
	}
	pages := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: &source})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if _, missing := errors.AsType[*types.NoSuchBucket](err); missing && !rollback {
			break
		}
		if err != nil {
			return err
		}
		for _, item := range page.Contents {
			key := aws.ToString(item.Key)
			if key == bucketMigrationKey {
				if rollback {
					continue
				}
				return errors.New("旧资源桶包含保留的迁移记录键")
			}
			if err := s.copyBucketObject(ctx, source, target, key); err != nil {
				return fmt.Errorf("迁移对象 %q: %w", key, err)
			}
		}
	}
	if rollback {
		_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(resourceBucket), Key: aws.String(bucketMigrationKey)})
	} else {
		_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(resourceBucket), Key: aws.String(bucketMigrationKey),
			Body: bytes.NewReader([]byte(bucketMigrationReceipt)), IfNoneMatch: aws.String("*"),
		})
		if err != nil {
			if completed, readErr := s.bucketMigrationComplete(ctx); readErr == nil && completed {
				return nil
			}
		}
	}
	return err
}

func (s *S3) copyBucketObject(ctx context.Context, source, target, key string) error {
	from, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &source, Key: &key})
	if err != nil {
		return err
	}
	defer from.Body.Close()
	to, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &target, Key: &key})
	if err == nil {
		defer to.Body.Close()
		if !reflect.DeepEqual(objectMetadata(from), objectMetadata(to)) {
			return errors.New("同名对象元数据不同，两个副本均已保留")
		}
		previous, current := sha256.New(), sha256.New()
		_, readSource := io.Copy(previous, from.Body)
		_, readTarget := io.Copy(current, to.Body)
		if err := errors.Join(readSource, readTarget); err != nil {
			return err
		}
		if !bytes.Equal(previous.Sum(nil), current.Sum(nil)) {
			return errors.New("同名对象内容不同，两个副本均已保留")
		}
		return nil
	}
	if _, missing := errors.AsType[*types.NoSuchKey](err); !missing {
		return err
	}
	temp, err := os.CreateTemp("", "aibertai-resource-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	size, err := io.Copy(temp, from.Body)
	if err != nil {
		return err
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	object := objectMetadata(from)
	object.Bucket, object.Key, object.Body = &target, &key, temp
	object.ContentLength, object.IfNoneMatch = &size, aws.String("*")
	_, err = s.client.PutObject(ctx, object)
	return err
}

func objectMetadata(o *s3.GetObjectOutput) *s3.PutObjectInput {
	return &s3.PutObjectInput{
		ContentType: o.ContentType, ContentDisposition: o.ContentDisposition, ContentEncoding: o.ContentEncoding,
		ContentLanguage: o.ContentLanguage, CacheControl: o.CacheControl, Expires: o.Expires, Metadata: o.Metadata,
	}
}

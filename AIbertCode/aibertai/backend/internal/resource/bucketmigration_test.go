package resource

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestBucketMigrationRestartAndRollback(t *testing.T) {
	type stored struct {
		body, mime string
	}
	buckets := map[string]map[string]stored{legacyResourceBucket: {"skills/test.zip": {"old skill content", "application/zip"}}}
	putCount := 0
	storageError := func(w http.ResponseWriter, status int, code string) {
		w.WriteHeader(status)
		io.WriteString(w, "<Error><Code>"+code+"</Code></Error>")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		objects, exists := buckets[bucket]
		if key == "" {
			switch r.Method {
			case http.MethodHead:
				if !exists {
					w.WriteHeader(http.StatusNotFound)
				}
			case http.MethodPut:
				buckets[bucket] = make(map[string]stored)
			case http.MethodGet:
				if !exists {
					storageError(w, http.StatusNotFound, "NoSuchBucket")
					return
				}
				io.WriteString(w, "<ListBucketResult><IsTruncated>false</IsTruncated>")
				for name := range objects {
					io.WriteString(w, "<Contents><Key>")
					xml.EscapeText(w, []byte(name))
					io.WriteString(w, "</Key></Contents>")
				}
				io.WriteString(w, "</ListBucketResult>")
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			item, present := objects[key]
			if !present {
				storageError(w, http.StatusNotFound, "NoSuchKey")
				return
			}
			w.Header().Set("Content-Type", item.mime)
			io.WriteString(w, item.body)
		case http.MethodPut:
			if _, present := objects[key]; r.Header.Get("If-None-Match") == "*" && present {
				storageError(w, http.StatusPreconditionFailed, "PreconditionFailed")
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			objects[key] = stored{string(body), r.Header.Get("Content-Type")}
			putCount++
		case http.MethodDelete:
			delete(objects, key)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)
	s := &S3{bucket: resourceBucket, client: s3.New(s3.Options{
		BaseEndpoint: aws.String(server.URL), UsePathStyle: true, Region: "us-east-1",
		Credentials: aws.AnonymousCredentials{}, RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
	})}
	ctx := t.Context()
	if err := s.Init(ctx); err == nil || !strings.Contains(err.Error(), "migrate-resource-bucket") {
		t.Fatalf("startup should require migration, got %v", err)
	}
	// A previous attempt copied the object but did not record completion.
	buckets[resourceBucket]["skills/test.zip"] = buckets[legacyResourceBucket]["skills/test.zip"]
	if err := s.migrateBucket(ctx, false); err != nil {
		t.Fatal(err)
	}
	if err := s.migrateBucket(ctx, false); err != nil || putCount != 1 {
		t.Fatalf("repeated migration copied data: %d %v", putCount, err)
	}
	if err := s.Init(ctx); err != nil {
		t.Fatal(err)
	}
	buckets[resourceBucket]["media/new"] = stored{"new resource", "text/plain"}
	if err := s.migrateBucket(ctx, true); err != nil {
		t.Fatal(err)
	}
	if err := s.migrateBucket(ctx, true); err != nil || putCount != 2 {
		t.Fatalf("repeated rollback copied data: %d %v", putCount, err)
	}
	if buckets[legacyResourceBucket]["media/new"].body != "new resource" || len(buckets[resourceBucket]) != 2 {
		t.Fatal("rollback lost a resource")
	}
	buckets[legacyResourceBucket]["media/new"] = stored{"different resource", "text/plain"}
	if err := s.migrateBucket(ctx, false); err == nil || !strings.Contains(err.Error(), "同名对象内容不同") {
		t.Fatalf("conflicting copy was accepted: %v", err)
	}
	if buckets[resourceBucket]["media/new"].body != "new resource" || buckets[legacyResourceBucket]["media/new"].body != "different resource" {
		t.Fatal("conflicting copy overwrote a resource")
	}
}

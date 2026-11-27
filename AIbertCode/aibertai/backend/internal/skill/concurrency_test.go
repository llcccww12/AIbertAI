package skill

import (
	"context"
	"testing"
)

func TestAcquireBoundsConcurrentPackageWork(t *testing.T) {
	s := NewService(nil, nil)
	var release [maxConcurrentSkillUploads]func()
	for i := range release {
		r, err := s.acquire(t.Context(), s.uploads)
		if err != nil {
			t.Fatalf("第 %d 个上传应当拿到名额: %v", i+1, err)
		}
		release[i] = r
	}

	// 名额用完时上传必须排队，请求被取消就立刻返回，不能无限堆积内存。
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.acquire(ctx, s.uploads); err == nil {
		t.Fatal("名额用完且请求已取消时应当返回错误")
	}

	release[0]()
	again, err := s.acquire(t.Context(), s.uploads)
	if err != nil {
		t.Fatalf("释放名额后应当能再拿到: %v", err)
	}
	again()
	for _, r := range release[1:] {
		r()
	}
}

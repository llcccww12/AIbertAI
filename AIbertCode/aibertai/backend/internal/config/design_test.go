package config

import "testing"

func TestDesignConfigDefaults(t *testing.T) {
	oldSecret := defaultDesignMediaJWTSecret
	t.Cleanup(func() { defaultDesignMediaJWTSecret = oldSecret })
	defaultDesignMediaJWTSecret = "build-secret"
	t.Setenv("MONKEYAI_INSTANCE_ID", "ignored-instance")
	t.Setenv("MONKEYAI_DESIGN_MEDIA_JWT_SECRET", "")
	for _, value := range []string{"", "https://override.example/api/design"} {
		t.Setenv("MONKEYAI_DESIGN_RESPUB_URL", value)
		got := loadDesign()
		if got.DesignRespubURL != value {
			t.Fatalf("发布源必须由部署方显式配置: %q", got.DesignRespubURL)
		}
		if got.DesignMediaJWTSecret != "" {
			t.Fatal("显式空环境变量未覆盖编译默认密钥")
		}
	}
	t.Setenv("MONKEYAI_DESIGN_MEDIA_JWT_SECRET", "runtime-secret")
	if got := loadDesign(); got.DesignMediaJWTSecret != "runtime-secret" {
		t.Fatal("运行环境密钥未覆盖编译默认值")
	}
}

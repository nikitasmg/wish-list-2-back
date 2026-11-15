package config

import "testing"

func TestLoadConfig_SantaBotTokenFallsBackToBotToken(t *testing.T) {
	cases := []struct{ name, santa, bot, want string }{
		{"empty santa", "", "x", "x"},
		{"santa set", "y", "x", "y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SANTA_BOT_TOKEN", tc.santa)
			t.Setenv("BOT_TOKEN", tc.bot)
			cfg, err := LoadConfig()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Notify.TelegramBotToken != tc.want {
				t.Fatalf("got %q, want %q", cfg.Notify.TelegramBotToken, tc.want)
			}
		})
	}
}

func TestLoadConfig_ExternalS3(t *testing.T) {
	values := map[string]string{
		"S3_ENDPOINT": "storage.yandexcloud.net", "S3_REGION": "ru-central1",
		"S3_BUCKET_NAME": "prosto-namekni-prod", "S3_ACCESS_KEY_ID": "test-access",
		"S3_SECRET_ACCESS_KEY": "test-secret", "S3_USE_SSL": "true",
		"S3_PUBLIC_URL":         "https://storage.yandexcloud.net/prosto-namekni-prod",
		"S3_AUTO_CREATE_BUCKET": "false", "MINIO_ENDPOINT": "old-local:9000",
	}
	for k, v := range values {
		t.Setenv(k, v)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.S3.Endpoint != "storage.yandexcloud.net" || cfg.S3.Region != "ru-central1" ||
		cfg.S3.BucketName != "prosto-namekni-prod" || cfg.S3.AccessKeyID != "test-access" ||
		cfg.S3.SecretAccessKey != "test-secret" || !cfg.S3.UseSSL || cfg.S3.AutoCreateBucket ||
		cfg.S3.PublicURL != "https://storage.yandexcloud.net/prosto-namekni-prod" {
		t.Fatal("S3 configuration was not loaded correctly")
	}
}

func TestLoadConfig_LegacyLocalStorage(t *testing.T) {
	for _, k := range []string{"S3_ENDPOINT", "S3_REGION", "S3_BUCKET_NAME", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY", "S3_PUBLIC_URL", "S3_USE_SSL"} {
		t.Setenv(k, "")
	}
	for k, v := range map[string]string{
		"MINIO_ENDPOINT": "minio:9000", "MINIO_BUCKET_NAME": "local-images",
		"MINIO_ROOT_USER": "local-access", "MINIO_ROOT_PASSWORD": "local-secret",
		"MINIO_PUBLIC_URL": "http://localhost:9000/", "MINIO_USE_SSL": "false",
		"S3_AUTO_CREATE_BUCKET": "true",
	} {
		t.Setenv(k, v)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.S3.Endpoint != "minio:9000" || cfg.S3.BucketName != "local-images" ||
		cfg.S3.AccessKeyID != "local-access" || cfg.S3.SecretAccessKey != "local-secret" ||
		cfg.S3.UseSSL || !cfg.S3.AutoCreateBucket || cfg.S3.PublicURL != "http://localhost:9000/local-images" {
		t.Fatal("legacy local storage settings are not compatible")
	}
}

func TestLoadConfig_BucketProvisioningDisabledByDefault(t *testing.T) {
	t.Setenv("S3_AUTO_CREATE_BUCKET", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.S3.AutoCreateBucket {
		t.Fatal("bucket provisioning must be opt-in")
	}
}

func TestLoadConfig_ExternalS3UsesTLSByDefault(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "storage.yandexcloud.net")
	t.Setenv("S3_USE_SSL", "")
	t.Setenv("MINIO_USE_SSL", "false")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.S3.UseSSL {
		t.Fatal("external S3 must default to TLS instead of inheriting local MinIO settings")
	}
}

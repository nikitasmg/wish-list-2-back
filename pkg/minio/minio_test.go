package minio

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"main/config"
)

func TestExternalS3UploadAndDelete(t *testing.T) {
	var mu sync.Mutex
	var uploaded, deleted string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Authorization"), "/ru-central1/s3/aws4_request") {
			t.Error("request is not signed for the configured region")
		}
		switch {
		case r.Method == http.MethodHead && r.URL.Path == "/prosto-namekni-prod/":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodHead && r.URL.Path == "/prosto-namekni-prod":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/prosto-namekni-prod/"):
			if r.Header.Get("Content-Type") != "image/webp" {
				t.Errorf("content type = %q", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || !strings.Contains(string(body), "WEBP") {
				t.Errorf("upload body is missing image data: %v", err)
			}
			mu.Lock()
			uploaded = strings.TrimPrefix(r.URL.Path, "/prosto-namekni-prod/")
			mu.Unlock()
			w.Header().Set("ETag", `"test-etag"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete:
			mu.Lock()
			deleted = strings.TrimPrefix(r.URL.Path, "/prosto-namekni-prod/")
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected storage request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer ts.Close()
	storage, err := New(config.S3Config{
		Endpoint: strings.TrimPrefix(ts.URL, "http://"),
		Region:   "ru-central1", BucketName: "prosto-namekni-prod",
		AccessKeyID: "test-access", SecretAccessKey: "test-secret",
		PublicURL: "https://storage.yandexcloud.net/prosto-namekni-prod/",
	})
	if err != nil {
		t.Fatal(err)
	}
	publicURL, err := storage.Upload("photo.webp", []byte("RIFF0000WEBPtest"))
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	key := uploaded
	mu.Unlock()
	if key == "" || publicURL != "https://storage.yandexcloud.net/prosto-namekni-prod/"+key {
		t.Fatalf("public URL = %q, object key = %q", publicURL, key)
	}
	if err := storage.Delete(key); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if deleted != key {
		t.Fatalf("deleted %q, uploaded %q", deleted, key)
	}
}

func TestExternalS3MissingOrForbiddenBucketDoesNotMutateStorage(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodHead {
					t.Errorf("existing-bucket mode must not mutate storage: %s %s", r.Method, r.URL)
				}
				w.WriteHeader(status)
			}))
			defer ts.Close()
			_, err := New(config.S3Config{
				Endpoint: strings.TrimPrefix(ts.URL, "http://"), Region: "ru-central1",
				BucketName: "prosto-namekni-prod", AccessKeyID: "test-access",
				SecretAccessKey: "test-secret", PublicURL: "https://example.com/images",
			})
			if err == nil {
				t.Fatal("expected an error for unavailable bucket")
			}
		})
	}
}

func TestExternalS3RejectsInvalidConfig(t *testing.T) {
	valid := config.S3Config{
		Endpoint: "storage.yandexcloud.net", Region: "ru-central1", UseSSL: true,
		BucketName: "prosto-namekni-prod", AccessKeyID: "test-access",
		SecretAccessKey: "test-secret", PublicURL: "https://example.com/images",
	}
	cases := []struct {
		name   string
		change func(*config.S3Config)
	}{
		{"missing endpoint", func(c *config.S3Config) { c.Endpoint = "" }},
		{"missing bucket", func(c *config.S3Config) { c.BucketName = "" }},
		{"missing key", func(c *config.S3Config) { c.AccessKeyID = "" }},
		{"missing secret", func(c *config.S3Config) { c.SecretAccessKey = "" }},
		{"invalid public URL", func(c *config.S3Config) { c.PublicURL = "not-a-url" }},
		{"public URL with query", func(c *config.S3Config) { c.PublicURL = "https://example.com/?token=secret" }},
		{"public URL with userinfo", func(c *config.S3Config) { c.PublicURL = "https://user:password@example.com" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.change(&cfg)
			if _, err := New(cfg); err == nil {
				t.Fatal("expected config error before making a network request")
			}
		})
	}
}

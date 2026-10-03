//go:build !cgo

package imageconv

// Convert без cgo ничего не конвертирует: libwebp — это C-библиотека, и без
// компилятора C пакет не собрался бы вовсе, а вместе с ним все, кто грузит
// картинки. Прод собирается с CGO_ENABLED=1 (см. Dockerfile) и сюда не попадает;
// заглушка нужна, чтобы go build и go test работали там, где gcc нет.
func Convert(data []byte) []byte {
	return data
}

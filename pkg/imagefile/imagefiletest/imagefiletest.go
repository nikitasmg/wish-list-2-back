// Package imagefiletest выдаёт настоящие картинки для тестов.
//
// С тех пор как загрузка проверяет сигнатуру файла, подставить []byte("data")
// вместо изображения больше нельзя — валидатор такое честно отвергает.
package imagefiletest

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// PNG — минимальная валидная PNG-картинка.
func PNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("imagefiletest.PNG: %v", err)
	}
	return buf.Bytes()
}

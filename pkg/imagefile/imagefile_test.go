package imagefile_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/pkg/imagefile"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func TestValidate_AcceptsPNGandJPEG(t *testing.T) {
	assert.NoError(t, imagefile.Validate(pngBytes(t)))
	assert.NoError(t, imagefile.Validate(jpegBytes(t)))
}

func TestValidate_RejectsEmpty(t *testing.T) {
	err := imagefile.Validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "пустой")
}

func TestValidate_RejectsNonImage(t *testing.T) {
	err := imagefile.Validate([]byte("%PDF-1.7\n... это не картинка ..."))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "формат не поддерживается")
}

func TestValidate_RejectsOversized(t *testing.T) {
	oversized := append(pngBytes(t), bytes.Repeat([]byte{0}, imagefile.MaxSize)...)

	err := imagefile.Validate(oversized)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "10 МБ")
}

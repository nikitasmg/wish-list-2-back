package imageconv_test

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/pkg/imageconv"
)

// helpers

func makeJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func makeStaticGIF(t *testing.T) []byte {
	t.Helper()
	p := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}}
	frame := image.NewPaletted(image.Rect(0, 0, 4, 4), p)
	var buf bytes.Buffer
	require.NoError(t, gif.Encode(&buf, frame, nil))
	return buf.Bytes()
}

func makeAnimatedGIF(t *testing.T) []byte {
	t.Helper()
	p := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}}
	f1 := image.NewPaletted(image.Rect(0, 0, 4, 4), p)
	f2 := image.NewPaletted(image.Rect(0, 0, 4, 4), p)
	anim := &gif.GIF{
		Image: []*image.Paletted{f1, f2},
		Delay: []int{10, 10},
	}
	var buf bytes.Buffer
	require.NoError(t, gif.EncodeAll(&buf, anim))
	return buf.Bytes()
}

func isWebP(data []byte) bool {
	return len(data) >= 12 &&
		string(data[:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP"
}

// tests

func TestConvert_JPEG_to_WebP(t *testing.T) {
	result := imageconv.Convert(makeJPEG(t))
	assert.True(t, isWebP(result), "expected WebP output for JPEG input")
}

func TestConvert_PNG_to_WebP(t *testing.T) {
	result := imageconv.Convert(makePNG(t))
	assert.True(t, isWebP(result), "expected WebP output for PNG input")
}

func TestConvert_StaticGIF_to_WebP(t *testing.T) {
	result := imageconv.Convert(makeStaticGIF(t))
	assert.True(t, isWebP(result), "expected WebP output for static GIF input")
}

func TestConvert_AnimatedGIF_passthrough(t *testing.T) {
	original := makeAnimatedGIF(t)
	result := imageconv.Convert(original)
	assert.Equal(t, original, result, "animated GIF must pass through unchanged")
}

func TestConvert_RandomBytes_passthrough(t *testing.T) {
	original := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
	result := imageconv.Convert(original)
	assert.Equal(t, original, result, "non-image bytes must pass through unchanged")
}

func TestConvert_CorruptJPEG_passthrough(t *testing.T) {
	// valid JPEG header, garbage body
	original := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 20)...)
	result := imageconv.Convert(original)
	assert.Equal(t, original, result, "corrupt JPEG must pass through unchanged")
}

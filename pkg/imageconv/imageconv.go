package imageconv

import (
	"bytes"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/chai2010/webp"
)

// Convert converts JPEG, PNG, and static GIF to WebP at 80% quality.
// Animated GIFs and non-image data are returned unchanged.
// Conversion errors also result in the original data being returned.
func Convert(data []byte) []byte {
	mime := http.DetectContentType(data)

	switch mime {
	case "image/gif":
		return convertGIF(data)
	case "image/jpeg", "image/png":
		return convertToWebP(data)
	default:
		return data
	}
}

func convertToWebP(data []byte) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Quality: 80}); err != nil {
		return data
	}
	return buf.Bytes()
}

func convertGIF(data []byte) []byte {
	anim, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return data
	}
	// animated GIF or zero-frame GIF — pass through unchanged
	if len(anim.Image) != 1 {
		return data
	}
	// static GIF — convert single frame to WebP
	var buf bytes.Buffer
	if err := webp.Encode(&buf, anim.Image[0], &webp.Options{Quality: 80}); err != nil {
		return data
	}
	return buf.Bytes()
}

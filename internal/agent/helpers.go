package agent

import "image"

// cropImage extracts a rectangular region from an image.
func cropImage(img image.Image, x, y, w, h int) image.Image {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	if si, ok := img.(subImager); ok {
		return si.SubImage(image.Rect(x, y, x+w, y+h))
	}

	// Fallback: copy pixels manually
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			dst.Set(dx, dy, img.At(x+dx, y+dy))
		}
	}
	return dst
}

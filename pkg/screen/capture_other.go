//go:build !darwin

package screen

import (
	"image"
)

func (c *Capturer) Capture() (image.Image, error) {
	return nil, nil
}

func (c *Capturer) CaptureRegion(rect image.Rectangle) (image.Image, error) {
	return nil, nil
}

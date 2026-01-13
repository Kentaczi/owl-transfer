//go:build darwin

package screen

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os/exec"
)

func (c *Capturer) Capture() (image.Image, error) {
	cmd := exec.Command("screencapture", "-x", "-t", "png", "-")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(output))
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (c *Capturer) CaptureRegion(rect image.Rectangle) (image.Image, error) {
	if rect.Empty() {
		return c.Capture()
	}

	cmd := exec.Command("screencapture", "-x", "-t", "png", "-R",
		fmt.Sprintf("%d,%d,%d,%d", rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy()), "-")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(output))
	if err != nil {
		return nil, err
	}

	return img, nil
}

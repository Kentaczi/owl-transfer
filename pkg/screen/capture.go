package screen

import (
	"image"
	"image/color"
	"math"
	"os/exec"
	"runtime"
)

type CaptureConfig struct {
	Region image.Rectangle
	FPS    int
}

type Capturer struct {
	config CaptureConfig
}

func NewCapturer(config CaptureConfig) *Capturer {
	return &Capturer{config: config}
}

func DetectQRRegion(img image.Image) image.Rectangle {
	bounds := img.Bounds()
	
	step := 10
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	
	nonWhiteCount := 0
	totalCount := 0
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			totalCount++
			
			r, g, b, _ := img.At(x, y).RGBA()
			
			if r < 0xFF00 || g < 0xFF00 || b < 0xFF00 {
				nonWhiteCount++
				
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	
	if nonWhiteCount == 0 || float64(nonWhiteCount)/float64(totalCount) < 0.1 {
		return image.Rectangle{}
	}
	
	padding := 20
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding
	
	if minX < bounds.Min.X {
		minX = bounds.Min.X
	}
	if minY < bounds.Min.Y {
		minY = bounds.Min.Y
	}
	if maxX > bounds.Max.X {
		maxX = bounds.Max.X
	}
	if maxY > bounds.Max.Y {
		maxY = bounds.Max.Y
	}
	
	return image.Rect(minX, minY, maxX, maxY)
}

func FindGridLines(img image.Image, blockSize int) (int, int, int, int) {
	bounds := img.Bounds()
	
	rows := make([]int, 0)
	cols := make([]int, 0)
	
	threshold := 128
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		edgeCount := 0
		for x := bounds.Min.X; x < bounds.Max.X-1; x++ {
			c1 := img.At(x, y)
			c2 := img.At(x+1, y)
			
			r1, g1, b1, _ := c1.RGBA()
			r2, g2, b2, _ := c2.RGBA()
			
			gray1 := int((r1+g1+b1)/3) >> 8
			gray2 := int((r2+g2+b2)/3) >> 8
			
			if abs(gray1-gray2) > threshold {
				edgeCount++
			}
		}
		
		if edgeCount > bounds.Dx()/4 {
			rows = append(rows, y)
		}
	}
	
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		edgeCount := 0
		for y := bounds.Min.Y; y < bounds.Max.Y-1; y++ {
			c1 := img.At(x, y)
			c2 := img.At(x, y+1)
			
			r1, g1, b1, _ := c1.RGBA()
			r2, g2, b2, _ := c2.RGBA()
			
			gray1 := int((r1+g1+b1)/3) >> 8
			gray2 := int((r2+g2+b2)/3) >> 8
			
			if abs(gray1-gray2) > threshold {
				edgeCount++
			}
		}
		
		if edgeCount > bounds.Dy()/4 {
			cols = append(cols, x)
		}
	}
	
	return len(rows), len(cols), bounds.Dx(), bounds.Dy()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func EstimateGridSize(img image.Image, blockSize int) (int, int) {
	detected := DetectQRRegion(img)
	if detected.Empty() {
		bounds := img.Bounds()
		return bounds.Dx() / blockSize, bounds.Dy() / blockSize
	}
	
	rows, cols, _, _ := FindGridLines(img, blockSize)
	
	if rows == 0 || cols == 0 {
		detected = DetectQRRegion(img)
		if detected.Empty() {
			return 0, 0
		}
		rows = detected.Dy() / blockSize
		cols = detected.Dx() / blockSize
	}
	
	if rows%2 == 0 {
		rows++
	}
	if cols%2 == 0 {
		cols++
	}
	
	return cols, rows
}

func GetDisplaySize() (int, int) {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("system_profiler", "SPDisplaysDataType", "-json")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return 1920, 1080
		}
		_ = output
		return 1920, 1080
	case "linux":
		cmd := exec.Command("xrandr", "--query", "--current")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return 1920, 1080
		}
		_ = output
		return 1920, 1080
	case "windows":
		return 1920, 1080
	default:
		return 1920, 1080
	}
}

type ColorAnalyzer struct{}

func (c *ColorAnalyzer) AnalyzeBlock(block image.Image) color.RGBA {
	bounds := block.Bounds()
	
	totalR, totalG, totalB := 0, 0, 0
	count := 0
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := block.At(x, y).RGBA()
			totalR += int(r >> 8)
			totalG += int(g >> 8)
			totalB += int(b >> 8)
			count++
		}
	}
	
	if count == 0 {
		return color.RGBA{0, 0, 0, 255}
	}
	
	return color.RGBA{
		R: uint8(totalR / count),
		G: uint8(totalG / count),
		B: uint8(totalB / count),
		A: 255,
	}
}

func (c *ColorAnalyzer) CalculateBlockSize(img image.Image, expectedGridWidth, expectedGridHeight int) int {
	bounds := img.Bounds()
	
	estBlockWidth := bounds.Dx() / expectedGridWidth
	estBlockHeight := bounds.Dy() / expectedGridHeight
	
	blockSize := (estBlockWidth + estBlockHeight) / 2
	
	if blockSize < 2 {
		blockSize = 10
	}
	
	return int(math.Ceil(float64(blockSize)))
}

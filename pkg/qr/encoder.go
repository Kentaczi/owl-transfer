package qr

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

type Block struct {
	R, G, B uint8
}

type Config struct {
	BlockSize     int
	GridWidth     int
	GridHeight    int
	BorderSize    int
	ErrorLevel    ErrorLevel
	UseColors     bool
}

type ErrorLevel int

const (
	ErrorLevelLow ErrorLevel = iota
	ErrorLevelMedium
	ErrorLevelHigh
)

type Encoder struct {
	config Config
}

func NewEncoder(config Config) *Encoder {
	return &Encoder{config: config}
}

func (e *Encoder) Encode(data []byte) []Block {
	blocks := make([]Block, e.config.GridWidth*e.config.GridHeight)
	
	redBits := 8
	greenBits := 8
	blueBits := 8
	
	switch e.config.ErrorLevel {
	case ErrorLevelLow:
		redBits = 8
		greenBits = 8
		blueBits = 8
	case ErrorLevelMedium:
		redBits = 6
		greenBits = 6
		blueBits = 6
	case ErrorLevelHigh:
		redBits = 4
		greenBits = 4
		blueBits = 4
	}
	
	redMask := (1 << redBits) - 1
	greenMask := (1 << greenBits) - 1
	blueMask := (1 << blueBits) - 1
	
	dataIndex := 0
	for y := 0; y < e.config.GridHeight; y++ {
		for x := 0; x < e.config.GridWidth; x++ {
			if dataIndex >= len(data) {
				blocks[y*e.config.GridWidth+x] = Block{0, 0, 0}
				continue
			}
			
			rVal := int(data[dataIndex])
			dataIndex++
			gVal := 0
			bVal := 0
			
			if dataIndex < len(data) {
				gVal = int(data[dataIndex])
				dataIndex++
			}
			if dataIndex < len(data) {
				bVal = int(data[dataIndex])
				dataIndex++
			}
			
			rShifted := rVal >> (8 - redBits)
			gShifted := gVal >> (8 - greenBits)
			bShifted := bVal >> (8 - blueBits)
			
			rCompressed := (rShifted * 255) / redMask
			gCompressed := (gShifted * 255) / greenMask
			bCompressed := (bShifted * 255) / blueMask
			
			blocks[y*e.config.GridWidth+x] = Block{
				R: uint8(rCompressed),
				G: uint8(gCompressed),
				B: uint8(bCompressed),
			}
		}
	}
	
	return blocks
}

func (e *Encoder) CreateImage(blocks []Block, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	blockPixelSize := width / (e.config.GridWidth + 2*e.config.BorderSize)
	
	for y := 0; y < e.config.GridHeight; y++ {
		for x := 0; x < e.config.GridWidth; x++ {
			block := blocks[y*e.config.GridWidth+x]
			
			startX := (x + e.config.BorderSize) * blockPixelSize
			startY := (y + e.config.BorderSize) * blockPixelSize
			
			c := color.RGBA{block.R, block.G, block.B, 255}
			
			rect := image.Rect(startX, startY, startX+blockPixelSize, startY+blockPixelSize)
			draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)
		}
	}
	
	borderColor := color.RGBA{255, 255, 255, 255}
	
	borderWidth := e.config.BorderSize * blockPixelSize
	borderRect := image.Rect(0, 0, width, borderWidth)
	draw.Draw(img, borderRect, &image.Uniform{borderColor}, image.Point{}, draw.Src)
	
	borderRect = image.Rect(0, height-borderWidth, width, height)
	draw.Draw(img, borderRect, &image.Uniform{borderColor}, image.Point{}, draw.Src)
	
	borderRect = image.Rect(0, 0, borderWidth, height)
	draw.Draw(img, borderRect, &image.Uniform{borderColor}, image.Point{}, draw.Src)
	
	borderRect = image.Rect(width-borderWidth, 0, width, height)
	draw.Draw(img, borderRect, &image.Uniform{borderColor}, image.Point{}, draw.Src)
	
	return img
}

type Decoder struct {
	config Config
}

func NewDecoder(config Config) *Decoder {
	return &Decoder{config: config}
}

func (d *Decoder) Decode(img image.Image) ([]Block, error) {
	bounds := img.Bounds()
	
	blockPixelSize := bounds.Dx() / (d.config.GridWidth + 2*d.config.BorderSize)
	
	blocks := make([]Block, d.config.GridWidth*d.config.GridHeight)
	
	redBits := 8
	greenBits := 8
	blueBits := 8
	
	switch d.config.ErrorLevel {
	case ErrorLevelLow:
		redBits = 8
		greenBits = 8
		blueBits = 8
	case ErrorLevelMedium:
		redBits = 6
		greenBits = 6
		blueBits = 6
	case ErrorLevelHigh:
		redBits = 4
		greenBits = 4
		blueBits = 4
	}
	
	redMask := (1 << redBits) - 1
	greenMask := (1 << greenBits) - 1
	blueMask := (1 << blueBits) - 1
	
	for y := 0; y < d.config.GridHeight; y++ {
		for x := 0; x < d.config.GridWidth; x++ {
			startX := (x + d.config.BorderSize) * blockPixelSize
			startY := (y + d.config.BorderSize) * blockPixelSize
			
			centerX := startX + blockPixelSize/2
			centerY := startY + blockPixelSize/2
			
			r, g, b, _ := img.At(centerX, centerY).RGBA()
			
			rVal := int(r >> 8)
			gVal := int(g >> 8)
			bVal := int(b >> 8)
			
			rShifted := (rVal * redMask) / 255
			gShifted := (gVal * greenMask) / 255
			bShifted := (bVal * blueMask) / 255
			
			rExpanded := (rShifted << (8 - redBits))
			gExpanded := (gShifted << (8 - greenBits))
			bExpanded := (bShifted << (8 - blueBits))
			
			if rShifted == redMask {
				rExpanded = 255
			}
			if gShifted == greenMask {
				gExpanded = 255
			}
			if bShifted == blueMask {
				bExpanded = 255
			}
			
			blocks[y*d.config.GridWidth+x] = Block{
				R: uint8(rExpanded),
				G: uint8(gExpanded),
				B: uint8(bExpanded),
			}
		}
	}
	
	return blocks, nil
}

func (d *Decoder) BlocksToData(blocks []Block) []byte {
	data := make([]byte, 0, len(blocks)*3)
	
	for _, block := range blocks {
		data = append(data, block.R)
		data = append(data, block.G)
		data = append(data, block.B)
	}
	
	return data
}

func OptimalGridSize(dataSize int) (width, height int) {
	area := int(math.Ceil(float64(dataSize) / 3.0))
	side := int(math.Ceil(math.Sqrt(float64(area))))
	
	if side%2 == 0 {
		side++
	}
	
	return side, side
}

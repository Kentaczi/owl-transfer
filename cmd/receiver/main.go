package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"image"
	"os"
	"sync"
	"time"
	
	"qrtransfer/pkg/chunk"
	"qrtransfer/pkg/qr"
	"qrtransfer/pkg/screen"
)

type ReceiverApp struct {
	app        fyne.App
	window     fyne.Window
	preview    *canvas.Image
	status     *widget.Label
	progress   *widget.ProgressBar
	
	screenCap  *screen.Capturer
	qrDec      *qr.Decoder
	chunkProc  *chunk.Processor
	
	receivedChunks map[uint32][]chunk.Chunk
	mu             sync.Mutex
	
	running      bool
	captureRate  time.Duration
	targetRegion image.Rectangle
	
	metadata    chunk.FileMetadata
	currentFile *os.File
}

func NewReceiverApp() *ReceiverApp {
	a := app.New()
	w := a.NewWindow("QR File Receiver")
	
	receiver := &ReceiverApp{
		app:        a,
		window:     w,
		screenCap:  screen.NewCapturer(screen.CaptureConfig{FPS: 10}),
		qrDec:      qr.NewDecoder(qr.Config{}),
		chunkProc:  chunk.NewProcessor(chunk.NewConfig(100, 1)),
		
		receivedChunks: make(map[uint32][]chunk.Chunk),
		captureRate:    500 * time.Millisecond,
		running:        false,
	}
	
	receiver.setupUI()
	
	return receiver
}

func (r *ReceiverApp) setupUI() {
	r.preview = &canvas.Image{
		FillMode: canvas.ImageFillContain,
	}
	
	r.preview.SetMinSize(fyne.NewSize(400, 400))
	
	startBtn := widget.NewButton("Start Capture", r.startCapture)
	startBtn.Disable()
	
	stopBtn := widget.NewButton("Stop Capture", r.stopCapture)
	stopBtn.Disable()
	
	saveBtn := widget.NewButton("Save File", r.saveFile)
	saveBtn.Disable()
	
	r.status = widget.NewLabel("Not capturing")
	r.progress = widget.NewProgressBar()
	
	rateSlider := widget.NewSlider(0.2, 2.0)
	rateSlider.Value = 0.5
	rateSlider.OnChanged = func(value float64) {
		r.captureRate = time.Duration(value * float64(time.Second))
	}
	
	controls := container.NewVBox(
		widget.NewLabel("Capture Rate (seconds):"),
		rateSlider,
		startBtn,
		stopBtn,
		saveBtn,
		r.status,
		r.progress,
	)
	
	content := container.NewHSplit(
		container.NewCenter(r.preview),
		controls,
	)
	
	r.window.SetContent(content)
	r.window.Resize(fyne.NewSize(800, 600))
	
	r.preview.Image = r.createPlaceholderImage()
	r.preview.Refresh()
}

func (r *ReceiverApp) startCapture() {
	if r.running {
		return
	}
	
	r.running = true
	
	go r.captureLoop()
}

func (r *ReceiverApp) stopCapture() {
	r.running = false
}

func (r *ReceiverApp) captureLoop() {
	for r.running {
		r.captureFrame()
		time.Sleep(r.captureRate)
	}
}

func (r *ReceiverApp) captureFrame() {
	img, err := r.screenCap.CaptureRegion(r.targetRegion)
	if err != nil {
		return
	}
	
	if img == nil {
		return
	}
	
	r.preview.Image = img
	r.preview.Refresh()
	
	gridWidth, gridHeight := screen.EstimateGridSize(img, 20)
	
	if gridWidth == 0 || gridHeight == 0 {
		return
	}
	
	r.qrDec = qr.NewDecoder(qr.Config{
		GridWidth:  gridWidth,
		GridHeight: gridHeight,
		BorderSize: 1,
	})
	
	blocks, err := r.qrDec.Decode(img)
	if err != nil {
		return
	}
	
	data := r.qrDec.BlocksToData(blocks)
	
	chunkData, err := r.chunkProc.DeserializeChunk(data)
	if err != nil {
		return
	}
	
	if !chunk.VerifyChunk(chunkData) {
		return
	}
	
	r.mu.Lock()
	
	if chunkData.Index == 0 && r.metadata.TotalChunks == 0 {
		metadataData, err := r.chunkProc.DeserializeMetadata(chunkData.Data)
		if err == nil {
			r.metadata = metadataData
		}
	}
	
	r.receivedChunks[chunkData.Index] = append(r.receivedChunks[chunkData.Index], chunkData)
	r.mu.Unlock()
	
	r.updateStatus(chunkData.Total, uint32(len(r.receivedChunks)))
}

func (r *ReceiverApp) updateStatus(total, received uint32) {
	percent := float64(0)
	if total > 0 {
		percent = float64(received) / float64(total) * 100
	}
	
	r.progress.SetValue(percent / 100)
	r.status.SetText(fmt.Sprintf("Received %d/%d chunks (%.1f%%)", received, total, percent))
}

func (r *ReceiverApp) saveFile() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, r.window)
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()
		
		r.assembleFile(writer)
	}, r.window)
}

func (r *ReceiverApp) assembleFile(writer fyne.URIWriteCloser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	missingChunks := make(map[uint32]bool)
	for i := uint32(0); i < r.metadata.TotalChunks; i++ {
		missingChunks[i] = true
	}
	
	for i := uint32(0); i < r.metadata.TotalChunks; i++ {
		chunks, ok := r.receivedChunks[i]
		if !ok {
			continue
		}
		
		delete(missingChunks, i)
		
		if len(chunks) == 0 {
			continue
		}
		
		data := chunks[0].Data
		for _, c := range chunks {
			if chunk.VerifyChunk(c) {
				data = c.Data
				break
			}
		}
		
		_, err := writer.Write(data)
		if err != nil {
			r.status.SetText(fmt.Sprintf("Error writing file: %v", err))
			return
		}
	}
	
	if len(missingChunks) > 0 {
		r.status.SetText(fmt.Sprintf("Warning: %d chunks missing", len(missingChunks)))
	} else {
		r.status.SetText("File assembled successfully!")
	}
}

func (r *ReceiverApp) createPlaceholderImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, image.White)
		}
	}
	
	return img
}

func (r *ReceiverApp) Run() {
	r.window.ShowAndRun()
}

func main() {
	app := NewReceiverApp()
	app.Run()
}

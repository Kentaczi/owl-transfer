package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"image"
	"os"
	"time"

	"qrtransfer/pkg/chunk"
	"qrtransfer/pkg/qr"
)

type SenderApp struct {
	app       fyne.App
	window    fyne.Window
	image     *canvas.Image
	filename  string
	origName  string
	startBtn  *widget.Button
	stopBtn   *widget.Button
	chunkProc *chunk.Processor
	qrEnc     *qr.Encoder
	qrConfig  qr.Config
	status    *widget.Label

	currentChunk uint32
	totalChunks  uint32
	metadata     chunk.FileMetadata
	chunks       [][]chunk.Chunk

	refreshRate time.Duration
	running     bool
}

func NewSenderApp() *SenderApp {
	a := app.New()
	w := a.NewWindow("QR File Sender")

	sender := &SenderApp{
		app:         a,
		window:      w,
		chunkProc:   chunk.NewProcessor(chunk.NewConfig(100, 1)),
		qrEnc:       qr.NewEncoder(qr.Config{}),
		qrConfig:    qr.Config{},
		refreshRate: 2 * time.Second,
		running:     false,
	}

	sender.setupUI()

	return sender
}

func (s *SenderApp) setupUI() {
	s.image = &canvas.Image{
		FillMode: canvas.ImageFillContain,
	}

	s.image.SetMinSize(fyne.NewSize(400, 400))

	selectBtn := widget.NewButton("Select File", s.selectFile)

	s.startBtn = widget.NewButton("Start Transfer", s.startTransfer)
	s.startBtn.Disable()

	s.stopBtn = widget.NewButton("Stop Transfer", s.stopTransfer)
	s.stopBtn.Disable()

	s.status = widget.NewLabel("No file selected")

	rateSlider := widget.NewSlider(0.5, 5.0)
	rateSlider.Value = 2.0
	rateSlider.OnChanged = func(value float64) {
		s.refreshRate = time.Duration(value * float64(time.Second))
	}

	redundancySelect := widget.NewSelect([]string{"1x", "2x", "3x"}, func(value string) {
		_ = value
	})
	redundancySelect.SetSelectedIndex(0)

	errorLevelSelect := widget.NewSelect([]string{"Low", "Medium", "High"}, func(value string) {
		switch value {
		case "Low":
			s.qrConfig.ErrorLevel = qr.ErrorLevelLow
		case "Medium":
			s.qrConfig.ErrorLevel = qr.ErrorLevelMedium
		case "High":
			s.qrConfig.ErrorLevel = qr.ErrorLevelHigh
		}
	})
	errorLevelSelect.SetSelectedIndex(1)

	controls := container.NewVBox(
		widget.NewLabel("File:"),
		selectBtn,
		widget.NewLabel("Error Correction:"),
		errorLevelSelect,
		widget.NewLabel("Redundancy:"),
		redundancySelect,
		widget.NewLabel("Refresh Rate (seconds):"),
		rateSlider,
		s.startBtn,
		s.stopBtn,
		s.status,
	)

	content := container.NewHSplit(
		container.NewCenter(s.image),
		controls,
	)

	s.window.SetContent(content)
	s.window.Resize(fyne.NewSize(800, 600))

	s.image.Image = s.createPlaceholderImage()
	s.image.Refresh()
}

func (s *SenderApp) selectFile() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		if reader == nil {
			s.status.SetText("No file selected")
			return
		}

		uri := reader.URI()
		s.filename = uri.String()
		if uri.Scheme() == "file" {
			s.filename = uri.Path()
		}
		s.origName = uri.Name()
		s.status.SetText("Selected: " + s.origName)
		reader.Close()

		s.loadFile()
	}, s.window)
}

func (s *SenderApp) loadFile() {
	file, err := os.Open(s.filename)
	if err != nil {
		dialog.ShowError(err, s.window)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		dialog.ShowError(err, s.window)
		return
	}

	s.metadata = chunk.FileMetadata{
		Filename:   s.origName,
		FileSize:   uint64(fileInfo.Size()),
		ChunkSize:  uint32(s.chunkProc.Config().ChunkSize),
		Timestamp:  uint64(time.Now().UnixNano()),
		Redundancy: 1,
	}

	s.metadata.TotalChunks = uint32((fileInfo.Size() + int64(s.chunkProc.Config().ChunkSize) - 1) / int64(s.chunkProc.Config().ChunkSize))

	file.Seek(0, 0)
	chunks, err := s.chunkProc.CreateChunks(file, s.metadata, 1)
	if err != nil {
		dialog.ShowError(err, s.window)
		return
	}

	s.chunks = chunks
	s.currentChunk = 0
	s.totalChunks = s.metadata.TotalChunks
	s.startBtn.Enable()

	s.image.Image = s.createPlaceholderImage()
	s.image.Refresh()
}

func (s *SenderApp) startTransfer() {
	if len(s.chunks) == 0 {
		return
	}

	s.running = true
	fyne.DoAndWait(func() {
		s.startBtn.Disable()
		s.stopBtn.Enable()
		s.status.SetText("Transfer running...")
	})
	s.displayCurrentChunk()
}

func (s *SenderApp) stopTransfer() {
	s.running = false
	fyne.DoAndWait(func() {
		s.stopBtn.Disable()
		s.startBtn.Enable()
		s.status.SetText("Transfer stopped")
	})
}

func (s *SenderApp) displayCurrentChunk() {
	if !s.running || s.currentChunk > s.totalChunks {
		s.running = false
		fyne.DoAndWait(func() {
			s.stopBtn.Disable()
			s.startBtn.Enable()
			if s.currentChunk > s.totalChunks {
				s.status.SetText("Transfer complete!")
			}
		})
		return
	}

	var serialized []byte
	var err error

	if s.currentChunk == 0 {
		metadataBytes, err := s.chunkProc.SerializeMetadata(s.metadata)
		if err != nil {
			s.running = false
			return
		}

		metadataChunk := chunk.Chunk{
			Index:     0,
			Total:     s.totalChunks + 1,
			Data:      metadataBytes,
			Timestamp: s.metadata.Timestamp,
		}
		serialized, err = s.chunkProc.SerializeChunk(metadataChunk)
		if err != nil {
			s.running = false
			return
		}
	} else {
		chunkSet := s.chunks[s.currentChunk-1]
		serialized, err = s.chunkProc.SerializeChunk(chunkSet[0])
		if err != nil {
			s.running = false
			return
		}
	}

	s.qrConfig.GridWidth, s.qrConfig.GridHeight = qr.OptimalGridSize(len(serialized))

	s.qrEnc = qr.NewEncoder(s.qrConfig)

	blocks := s.qrEnc.Encode(serialized)

	img := s.qrEnc.CreateImage(blocks, 400, 400)

	fyne.DoAndWait(func() {
		s.image.Image = img
		s.image.Refresh()
	})

	s.currentChunk++

	if s.currentChunk > s.totalChunks+1 {
		s.running = false
		fyne.DoAndWait(func() {
			s.stopBtn.Disable()
			s.startBtn.Enable()
			s.status.SetText("Transfer complete!")
		})
		return
	}

	go func() {
		time.Sleep(s.refreshRate)
		if s.running {
			s.displayCurrentChunk()
		}
	}()
}

func (s *SenderApp) createPlaceholderImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))

	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, image.White)
		}
	}

	return img
}

func (s *SenderApp) Run() {
	s.window.ShowAndRun()
}

func main() {
	app := NewSenderApp()
	app.Run()
}

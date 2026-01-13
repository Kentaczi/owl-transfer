# QR File Transfer

A Go-based tool for transferring files between air-gapped systems using color-based QR codes displayed on screen and captured via screen recording.

## Overview

This project implements a novel file transfer method that:
- Encodes file data into color-based QR codes (RGB blocks)
- Displays QR codes in a GUI window on the sender
- Captures and decodes QR codes via screen recording on the receiver
- Supports Reed-Solomon error correction for reliability
- Handles large files through chunking and redundancy

## Features

### Sender (`qrtransfer-sender`)
- **GUI Interface**: Cross-platform desktop app using Fyne
- **File Selection**: Browse and select files to transfer
- **Configurable Settings**:
  - Error correction levels (Low/Medium/High)
  - Redundancy (1x/2x/3x)
  - Refresh rate (0.5-5 seconds)
- **Auto-refresh**: Automatically cycles through QR codes
- **Progress Tracking**: Shows current chunk and transfer status

### Receiver (`qrtransfer-receiver`)
- **Screen Capture**: Real-time screen monitoring
- **QR Detection**: Automatic QR code detection and decoding
- **File Reassembly**: Reconstructs original file from chunks
- **Gap Filling**: Handles missing chunks gracefully
- **Progress Display**: Shows transfer completion percentage

### Technical Features
- **Color-Based Encoding**: Uses RGB values for high data density
- **Reed-Solomon Error Correction**: Configurable error correction levels
- **Chunking**: Supports files of any size through chunking
- **Redundancy**: Overlapping chunks for fault tolerance
- **Compression**: Built-in compression for efficient transfer
- **Cross-Platform**: macOS, Linux, Windows support

## Installation

### Prerequisites
- Go 1.25+ 
- For macOS: `screencapture` command-line tool (built-in)
- For Linux: `scrot` or similar screen capture tool
- For Windows: Screen capture capabilities

### Build from Source

```bash
git clone <repository-url>
cd qrtransfer
go mod tidy
go build ./cmd/sender
go build ./cmd/receiver
```

### Build for Specific Platform

```bash
# macOS
GOOS=darwin go build ./cmd/sender -o qrtransfer-sender-darwin
GOOS=darwin go build ./cmd/receiver -o qrtransfer-receiver-darwin

# Linux
GOOS=linux go build ./cmd/sender -o qrtransfer-sender-linux
GOOS=linux go build ./cmd/receiver -o qrtransfer-receiver-linux

# Windows
GOOS=windows go build ./cmd/sender -o qrtransfer-sender.exe
GOOS=windows go build ./cmd/receiver -o qrtransfer-receiver.exe
```

## Usage

### Basic Workflow

1. **On Remote Machine (Sender)**:
   ```bash
   ./qrtransfer-sender
   ```
   - Click "Select File" to choose file to transfer
   - Configure error correction and redundancy settings
   - Click "Start Transfer" to begin displaying QR codes

2. **On Local Machine (Receiver)**:
   ```bash
   ./qrtransfer-receiver
   ```
   - Position the receiver window to capture the sender's QR codes
   - Click "Start Capture" to begin monitoring
   - Wait for transfer to complete
   - Click "Save File" to reconstruct and save the file

### Configuration Options

#### Error Correction Levels
- **Low**: 8-bit color depth, maximum data capacity
- **Medium**: 6-bit color depth, balanced error correction
- **High**: 4-bit color depth, maximum error correction

#### Redundancy Levels
- **1x**: No redundancy (fastest)
- **2x**: Each chunk sent twice
- **3x**: Each chunk sent three times (most reliable)

#### Refresh Rate
- **0.5-5 seconds**: Controls how quickly QR codes cycle
- **Slower**: More reliable capture
- **Faster**: Faster transfer

## Architecture

### Core Components

```
qrtransfer/
├── cmd/
│   ├── sender/          # GUI sender application
│   └── receiver/        # GUI receiver application
├── pkg/
│   ├── qr/             # QR encoding/decoding
│   ├── ec/             # Reed-Solomon error correction
│   ├── chunk/          # File chunking and metadata
│   ├── screen/         # Screen capture utilities
│   └── compress/       # Compression algorithms
```

### Data Flow

1. **File Processing**:
   - File → Chunks → Compression → Error Correction
   - Metadata encoding (filename, size, chunk count)

2. **QR Encoding**:
   - Data → RGB blocks → Grid layout → Image generation
   - Configurable color depth based on error correction level

3. **Display & Capture**:
   - GUI displays QR codes with automatic refresh
   - Screen capture monitors for QR codes
   - Automatic detection and decoding

4. **File Reassembly**:
   - QR decode → Data extraction → Chunk verification
   - Gap filling and redundancy resolution
   - File reconstruction and saving

## Error Correction

The system uses Reed-Solomon error correction with configurable levels:

### Error Correction Capacity
- **Low**: Can correct up to 10% data loss
- **Medium**: Can correct up to 20% data loss  
- **High**: Can correct up to 30% data loss

### Redundancy Strategy
- Overlapping chunks provide additional recovery capability
- Multiple copies of each chunk ensure reliability
- Automatic selection of best-quality chunks during reassembly

## Performance

### Data Capacity
- **Low Error Correction**: ~16.7M values per pixel (24-bit RGB)
- **Medium Error Correction**: ~262K values per pixel (18-bit RGB)
- **High Error Correction**: ~4K values per pixel (12-bit RGB)

### Transfer Speed
- **Typical**: 1-5 KB per QR code
- **With 2-second refresh**: 0.5-2.5 KB/second
- **Large files**: Automatically chunked and transferred sequentially

## Troubleshooting

### Common Issues

1. **QR Code Not Detected**:
   - Ensure receiver window captures the sender's QR codes
   - Adjust capture rate if QR codes are changing too fast
   - Check for proper lighting and screen visibility

2. **Missing Chunks**:
   - Increase redundancy level
   - Slow down refresh rate
   - Ensure stable screen capture

3. **File Corruption**:
   - Use higher error correction level
   - Check for screen interference or compression
   - Verify complete capture of all QR codes

### Debug Mode

Enable verbose logging by setting environment variable:
```bash
export QRTRANSFER_DEBUG=1
./qrtransfer-sender
./qrtransfer-receiver
```

## Development

### Project Structure

- **pkg/qr/**: QR code encoding and decoding with color blocks
- **pkg/ec/**: Reed-Solomon error correction implementation
- **pkg/chunk/**: File chunking, metadata, and serialization
- **pkg/screen/**: Cross-platform screen capture utilities
- **cmd/sender/**: Fyne-based GUI sender application
- **cmd/receiver/**: Fyne-based GUI receiver application

### Adding Features

1. **New Error Correction Algorithms**:
   - Implement in `pkg/ec/`
   - Add configuration options in GUI

2. **Additional Compression**:
   - Add to `pkg/compress/`
   - Integrate with chunk processor

3. **Enhanced Screen Capture**:
   - Extend `pkg/screen/`
   - Add platform-specific optimizations

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## Acknowledgments

- Fyne framework for cross-platform GUI
- Reed-Solomon error correction algorithms
- Screen capture utilities across platforms
- Color-based QR code encoding research
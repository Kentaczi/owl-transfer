package chunk

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"io"
)

type Chunk struct {
	Index     uint32
	Total     uint32
	Data      []byte
	Checksum  [32]byte
	Timestamp uint64
}

type FileMetadata struct {
	Filename    string
	FileSize    uint64
	ChunkSize   uint32
	TotalChunks uint32
	Checksum    [32]byte
	Timestamp   uint64
	Redundancy  uint8
}

type Progress struct {
	CurrentChunk   uint32
	TotalChunks    uint32
	BytesReceived  uint64
	PercentComplete float64
}

type Config struct {
	ChunkSize  int
	Redundancy int
}

func NewConfig(chunkSize int, redundancy int) Config {
	return Config{
		ChunkSize:  chunkSize,
		Redundancy: redundancy,
	}
}

type Processor struct {
	config Config
}

func NewProcessor(config Config) *Processor {
	return &Processor{config: config}
}

func (p *Processor) Config() Config {
	return p.config
}

func (p *Processor) CreateChunks(file io.Reader, metadata FileMetadata, redundancy uint8) ([][]Chunk, error) {
	chunks := make([][]Chunk, 0)
	
	data := make([]byte, p.config.ChunkSize)
	chunkIndex := uint32(0)
	
	for {
		n, err := file.Read(data)
		if n == 0 && err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
		
		chunkData := make([]byte, n)
		copy(chunkData, data[:n])
		
		checksum := sha256.Sum256(chunkData)
		
		chunk := Chunk{
			Index:     chunkIndex,
			Total:     metadata.TotalChunks,
			Data:      chunkData,
			Checksum:  checksum,
			Timestamp: metadata.Timestamp,
		}
		
		redundantChunks := make([]Chunk, 1+int(redundancy))
		redundantChunks[0] = chunk
		
		for i := 1; i <= int(redundancy); i++ {
			redundantChunks[i] = Chunk{
				Index:     chunkIndex,
				Total:     metadata.TotalChunks,
				Data:      chunkData,
				Checksum:  checksum,
				Timestamp: metadata.Timestamp + uint64(i),
			}
		}
		
		chunks = append(chunks, redundantChunks)
		chunkIndex++
	}
	
	return chunks, nil
}

func (p *Processor) SerializeChunk(chunk Chunk) ([]byte, error) {
	serialized := make([]byte, 0)
	
	serialized = append(serialized, byte(chunk.Index>>24))
	serialized = append(serialized, byte(chunk.Index>>16))
	serialized = append(serialized, byte(chunk.Index>>8))
	serialized = append(serialized, byte(chunk.Index))
	
	serialized = append(serialized, byte(chunk.Total>>24))
	serialized = append(serialized, byte(chunk.Total>>16))
	serialized = append(serialized, byte(chunk.Total>>8))
	serialized = append(serialized, byte(chunk.Total))
	
	serialized = append(serialized, byte(len(chunk.Data)>>24))
	serialized = append(serialized, byte(len(chunk.Data)>>16))
	serialized = append(serialized, byte(len(chunk.Data)>>8))
	serialized = append(serialized, byte(len(chunk.Data)))
	
	serialized = append(serialized, chunk.Data...)
	
	serialized = append(serialized, chunk.Checksum[:]...)
	
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, chunk.Timestamp)
	serialized = append(serialized, ts...)
	
	return serialized, nil
}

func (p *Processor) DeserializeChunk(data []byte) (Chunk, error) {
	if len(data) < 45 {
		return Chunk{}, io.ErrShortBuffer
	}
	
	chunk := Chunk{}
	
	chunk.Index = uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	chunk.Total = uint32(data[4])<<24 | uint32(data[5])<<16 | uint32(data[6])<<8 | uint32(data[7])
	
	dataLen := uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])
	
	expectedLen := 12 + dataLen + 32 + 8
	if uint32(len(data)) < expectedLen {
		return Chunk{}, io.ErrShortBuffer
	}
	
	chunk.Data = make([]byte, dataLen)
	copy(chunk.Data, data[12:12+dataLen])
	
	copy(chunk.Checksum[:], data[12+dataLen:12+dataLen+32])
	
	chunk.Timestamp = binary.BigEndian.Uint64(data[12+dataLen+32:])
	
	return chunk, nil
}

func (p *Processor) SerializeMetadata(metadata FileMetadata) ([]byte, error) {
	return json.Marshal(metadata)
}

func (p *Processor) DeserializeMetadata(data []byte) (FileMetadata, error) {
	var metadata FileMetadata
	err := json.Unmarshal(data, &metadata)
	return metadata, err
}

func VerifyChunk(chunk Chunk) bool {
	checksum := sha256.Sum256(chunk.Data)
	return checksum == chunk.Checksum
}

func CalculateProgress(received, total uint32, bytesReceived, fileSize uint64) Progress {
	percent := float64(0)
	if fileSize > 0 {
		percent = float64(bytesReceived) / float64(fileSize) * 100
	}
	
	return Progress{
		CurrentChunk:   received,
		TotalChunks:    total,
		BytesReceived:  bytesReceived,
		PercentComplete: percent,
	}
}

func CountTotalBytes(chunks [][]Chunk) uint64 {
	var total uint64
	for _, redundantSet := range chunks {
		for _, c := range redundantSet {
			total += uint64(len(c.Data))
		}
	}
	return total
}

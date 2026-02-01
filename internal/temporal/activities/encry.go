package activities

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"go.temporal.io/sdk/activity"
	"golang.org/x/crypto/hkdf"
)

type FileEncryptionActivityInput struct {
	FilePath string
	Key      string // Hex-encoded 32-byte master key
}

type FileEncryptionActivityOutput struct {
	FilePath string
	Size     int64
	Checksum string // SHA256
}

// AES-256-CTR with HMAC-SHA256 for streaming authenticated encryption
func (a *Activities) FileEncryptionActivity(ctx context.Context, input FileEncryptionActivityInput) (*FileEncryptionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileEncryptionActivity started with streaming, AES-256, and HKDF")

	// 1. Decode and validate the master key
	masterKey, err := hex.DecodeString(input.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("invalid master key length: must be 32 bytes")
	}

	// 2. Open input file
	inputFile, err := os.Open(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// 3. Create output file
	outputFilePath := input.FilePath + ".enc"
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// 4. Generate a random salt for HKDF
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// 5. Derive encryption and MAC keys using HKDF
	kdf := hkdf.New(sha256.New, masterKey, salt, []byte("agent-backup-encryption"))

	encKey := make([]byte, 32) // 32 bytes for AES-256
	if _, err := io.ReadFull(kdf, encKey); err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	macKey := make([]byte, 32) // 32 bytes for HMAC-SHA256
	if _, err := io.ReadFull(kdf, macKey); err != nil {
		return nil, fmt.Errorf("failed to derive MAC key: %w", err)
	}

	// 6. Generate IV (Initialization Vector) for AES-CTR
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// 7. Create AES-256 block cipher
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-256 cipher block: %w", err)
	}

	// 8. Create CTR stream cipher
	stream := cipher.NewCTR(block, iv)

	// 9. Create HMAC-SHA256 for authentication and SHA256 for file checksum
	hmac := hmac.New(sha256.New, macKey)
	fileChecksum := sha256.New()

	// 10. Write salt and IV to the start of the file and update checksum
	// The order is: salt -> iv -> ciphertext -> hmac
	for _, data := range [][]byte{salt, iv} {
		if _, err := outputFile.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write header to output file: %w", err)
		}
		if _, err := fileChecksum.Write(data); err != nil {
			return nil, fmt.Errorf("failed to write header to checksum: %w", err)
		}
	}

	// 11. Create a multi-writer to write to both the file and the HMAC
	multiWriter := io.MultiWriter(outputFile, hmac, fileChecksum)

	// 12. Stream data: read -> encrypt -> write to file/hmac/checksum
	buf := make([]byte, 32*1024)
	for {
		n, err := inputFile.Read(buf)
		if n > 0 {
			ciphertextChunk := buf[:n]
			stream.XORKeyStream(ciphertextChunk, ciphertextChunk) // Encrypt in-place
			if _, writeErr := multiWriter.Write(ciphertextChunk); writeErr != nil {
				return nil, fmt.Errorf("failed to write ciphertext chunk: %w", writeErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read from input file: %w", err)
		}
	}

	// 13. Calculate and write the HMAC tag
	macTag := hmac.Sum(nil)
	if _, err := outputFile.Write(macTag); err != nil {
		return nil, fmt.Errorf("failed to write HMAC tag: %w", err)
	}
	if _, err := fileChecksum.Write(macTag); err != nil {
		return nil, fmt.Errorf("failed to update checksum with HMAC tag: %w", err)
	}

	// 14. Finalize checksum
	finalChecksum := hex.EncodeToString(fileChecksum.Sum(nil))

	// 15. Get file size by closing the file and using os.Stat
	if err := outputFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close output file before stat: %w", err)
	}

	fileInfo, err := os.Stat(outputFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for output file: %w", err)
	}
	finalSize := fileInfo.Size()

	logger.Debug("FileEncryptionActivity completed", "output_path", outputFilePath, "size", finalSize, "checksum", finalChecksum)

	return &FileEncryptionActivityOutput{
		FilePath: outputFilePath,
		Size:     finalSize,
		Checksum: finalChecksum,
	}, nil
}

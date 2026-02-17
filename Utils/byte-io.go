package Utils

import (
	"fmt"
	"os"
)

func SaveToBinFile(filename string, data []byte) error {
	file, err := os.Create(filename)

	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	defer file.Close()

	_, err = file.Write(data)

	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

func ReadFromBinFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	fileInfo, err := file.Stat()

	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	data := make([]byte, fileInfo.Size())

	_, err = file.Read(data)

	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

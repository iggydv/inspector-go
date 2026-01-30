package dataset

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"inspectgo/pkg/core"
)

type FileDataset struct {
	Path     string
	NameHint string
}

func NewFileDataset(path string) *FileDataset {
	return &FileDataset{Path: path}
}

func (d *FileDataset) Name() string {
	if d.NameHint != "" {
		return d.NameHint
	}
	return filepath.Base(d.Path)
}

func (d *FileDataset) Len(ctx context.Context) (int, error) {
	format, err := detectFormat(d.Path)
	if err != nil {
		return 0, err
	}

	switch format {
	case "json":
		samples, err := loadJSONSamples(d.Path)
		if err != nil {
			return 0, err
		}
		return len(samples), nil
	case "jsonl":
		return countJSONLLines(ctx, d.Path)
	default:
		return 0, errors.New("dataset: unsupported format")
	}
}

func (d *FileDataset) Samples(ctx context.Context) (<-chan core.Sample, <-chan error) {
	sampleCh := make(chan core.Sample)
	errCh := make(chan error, 1)

	go func() {
		defer close(sampleCh)
		defer close(errCh)

		format, err := detectFormat(d.Path)
		if err != nil {
			errCh <- err
			return
		}

		switch format {
		case "json":
			samples, err := loadJSONSamples(d.Path)
			if err != nil {
				errCh <- err
				return
			}
			for _, sample := range samples {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case sampleCh <- sample:
				}
			}
		case "jsonl":
			err = streamJSONL(ctx, d.Path, sampleCh)
			if err != nil {
				errCh <- err
			}
		default:
			errCh <- errors.New("dataset: unsupported format")
		}
	}()

	return sampleCh, errCh
}

func detectFormat(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jsonl":
		return "jsonl", nil
	case ".json":
		return "json", nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(string(b)) == "" {
			continue
		}
		if b == '[' {
			return "json", nil
		}
		if b == '{' {
			return "", errors.New("dataset: JSON object is not supported, use array or JSONL")
		}
		return "", errors.New("dataset: unsupported format")
	}
}

func loadJSONSamples(path string) ([]core.Sample, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var samples []core.Sample
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&samples); err != nil {
		return nil, err
	}
	return samples, nil
}

func streamJSONL(ctx context.Context, path string, out chan<- core.Sample) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Bytes()
		var sample core.Sample
		if err := json.Unmarshal(line, &sample); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- sample:
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func countJSONLLines(ctx context.Context, path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	count := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

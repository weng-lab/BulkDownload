package core

import (
	"fmt"
	"io"
	"os"
)

const copyBufferSize = 32 * 1024

type progressReporter struct {
	total       int64
	copied      int64
	lastPercent int
	onUpdate    func(int)
}

func newProgressReporter(total int64, onUpdate func(int)) *progressReporter {
	return &progressReporter{
		total:       total,
		lastPercent: 0,
		onUpdate:    onUpdate,
	}
}

func (p *progressReporter) Add(n int) {
	if p == nil || p.onUpdate == nil || p.total <= 0 || n <= 0 {
		return
	}

	p.copied += int64(n)
	percent := int((p.copied * 100) / p.total)
	if percent >= 100 {
		percent = 99
	}
	if percent <= p.lastPercent {
		return
	}

	p.lastPercent = percent
	p.onUpdate(percent)
}

func totalFileSize(paths []string) (int64, error) {
	var total int64
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return 0, fmt.Errorf("stat %s: %w", path, err)
		}
		total += info.Size()
	}

	return total, nil
}

func copyWithProgress(dst io.Writer, src io.Reader, reporter *progressReporter) error {
	buf := make([]byte, copyBufferSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			reporter.Add(n)
		}
		if err == nil {
			continue
		}
		if err == io.EOF {
			return nil
		}
		return err
	}
}

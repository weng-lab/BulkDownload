package artifacts

import (
	"context"
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

type progressWriter struct {
	dst      io.Writer
	reporter *progressReporter
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
	if percent > 100 {
		percent = 100
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

func (w progressWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	w.reporter.Add(n)
	return n, err
}

func copyWithProgress(dst io.Writer, src io.Reader, reporter *progressReporter) error {
	return copyWithProgressContext(context.Background(), dst, src, reporter)
}

func copyWithProgressContext(ctx context.Context, dst io.Writer, src io.Reader, reporter *progressReporter) error {
	buf := make([]byte, copyBufferSize)
	writer := progressWriter{dst: dst, reporter: reporter}

	for {
		if err := checkContext(ctx); err != nil {
			return err
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			if err := checkContext(ctx); err != nil {
				return err
			}

			written, writeErr := writer.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
		}

		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

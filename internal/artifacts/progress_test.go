package artifacts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProgressReporter_ReachesOneHundred(t *testing.T) {
	t.Parallel()

	var got []int
	reporter := newProgressReporter(10, func(progress int) {
		got = append(got, progress)
	})

	reporter.Add(3)
	reporter.Add(3)
	reporter.Add(10)

	want := []int{30, 60, 100}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("progress updates mismatch (-want +got):\n%s", diff)
	}
}

func TestCopyWithProgress_CopiesBytesAndReportsProgress(t *testing.T) {
	t.Parallel()

	src := strings.NewReader("abcdefghij")
	var dst bytes.Buffer
	var got []int
	reporter := newProgressReporter(10, func(progress int) {
		got = append(got, progress)
	})

	if err := copyWithProgress(&dst, src, reporter); err != nil {
		t.Fatalf("copyWithProgress() error = %v", err)
	}
	if diff := cmp.Diff("abcdefghij", dst.String()); diff != "" {
		t.Errorf("copied contents mismatch (-want +got):\n%s", diff)
	}

	want := []int{100}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("progress updates mismatch (-want +got):\n%s", diff)
	}
}

func TestCopyWithProgressContext_CancelsBeforeRead(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var dst bytes.Buffer
	reporter := newProgressReporter(10, nil)

	err := copyWithProgressContext(ctx, &dst, strings.NewReader("abcdefghij"), reporter)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copyWithProgressContext() error = %v, want context.Canceled", err)
	}
	if diff := cmp.Diff("", dst.String()); diff != "" {
		t.Errorf("copied contents mismatch (-want +got):\n%s", diff)
	}
}

func TestCopyWithProgressContext_CancelsBeforeWrite(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	reader := &blockingReader{
		chunks: [][]byte{[]byte("abc")},
		afterRead: func() {
			cancel()
		},
	}
	writer := &recordingWriter{}
	reporter := newProgressReporter(3, nil)

	err := copyWithProgressContext(ctx, writer, reader, reporter)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copyWithProgressContext() error = %v, want context.Canceled", err)
	}
	if diff := cmp.Diff(0, writer.writes()); diff != "" {
		t.Errorf("write count mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", writer.String()); diff != "" {
		t.Errorf("written contents mismatch (-want +got):\n%s", diff)
	}
}

type blockingReader struct {
	chunks    [][]byte
	index     int
	afterRead func()
}

func (r *blockingReader) Read(p []byte) (int, error) {
	if r.index >= len(r.chunks) {
		return 0, io.EOF
	}

	n := copy(p, r.chunks[r.index])
	r.index++
	if r.afterRead != nil {
		r.afterRead()
	}
	return n, nil
}

type recordingWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
	n   int
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.n++
	return w.buf.Write(p)
}

func (w *recordingWriter) writes() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.n
}

func (w *recordingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.String()
}

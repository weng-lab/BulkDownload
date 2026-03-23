package artifacts

import (
	"bytes"
	"strings"
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

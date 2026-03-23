package core

import (
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

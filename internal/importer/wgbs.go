package metadata

import (
	"database/sql"
	"fmt"
	"strings"
)

type wgbsSampleRow struct {
	Site     string
	Sex      string
	Status   string
	SampleID string
}

func validateWGBSHeaders(headers []string) error {
	want := []string{"Site", "Sex", "Status", "sample_id", "file name", "file type", "url"}
	if len(headers) != len(want) {
		return fmt.Errorf("read WGBS TSV header: expected %d columns, got %d", len(want), len(headers))
	}

	for i := range want {
		if headers[i] != want[i] {
			return fmt.Errorf("read WGBS TSV header: expected column %d to be %q, got %q", i+1, want[i], headers[i])
		}
	}

	return nil
}

func parseWGBSRow(record []string) (string, wgbsSampleRow, error) {
	if len(record) != 7 {
		return "", wgbsSampleRow{}, fmt.Errorf("expected 7 columns, got %d", len(record))
	}

	row := wgbsSampleRow{
		Site:     strings.TrimSpace(record[0]),
		Sex:      strings.TrimSpace(record[1]),
		Status:   strings.TrimSpace(record[2]),
		SampleID: strings.TrimSpace(record[3]),
	}

	if row.Site == "" && row.Sex == "" && row.Status == "" && row.SampleID == "" {
		return "", wgbsSampleRow{}, ErrSkipRow
	}

	if row.SampleID == "" {
		return "", wgbsSampleRow{}, fmt.Errorf("sample_id is required")
	}

	return row.SampleID, row, nil
}

func insertWGBSRow(tx *sql.Tx, row wgbsSampleRow) error {
	_, err := tx.Exec(
		`INSERT INTO samples_wgbs (sample_id, site, sex, status) VALUES (?, ?, ?, ?)`,
		row.SampleID,
		row.Site,
		row.Sex,
		row.Status,
	)
	if err != nil {
		return fmt.Errorf("insert WGBS sample %q: %w", row.SampleID, err)
	}

	return nil
}

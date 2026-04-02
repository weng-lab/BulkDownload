package metadata

import (
	"database/sql"
	"fmt"
	"strings"
)

type rnaSampleRow struct {
	Site     string
	Sex      string
	Status   string
	SampleID string
}

func validateRNAHeaders(headers []string) error {
	want := []string{"Site", "Sex", "Status", "sample_id", "file name", "file type", "url"}
	if len(headers) != len(want) {
		return fmt.Errorf("read RNA TSV header: expected %d columns, got %d", len(want), len(headers))
	}

	for i := range want {
		if headers[i] != want[i] {
			return fmt.Errorf("read RNA TSV header: expected column %d to be %q, got %q", i+1, want[i], headers[i])
		}
	}

	return nil
}

func parseRNARow(record []string) (string, rnaSampleRow, error) {
	if len(record) != 7 {
		return "", rnaSampleRow{}, fmt.Errorf("expected 7 columns, got %d", len(record))
	}

	row := rnaSampleRow{
		Site:     strings.TrimSpace(record[0]),
		Sex:      strings.TrimSpace(record[1]),
		Status:   strings.TrimSpace(record[2]),
		SampleID: strings.TrimSpace(record[3]),
	}

	if row.Site == "" && row.Sex == "" && row.Status == "" && row.SampleID == "" {
		return "", rnaSampleRow{}, ErrSkipRow
	}

	if row.SampleID == "" {
		return "", rnaSampleRow{}, fmt.Errorf("sample_id is required")
	}

	return row.SampleID, row, nil
}

func insertRNARow(tx *sql.Tx, row rnaSampleRow) error {
	_, err := tx.Exec(
		`INSERT INTO samples_rna (sample_id, site, sex, status) VALUES (?, ?, ?, ?)`,
		row.SampleID,
		row.Site,
		row.Sex,
		row.Status,
	)
	if err != nil {
		return fmt.Errorf("insert RNA sample %q: %w", row.SampleID, err)
	}

	return nil
}

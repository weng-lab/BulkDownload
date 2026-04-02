package metadata

import (
	"database/sql"
	"fmt"
	"strings"
)

type atacSampleRow struct {
	Site     string
	Status   string
	Sex      string
	Protocol string
	SampleID string
}

func validateATACHeaders(headers []string) error {
	want := []string{"Site", "Status", "Sex", "Protocol", "sample_id", "file name", "file type", "url"}
	if len(headers) != len(want) {
		return fmt.Errorf("read ATAC TSV header: expected %d columns, got %d", len(want), len(headers))
	}

	for i := range want {
		if headers[i] != want[i] {
			return fmt.Errorf("read ATAC TSV header: expected column %d to be %q, got %q", i+1, want[i], headers[i])
		}
	}

	return nil
}

func parseATACRow(record []string) (string, atacSampleRow, error) {
	if len(record) != 8 {
		return "", atacSampleRow{}, fmt.Errorf("expected 8 columns, got %d", len(record))
	}

	row := atacSampleRow{
		Site:     strings.TrimSpace(record[0]),
		Status:   strings.TrimSpace(record[1]),
		Sex:      strings.TrimSpace(record[2]),
		Protocol: strings.TrimSpace(record[3]),
		SampleID: strings.TrimSpace(record[4]),
	}

	if row.Site == "" && row.Status == "" && row.Sex == "" && row.Protocol == "" && row.SampleID == "" {
		return "", atacSampleRow{}, ErrSkipRow
	}

	if row.SampleID == "" {
		return "", atacSampleRow{}, fmt.Errorf("sample_id is required")
	}

	return row.SampleID, row, nil
}

func insertATACRow(tx *sql.Tx, row atacSampleRow) error {
	_, err := tx.Exec(
		`INSERT INTO samples_atac (sample_id, site, status, sex, protocol) VALUES (?, ?, ?, ?, ?)`,
		row.SampleID,
		row.Site,
		row.Status,
		row.Sex,
		row.Protocol,
	)
	if err != nil {
		return fmt.Errorf("insert ATAC sample %q: %w", row.SampleID, err)
	}

	return nil
}

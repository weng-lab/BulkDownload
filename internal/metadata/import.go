package metadata

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	DefaultOutputPath = "data/metadata.db"
	atacTSVPath       = "tsv/atac.tsv"
	atacTableName     = "samples_atac"
)

type atacSampleRow struct {
	Site     string
	Status   string
	Sex      string
	Protocol string
	SampleID string
}

func BuildDatabase(outPath string) error {
	if outPath == "" {
		outPath = DefaultOutputPath
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.Remove(outPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing database: %w", err)
	}

	db, err := sql.Open("sqlite", outPath)
	if err != nil {
		return fmt.Errorf("open sqlite database: %w", err)
	}
	defer db.Close()

	if err := createATACTable(db); err != nil {
		return err
	}

	if err := importATACSamples(db, atacTSVPath); err != nil {
		return err
	}

	return nil
}

func createATACTable(db *sql.DB) error {
	const query = `
	CREATE TABLE samples_atac (
		sample_id TEXT PRIMARY KEY,
		site TEXT NOT NULL,
		status TEXT NOT NULL,
		sex TEXT NOT NULL,
		protocol TEXT NOT NULL
	)`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("create %s table: %w", atacTableName, err)
	}

	return nil
}

func importATACSamples(db *sql.DB, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open ATAC TSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'

	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read ATAC TSV header: %w", err)
	}

	if err := validateATACHeaders(headers); err != nil {
		return err
	}

	rowsBySampleID := make(map[string]atacSampleRow)
	line := 1
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read ATAC TSV row %d: %w", line+1, err)
		}

		line++
		if len(record) != len(headers) {
			return fmt.Errorf("read ATAC TSV row %d: expected %d columns, got %d", line, len(headers), len(record))
		}

		row := atacSampleRow{
			Site:     strings.TrimSpace(record[0]),
			Status:   strings.TrimSpace(record[1]),
			Sex:      strings.TrimSpace(record[2]),
			Protocol: strings.TrimSpace(record[3]),
			SampleID: strings.TrimSpace(record[4]),
		}

		isBlankRow := row.Site == "" && row.Status == "" && row.Sex == "" && row.Protocol == "" && row.SampleID == ""
		if isBlankRow {
			continue
		}

		if row.SampleID == "" {
			return fmt.Errorf("read ATAC TSV row %d: sample_id is required", line)
		}

		if _, ok := rowsBySampleID[row.SampleID]; ok {
			continue
		}

		rowsBySampleID[row.SampleID] = row
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin ATAC import transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO samples_atac (sample_id, site, status, sex, protocol) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare ATAC insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rowsBySampleID {
		if _, err := stmt.Exec(row.SampleID, row.Site, row.Status, row.Sex, row.Protocol); err != nil {
			return fmt.Errorf("insert ATAC sample %q: %w", row.SampleID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ATAC import: %w", err)
	}

	return nil
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

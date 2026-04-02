package metadata

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type fileRow struct {
	Ome        string
	SampleID   string
	Filename   string
	FileType   string
	Size       int64
	OpenAccess bool
}

var supportedFileOmes = map[string]string{
	"ATAC_SEQ": "atac",
	"RNA_SEQ":  "rna",
	"WGBS":     "wgbs",
}

func importFiles(db *sql.DB, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'

	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read %s header: %w", path, err)
	}

	if err := validateFileHeaders(headers); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin %s import transaction: %w", path, err)
	}
	defer tx.Rollback()

	line := 1
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read %s row %d: %w", path, line+1, err)
		}

		line++

		row, err := parseFileRow(record)
		if err != nil {
			if err == ErrSkipRow {
				continue
			}
			return fmt.Errorf("parse %s row %d: %w", path, line, err)
		}

		if err := insertFileRow(tx, row); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s import: %w", path, err)
	}

	return nil
}

func validateFileHeaders(headers []string) error {
	want := []string{"sample_id", "filename", "file_type", "size", "file_ome", "open_access"}
	if len(headers) != len(want) {
		return fmt.Errorf("read files TSV header: expected %d columns, got %d", len(want), len(headers))
	}

	for i := range want {
		if headers[i] != want[i] {
			return fmt.Errorf("read files TSV header: expected column %d to be %q, got %q", i+1, want[i], headers[i])
		}
	}

	return nil
}

func parseFileRow(record []string) (fileRow, error) {
	if len(record) != 6 {
		return fileRow{}, fmt.Errorf("expected 6 columns, got %d", len(record))
	}

	sampleID := strings.TrimSpace(record[0])
	filename := strings.TrimSpace(record[1])
	fileType := strings.TrimSpace(record[2])
	sizeValue := strings.TrimSpace(record[3])
	fileOme := strings.TrimSpace(record[4])
	openAccessValue := strings.TrimSpace(record[5])

	if sampleID == "" && filename == "" && fileType == "" && sizeValue == "" && fileOme == "" && openAccessValue == "" {
		return fileRow{}, ErrSkipRow
	}

	ome, ok := supportedFileOmes[fileOme]
	if !ok {
		return fileRow{}, ErrSkipRow
	}

	if sampleID == "" {
		return fileRow{}, fmt.Errorf("sample_id is required")
	}

	if filename == "" {
		return fileRow{}, fmt.Errorf("filename is required")
	}

	if fileType == "" {
		return fileRow{}, fmt.Errorf("file_type is required")
	}

	size, err := strconv.ParseInt(sizeValue, 10, 64)
	if err != nil {
		return fileRow{}, fmt.Errorf("parse size %q: %w", sizeValue, err)
	}

	openAccess, err := parseOpenAccess(openAccessValue)
	if err != nil {
		return fileRow{}, err
	}

	return fileRow{
		Ome:        ome,
		SampleID:   sampleID,
		Filename:   filename,
		FileType:   fileType,
		Size:       size,
		OpenAccess: openAccess,
	}, nil
}

func parseOpenAccess(value string) (bool, error) {
	switch value {
	case "True":
		return true, nil
	case "False":
		return false, nil
	default:
		return false, fmt.Errorf("parse open_access %q: invalid boolean", value)
	}
}

func insertFileRow(tx *sql.Tx, row fileRow) error {
	_, err := tx.Exec(
		`INSERT INTO files (ome, sample_id, filename, file_type, size, open_access) VALUES (?, ?, ?, ?, ?, ?)`,
		row.Ome,
		row.SampleID,
		row.Filename,
		row.FileType,
		row.Size,
		row.OpenAccess,
	)
	if err != nil {
		return fmt.Errorf("insert file %q for sample %q: %w", row.Filename, row.SampleID, err)
	}

	return nil
}

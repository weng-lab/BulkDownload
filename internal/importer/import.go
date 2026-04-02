package metadata

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

const (
	DefaultOutputPath    = "data/metadata.db"
	defaultMigrationsDir = "db/migrations"
	atacTSVPath          = "tsv/atac.tsv"
	atacTableName        = "samples_atac"
	filesTSVPath         = "tsv/mohd_phase_0_download_files.tsv"
	rnaTSVPath           = "tsv/rna.tsv"
	rnaTableName         = "samples_rna"
	wgbsTSVPath          = "tsv/wgbs.tsv"
	wgbsTableName        = "samples_wgbs"
)

var ErrSkipRow = errors.New("skip row")

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

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, defaultMigrationsDir); err != nil {
		return fmt.Errorf("apply goose migrations: %w", err)
	}

	if err := importRows(db, atacTSVPath, validateATACHeaders, parseATACRow, insertATACRow); err != nil {
		return err
	}

	if err := importRows(db, rnaTSVPath, validateRNAHeaders, parseRNARow, insertRNARow); err != nil {
		return err
	}

	if err := importRows(db, wgbsTSVPath, validateWGBSHeaders, parseWGBSRow, insertWGBSRow); err != nil {
		return err
	}

	if err := importFiles(db, filesTSVPath); err != nil {
		return err
	}

	return nil
}

func importRows[T any](
	db *sql.DB,
	path string,
	validateHeader func([]string) error,
	parseRow func([]string) (string, T, error),
	insertRow func(*sql.Tx, T) error,
) error {
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

	if err := validateHeader(headers); err != nil {
		return err
	}

	rowsBySampleID := make(map[string]T)
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

		sampleID, row, err := parseRow(record)
		if err != nil {
			if errors.Is(err, ErrSkipRow) {
				continue
			}
			return fmt.Errorf("parse %s row %d: %w", path, line, err)
		}

		if _, ok := rowsBySampleID[sampleID]; ok {
			continue
		}

		rowsBySampleID[sampleID] = row
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin %s import transaction: %w", path, err)
	}
	defer tx.Rollback()

	for _, row := range rowsBySampleID {
		if err := insertRow(tx, row); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s import: %w", path, err)
	}

	return nil
}

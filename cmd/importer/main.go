package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jair/bulkdownload/internal/importer"
)

func main() {
	outPath := flag.String("out", importer.DefaultOutputPath, "output SQLite database path")
	flag.Parse()

	if err := importer.BuildDatabase(*outPath); err != nil {
		fmt.Fprintf(os.Stderr, "import metadata: %v\n", err)
		os.Exit(1)
	}
}

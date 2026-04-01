package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jair/bulkdownload/internal/metadata"
)

func main() {
	outPath := flag.String("out", metadata.DefaultOutputPath, "output SQLite database path")
	flag.Parse()

	if err := metadata.BuildDatabase(*outPath); err != nil {
		fmt.Fprintf(os.Stderr, "import metadata: %v\n", err)
		os.Exit(1)
	}
}

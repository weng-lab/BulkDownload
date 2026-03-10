package main

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

func startCleanup(store *Store) {
	ticker := time.NewTicker(cleanupTick)

	go func() {
		for range ticker.C {
			log.Println("cleanup: attemping cleanup")

			expired := store.Expired()
			if len(expired) > 0 {
				log.Printf("cleanup: found %d expired jobs", len(expired))
			}
			for _, job := range expired {
				if job.Filename != "" {
					path := filepath.Join(outputDir, job.Filename)
					if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
						log.Printf("cleanup: failed to remove %s: %v", path, err)
					} else {
						log.Printf("cleanup: removed %s", path)
					}
				}

				log.Printf("cleanup: deleting job %s", job.ID)
				store.Delete(job.ID)
			}
		}
	}()
}

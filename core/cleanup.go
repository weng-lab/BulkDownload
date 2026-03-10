package core

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

func StartCleanup(store *Store) {
	ticker := time.NewTicker(CleanupTick)

	go func() {
		for range ticker.C {
			log.Printf("cleanup: sweep started (interval %s)", CleanupTick)
			expired := store.Expired()
			if len(expired) > 0 {
				log.Printf("cleanup: found %d expired jobs", len(expired))
			}

			for _, job := range expired {
				if job.Filename != "" {
					path := filepath.Join(OutputDir, job.Filename)
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

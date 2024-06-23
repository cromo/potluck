package index

import (
	"context"
	"crypto/sha256"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cromo/potluck/persistence"
)

const (
	indexUpdated = "INDEX_UPDATED"
)

type FileWalker struct {
	Dir    string
	Period time.Duration
}

func (walker *FileWalker) Index(ctx context.Context, db *persistence.HashDB, status chan<- string) {
	indexStart := time.Now()
	walk(ctx, walker.Dir, "", db)
	if err := db.DeleteFilesWithLastCheckTimestampBefore(indexStart); err != nil {
		log.Fatal("Failed to remove old hash entries", err)
	}
	log.Printf("Initial index took %s", time.Since(indexStart))
	status <- indexUpdated

	if walker.Period < 0 {
		return
	}

	ticker := time.NewTicker(walker.Period)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			indexStart = time.Now()
			walk(ctx, walker.Dir, "", db)
			if err := db.DeleteFilesWithLastCheckTimestampBefore(indexStart); err != nil {
				log.Fatal("Failed to remove old hash entries", err)
			}
			log.Printf("Subsequent index took %s", time.Since(indexStart))
			status <- indexUpdated
		}
	}
}

func walk(ctx context.Context, baseDir string, subDir string, db *persistence.HashDB) {
	dir := filepath.Join(baseDir, subDir)
	d, err := os.Open(dir)
	if err != nil {
		log.Fatal("Failed to open directory")
	}
	defer d.Close()

	files, err := d.ReadDir(-1)
	if err != nil {
		log.Fatal("Failed to read directory")
	}
	for _, file := range files {
		select {
		case <-ctx.Done():
			return
		default:
			fullPath := filepath.Join(dir, file.Name())
			if file.IsDir() {
				walk(ctx, baseDir, filepath.Join(subDir, file.Name()), db)
			} else {
				err := db.UpsertFileHash(filepath.Join(subDir, file.Name()), hashFile(fullPath))
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func hashFile(path string) []byte {
	in, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	hasher := sha256.New()
	io.Copy(hasher, in)
	return hasher.Sum(nil)
}

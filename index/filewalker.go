package index

import (
	"context"
	"crypto/sha256"
	"log"
	"os"
	"path/filepath"

	"github.com/cromo/potluck/persistence"
	_ "github.com/glebarez/go-sqlite"
)

const (
	indexUpdated = "INDEX_UPDATED"
)

type FileWalker struct {
	Dir string
}

func (walker *FileWalker) Index(ctx context.Context, db *persistence.HashDB, status chan<- string) {
	walk(walker.Dir, "", db)
	status <- indexUpdated
	<-ctx.Done()
}

func walk(baseDir string, subDir string, db *persistence.HashDB) {
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
		fullPath := filepath.Join(dir, file.Name())
		if file.IsDir() {
			walk(baseDir, filepath.Join(subDir, file.Name()), db)
		} else {
			err := db.AddFileHash(filepath.Join(subDir, file.Name()), hashFile(fullPath))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func hashFile(path string) []byte {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	hash := sha256.Sum256(content)
	return hash[:]
}

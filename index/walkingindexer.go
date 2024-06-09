package index

import (
	"crypto/sha256"
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

const (
	indexUpdated = "INDEX_UPDATED"
)

func FileWalker(dir string, db *sql.DB, status chan<- string, done <-chan struct{}) {
	walk(dir, "", db)
	status <- indexUpdated
	<-done
}

func walk(baseDir string, subDir string, db *sql.DB) {
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
			_, err := db.Exec(`insert into content (path, hash) VALUES (?, ?)`, filepath.Join(subDir, file.Name()), hashFile(fullPath))
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

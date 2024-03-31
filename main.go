package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%s\n\n", workingDir)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`create table if not exists content (
		path text primary key not null,
		hash blob not null
	)`)
	if err != nil {
		log.Fatal(err)
	}

	q := db.QueryRow("select sqlite_version()")
	var version string
	q.Scan(&version)
	log.Println(version)

	walk(workingDir, "", db)

	var path string
	var hash []byte
	rows, err := db.Query(`select path, hash from content order by hash`)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		rows.Scan(&path, &hash)
		log.Printf("%s %s", hex.EncodeToString(hash), path)
	}
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

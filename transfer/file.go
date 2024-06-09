package transfer

import (
	"database/sql"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
)

func File(dir string, db *sql.DB, transferRequests <-chan Request, done <-chan struct{}) {
	for {
		select {
		case request := <-transferRequests:
			path := pathFromHash(db, request.Hash)
			copyFile(path, filepath.Join(dir, request.Hash+filepath.Ext(path)))
		case <-done:
			return
		}
	}
}

func pathFromHash(db *sql.DB, hash string) string {
	hashBin, err := hex.DecodeString(hash)
	if err != nil {
		log.Fatalf("Error decoding hash: %v\n", err)
	}
	result := db.QueryRow(`select path from content where hash = ?`, hashBin)
	var path string
	err = result.Scan(&path)
	if err == sql.ErrNoRows {
		log.Fatal(err)
	}
	return path
}

func copyFile(src, dst string) {
	out, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		log.Fatal(err)
	}
}

package transfer

import (
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/cromo/potluck/persistence"
)

func File(dir string, db *persistence.HashDB, transferRequests <-chan Request, done <-chan struct{}) {
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

func pathFromHash(db *persistence.HashDB, hash string) string {
	hashBin, err := hex.DecodeString(hash)
	if err != nil {
		log.Fatalf("Error decoding hash: %v\n", err)
	}
	path, err := db.GetPathForHash(hashBin)
	if err != nil {
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

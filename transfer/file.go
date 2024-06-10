package transfer

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/cromo/potluck/persistence"
)

func File(ctx context.Context, dir string, db *persistence.HashDB, transferRequests <-chan Request) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-transferRequests:
			path, err := db.GetPathForHashHexString(request.Hash)
			if err != nil {
				log.Fatalf("Error decoding hash: %v\n", err)
			}
			copyFile(path, filepath.Join(dir, request.Hash+filepath.Ext(path)))
		}
	}
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

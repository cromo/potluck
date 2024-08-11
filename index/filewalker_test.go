package index

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/cromo/potluck/persistence"
)

type file struct {
	filename string
	contents string
	hash     string
}

var helloFile = &file{
	filename: "hello.txt",
	contents: "Hello, world!",
	hash:     "315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3",
}
var testingFile = &file{
	filename: "testing.txt",
	contents: "Testing, testing,\n1, 2, 3",
	hash:     "0829082fe37cfb4db4e0d551ef5c09c54f8a0b19dc5c86038158a6b89ca9d53d",
}
var updatedTestingFile = &file{
	filename: "testing.txt",
	contents: "Make sure to wear your hard hat!",
	hash:     "a873217c3c0789042bb42991416ad491f1a01d4f09a329a49fe2438ba05ed014",
}

func setupTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp(".", "potluck-index-")
	if err != nil {
		t.Fatal("Unable to make temporary directory", err)
	}
	return dir
}

func writeTestFile(t *testing.T, dir string, file *file) {
	if err := os.WriteFile(filepath.Join(dir, file.filename), []byte(file.contents), 0666); err != nil {
		t.Fatal("Failed to write test file", file.filename, err)
	}
}

func setupTestDB(t *testing.T) *persistence.HashDB {
	db, err := persistence.CreateInMemory()
	if err != nil {
		t.Fatal("Could not set up database", err)
	}
	return db
}

func TestWalkerSignalsWalkCompleted(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)
	writeTestFile(t, dir, helloFile)
	writeTestFile(t, dir, testingFile)
	db := setupTestDB(t)
	c := make(chan string, 1)
	w := FileWalker{Dir: dir, Period: -1}

	w.Index(context.Background(), db, c)

	select {
	case <-c:
		// success
	default:
		t.Fatal("Walker did not send a message that it finished a scan")
	}
}

func TestWalkerAddsHashesToDatabase(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)
	writeTestFile(t, dir, helloFile)
	writeTestFile(t, dir, testingFile)
	db := setupTestDB(t)
	c := make(chan string, 1)
	w := FileWalker{Dir: dir, Period: -1}

	w.Index(context.Background(), db, c)

	files, err := db.ListAll(context.Background())
	if err != nil {
		t.Fatal("Unable to get files and hashes", err)
	}
	if len(files) != 2 {
		t.Fatal("Unexpected number of files returned", len(files))
	}
	hashes := make(map[string]bool)
	for _, f := range files {
		hashes[hex.EncodeToString(f.Hash)] = true
	}
	if _, ok := hashes[helloFile.hash]; !ok {
		t.Fatal("Hash for file was not found", helloFile.filename)
	}
	if _, ok := hashes[testingFile.hash]; !ok {
		t.Fatal("Hash for file was not found", testingFile.filename)
	}
}

func TestWalkerUpdatesHashesWhenContentsChange(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)
	writeTestFile(t, dir, helloFile)
	writeTestFile(t, dir, testingFile)
	db := setupTestDB(t)
	c := make(chan string, 2)
	w := FileWalker{Dir: dir, Period: -1}

	w.Index(context.Background(), db, c)
	writeTestFile(t, dir, updatedTestingFile)
	w.Index(context.Background(), db, c)

	files, err := db.ListAll(context.Background())
	if err != nil {
		t.Fatal("Unable to get files and hashes", err)
	}
	if len(files) != 2 {
		t.Fatal("Unexpected number of files returned", len(files))
	}
	hashes := make(map[string]bool)
	for _, f := range files {
		hashes[hex.EncodeToString(f.Hash)] = true
	}
	if _, ok := hashes[helloFile.hash]; !ok {
		t.Fatal("Hash for file was not found", helloFile.filename)
	}
	if _, ok := hashes[updatedTestingFile.hash]; !ok {
		t.Fatal("Hash for file was not found", updatedTestingFile.filename)
	}
}

func TestWalkerRemovesHashesWhenFilesDisappear(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)
	writeTestFile(t, dir, helloFile)
	writeTestFile(t, dir, testingFile)
	db := setupTestDB(t)
	c := make(chan string, 2)
	w := FileWalker{Dir: dir, Period: -1}

	w.Index(context.Background(), db, c)
	if err := os.Remove(filepath.Join(dir, testingFile.filename)); err != nil {
		t.Fatal("Unable to remove file", err)
	}
	w.Index(context.Background(), db, c)

	files, err := db.ListAll(context.Background())
	if err != nil {
		t.Fatal("Unable to get files and hashes", err)
	}
	if len(files) != 1 {
		t.Fatal("Unexpected number of files returned", len(files))
	}
	hashes := make(map[string]bool)
	for _, f := range files {
		hashes[hex.EncodeToString(f.Hash)] = true
	}
	if _, ok := hashes[helloFile.hash]; !ok {
		t.Fatal("Hash for file was not found", helloFile.filename)
	}
	if _, ok := hashes[testingFile.hash]; ok {
		t.Fatal("Hash for deleted file was found", testingFile.filename)
	}
}

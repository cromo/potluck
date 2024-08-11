package persistence

import (
	"database/sql"
	"encoding/hex"
	"log"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

type HashDB struct{ db *sql.DB }
type HashSet struct {
	Path              string
	Hash              []byte
	LastHashTimestamp string
}

func CreateInMemory() (*HashDB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		create table if not exists content (
			path text primary key not null,
			hash blob not null,
			lastHashTimestamp text not null,
			lastCheckTimestamp text not null
		)`)
	if err != nil {
		return nil, err
	}
	return &HashDB{db: db}, nil
}

func (db *HashDB) GetFileCount() (int, error) {
	result := db.db.QueryRow(`select count(*) from content`)
	var count int
	err := result.Scan(&count)
	return count, err
}

func (db *HashDB) UpsertFileHash(path string, hash []byte) error {
	now := time.Now().UTC()
	_, err := db.db.Exec(`
		insert into content (
			path,
			hash,
			lastHashTimestamp,
			lastCheckTimestamp
		) values (
			@path,
			@hash,
			@lastHashTimestamp,
			@lastCheckTimestamp
		)	on conflict do update set
			hash = @hash,
			lastHashTimestamp = @lastHashTimestamp,
			lastCheckTimestamp = @lastCheckTimestamp`,
		sql.Named("path", path),
		sql.Named("hash", hash),
		sql.Named("lastHashTimestamp", now),
		sql.Named("lastCheckTimestamp", now))
	return err
}

func (db *HashDB) GetLastCheckTimestamp(path string) (time.Time, error) {
	result := db.db.QueryRow(
		`select lastCheckTimestamp from content where path = @path`,
		sql.Named("path", path))
	var timestamp string
	err := result.Scan(&timestamp)
	if err == sql.ErrNoRows {
		return time.Time{}, err
	}
	return time.Parse("2006-01-02 03:04:05", timestamp)
}

func (db *HashDB) DeleteFilesWithLastCheckTimestampBefore(cutoffTime time.Time) error {
	_, err := db.db.Exec(`
		delete from content
		where lastCheckTimestamp < @cutoffTimestamp`,
		sql.Named("cutoffTimestamp", cutoffTime.UTC()))
	return err
}

func (db *HashDB) GetPathForHash(hash []byte) (string, error) {
	result := db.db.QueryRow(
		`select path from content where hash = @hash`,
		sql.Named("hash", hash))
	var path string
	err := result.Scan(&path)
	if err == sql.ErrNoRows {
		return "", err
	}
	return path, nil
}

func (db *HashDB) GetPathForHashHexString(hash string) (string, error) {
	hashBin, err := hex.DecodeString(hash)
	if err != nil {
		return "", err
	}
	return db.GetPathForHash(hashBin)
}

func (db *HashDB) HaveHash(hash []byte) bool {
	result := db.db.QueryRow(
		`select path from content where hash = @hash`,
		sql.Named("hash", hash))
	var path string
	err := result.Scan(&path)
	return err != sql.ErrNoRows
}

func (db *HashDB) HaveHashHexString(hash string) bool {
	hashBin, err := hex.DecodeString(hash)
	if err != nil {
		log.Printf("Error decoding hash: %v\n", err)
		return false
	}
	return db.HaveHash(hashBin)
}

func (db *HashDB) ListAll() ([]HashSet, error) {
	var hashSet []HashSet
	rows, err := db.db.Query(`select path, hash, lastHashTimestamp from content order by hash`)

	if err != nil {
		return hashSet, err
	}
	var path string
	var hash []byte
	var lastHashTimestamp string
	for rows.Next() {
		rows.Scan(&path, &hash, &lastHashTimestamp)
		hashSet = append(hashSet, HashSet{Path: path, Hash: hash, LastHashTimestamp: lastHashTimestamp})
	}
	return hashSet, nil
}

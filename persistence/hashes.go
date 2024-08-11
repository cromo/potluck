package persistence

import (
	"context"
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

func (db *HashDB) GetFileCount(ctx context.Context) (int, error) {
	result := db.db.QueryRowContext(ctx, `select count(*) from content`)
	var count int
	err := result.Scan(&count)
	return count, err
}

func (db *HashDB) UpsertFileHash(ctx context.Context, path string, hash []byte) error {
	now := time.Now().UTC()
	_, err := db.db.ExecContext(
		ctx,
		`insert into content (
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

func (db *HashDB) GetLastCheckTimestamp(ctx context.Context, path string) (time.Time, error) {
	result := db.db.QueryRowContext(
		ctx,
		`select lastCheckTimestamp from content where path = @path`,
		sql.Named("path", path))
	var timestamp string
	err := result.Scan(&timestamp)
	if err == sql.ErrNoRows {
		return time.Time{}, err
	}
	return time.Parse("2006-01-02 03:04:05", timestamp)
}

func (db *HashDB) DeleteFilesWithLastCheckTimestampBefore(ctx context.Context, cutoffTime time.Time) error {
	_, err := db.db.ExecContext(
		ctx,
		`delete from content
		where lastCheckTimestamp < @cutoffTimestamp`,
		sql.Named("cutoffTimestamp", cutoffTime.UTC()))
	return err
}

func (db *HashDB) GetPathForHash(ctx context.Context, hash []byte) (string, error) {
	result := db.db.QueryRowContext(
		ctx,
		`select path from content where hash = @hash`,
		sql.Named("hash", hash))
	var path string
	err := result.Scan(&path)
	if err == sql.ErrNoRows {
		return "", err
	}
	return path, nil
}

func (db *HashDB) GetPathForHashHexString(ctx context.Context, hash string) (string, error) {
	hashBin, err := hex.DecodeString(hash)
	if err != nil {
		return "", err
	}
	return db.GetPathForHash(ctx, hashBin)
}

func (db *HashDB) HaveHash(ctx context.Context, hash []byte) bool {
	result := db.db.QueryRowContext(
		ctx,
		`select path from content where hash = @hash`,
		sql.Named("hash", hash))
	var path string
	err := result.Scan(&path)
	return err != sql.ErrNoRows
}

func (db *HashDB) HaveHashHexString(ctx context.Context, hash string) bool {
	hashBin, err := hex.DecodeString(hash)
	if err != nil {
		log.Printf("Error decoding hash: %v\n", err)
		return false
	}
	return db.HaveHash(ctx, hashBin)
}

func (db *HashDB) ListAll(ctx context.Context) ([]HashSet, error) {
	var hashSet []HashSet
	rows, err := db.db.QueryContext(ctx, `select path, hash, lastHashTimestamp from content order by hash`)

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

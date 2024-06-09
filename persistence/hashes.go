package persistence

import (
	"database/sql"
)

type HashDB struct{ db *sql.DB }
type HashSet struct {
	Path string
	Hash []byte
}

func CreateInMemory() (*HashDB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table if not exists content (
		path text primary key not null,
		hash blob not null
	)`)
	if err != nil {
		return nil, err
	}
	return &HashDB{db: db}, nil
}

func (db *HashDB) AddFileHash(path string, hash []byte) error {
	_, err := db.db.Exec(`insert into content (path, hash) VALUES (?, ?)`, path, hash)
	return err
}

func (db *HashDB) GetPathForHash(hash []byte) (string, error) {
	result := db.db.QueryRow(`select path from content where hash = ?`, hash)
	var path string
	err := result.Scan(&path)
	if err == sql.ErrNoRows {
		return "", err
	}
	return path, nil
}

func (db *HashDB) HaveHash(hash []byte) bool {
	result := db.db.QueryRow(`select path from content where hash = ?`, hash)
	var path string
	err := result.Scan(&path)
	return err != sql.ErrNoRows
}

func (db *HashDB) ListAll() ([]HashSet, error) {
	var hashSet []HashSet
	rows, err := db.db.Query(`select path, hash from content order by hash`)

	if err != nil {
		return hashSet, err
	}
	var path string
	var hash []byte
	for rows.Next() {
		rows.Scan(&path, &hash)
		hashSet = append(hashSet, HashSet{Path: path, Hash: hash})
	}
	return hashSet, nil
}

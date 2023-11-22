/*
Copyright 2023 Milan Suk

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this db except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"database/sql"
	"fmt"
)

type DiskDb struct {
	dbs  *Disk
	path string

	db *sql.DB
	tx *sql.Tx

	cache          []*DiskDbCache
	lastWriteTicks int64
}

func NewDiskDb(dbs *Disk, sqlDb *sql.DB) *DiskDb {
	var db DiskDb
	db.dbs = dbs
	db.db = sqlDb

	return &db
}

func (db *DiskDb) Destroy() {

	db.db.Exec("PRAGMA wal_checkpoint(full);")

	db.Commit()

	err := db.db.Close()
	if err != nil {
		fmt.Printf("db(%s).Destroy() failed: %v\n", db.path, err)
	}
}

func (db *DiskDb) Begin() (*sql.Tx, error) {
	if db.tx == nil {
		var err error
		db.tx, err = db.db.Begin()
		if err != nil {
			return nil, fmt.Errorf("Begin(%s) failed: %w", db.path, err)
		}
	}
	return db.tx, nil
}

func (db *DiskDb) Commit() error {
	if db.tx == nil {
		return nil
	}

	err := db.tx.Commit()
	db.tx = nil

	//reset queries
	db.cache = nil

	db.dbs.ResetTick()
	return err
}
func (db *DiskDb) Rollback() error {
	if db.tx == nil {
		return nil
	}

	err := db.tx.Rollback()
	db.tx = nil

	//reset queries
	db.cache = nil

	db.dbs.ResetTick()
	return err
}

func (db *DiskDb) FindCache(query_hash int64) *DiskDbCache {

	//find
	for _, it := range db.cache {
		if it.query_hash == query_hash {
			return it
		}
	}
	return nil
}

func (db *DiskDb) AddCache(query string) (*DiskDbCache, error) {

	//find
	for _, it := range db.cache {
		if it.query == query {
			return it, nil
		}
	}

	//add
	cache, err := NewDiskDbCache(query, db.db)
	if err != nil {
		return nil, fmt.Errorf("NewDbCache(%s) failed: %w", db.path, err)
	}

	db.cache = append(db.cache, cache)
	return cache, nil
}

func (db *DiskDb) Write(query string, params ...any) (sql.Result, error) {

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query(%s) failed: %w", query, err)
	}

	db.lastWriteTicks = int64(OsTicks())

	return res, nil
}

func (db *DiskDb) Tick() {

}

func (db *DiskDb) Vacuum() {
	db.db.Exec("VACUUM;")
}

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
	"strings"
	"sync"
)

type DiskDbIndexColumn struct {
	Name string
	Type string
}
type DiskDbIndexTable struct {
	Name    string
	Columns []*DiskDbIndexColumn
}

func (tb *DiskDbIndexTable) ListOfColumnNames() string {
	str := ""

	for _, c := range tb.Columns {
		str += c.Name + ", "
	}
	str, _ = strings.CutSuffix(str, ", ")

	return str
}

func (tb *DiskDbIndexTable) ListOfColumnValues() string {
	str := ""

	for range tb.Columns {
		str += `"", `
	}
	str, _ = strings.CutSuffix(str, ", ")

	return str
}

type DiskDb struct {
	disk *Disk

	lock sync.Mutex

	path     string
	inMemory bool
	db       *sql.DB
	//tx *sql.Tx

	lastWriteTicks int64
	lastReadTicks  int64

	written bool

	file_time     int64
	file_time_wal int64
	file_time_shm int64

	last_file_tick int64
}

func NewDiskDb(path string, inMemory bool, disk *Disk) (*DiskDb, error) {
	var db DiskDb
	db.path = path
	db.disk = disk
	db.inMemory = inMemory

	if inMemory {
		var err error
		db.db, err = sql.Open("sqlite3", "file:"+path+"?mode=memory&cache=shared") //sqlite3 -> sqlite3_skyalt
		if err != nil {
			return nil, fmt.Errorf("sql.Open(%s) in memory failed: %w", path, err)
		}
	} else {
		var err error
		db.db, err = sql.Open("sqlite3", "file:"+path+"?&_journal_mode=WAL") //sqlite3 -> sqlite3_skyalt
		if err != nil {
			return nil, fmt.Errorf("sql.Open(%s) from file failed: %w", path, err)
		}
	}

	db.HasFileChanged() //set 'file_time'

	return &db, nil
}

func (db *DiskDb) Destroy() {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db.Exec("PRAGMA wal_checkpoint(full);")

	//db.Commit()

	err := db.db.Close()
	if err != nil {
		fmt.Printf("db(%s).Destroy() failed: %v\n", db.path, err)
	}
}

func (db *DiskDb) hasFileChanged() bool {
	tm := OsFileTime(db.path)
	tm_wal := OsFileTime(db.path + "-wal")
	tm_shm := OsFileTime(db.path + "-shm")

	changed := (tm != db.file_time || tm_wal != db.file_time_wal || tm_shm != db.file_time_shm)

	//update
	db.file_time = tm
	db.file_time_wal = tm_wal
	db.file_time_shm = tm_shm

	return changed
}
func (db *DiskDb) hasBeenWritten() bool {
	written := db.written
	db.written = false

	return written
}

func (db *DiskDb) setWritten() {
	db.written = true
	db.hasFileChanged() //update
}

func (db *DiskDb) HasFileChanged() bool {

	if !OsIsTicksIn(db.last_file_tick, 2000) {
		db.last_file_tick = OsTicks()

		if db.hasFileChanged() {
			return true
		}
	}

	if db.hasBeenWritten() {
		return true
	}

	return false
}

func (db *DiskDb) GetTableInfo() ([]*DiskDbIndexTable, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	var tables []*DiskDbIndexTable

	tableRows, err := db.Read_unsafe("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return nil, fmt.Errorf("Read() failed: %w", err)
	}
	for tableRows.Next() {
		var tname string
		err = tableRows.Scan(&tname)
		if err != nil {
			return nil, fmt.Errorf("Scan() failed: %w", err)
		}

		indt := &DiskDbIndexTable{Name: tname}
		tables = append(tables, indt)

		query := "pragma table_info(" + indt.Name + ");"
		columnRows, err := db.Read_unsafe(query)
		if err != nil {
			return nil, fmt.Errorf("Query(%s) failed: %w", query, err)
		}
		for columnRows.Next() {
			var cid int
			var cname, ctype string
			var pk int
			var notnull, dflt_value interface{}
			err = columnRows.Scan(&cid, &cname, &ctype, &notnull, &dflt_value, &pk)
			if err != nil {
				return nil, fmt.Errorf("Scan(%s) failed: %w", db.path, err)
			}

			c := &DiskDbIndexColumn{Name: cname, Type: ctype}
			indt.Columns = append(indt.Columns, c)
		}
	}

	return tables, nil
}

func (db *DiskDb) Vacuum() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	_, err := db.db.Exec("VACUUM;")
	db.setWritten()
	return err
}

func (db *DiskDb) Write_unsafe(query string, params ...any) (sql.Result, error) { //call Read_Lock() - Read_Unlock()
	//tx, err := db.Begin()
	//if err != nil {
	//	return nil, err
	//}
	//res, err := tx.Exec(query, params...)

	res, err := db.db.Exec(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query(%s) failed: %w", query, err)
	}

	db.lastWriteTicks = int64(OsTicks())

	db.setWritten()
	return res, nil
}

func (db *DiskDb) Write(query string, params ...any) (sql.Result, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	return db.Write_unsafe(query, params...)
}

func (db *DiskDb) Lock() {
	db.lock.Lock()
}
func (db *DiskDb) Unlock() {
	db.lock.Unlock()
}

func (db *DiskDb) ReadRow_unsafe(query string, params ...any) *sql.Row { //call Read_Lock() - Read_Unlock()
	db.lastReadTicks = int64(OsTicks())
	return db.db.QueryRow(query, params...)
}

func (db *DiskDb) Read_unsafe(query string, params ...any) (*sql.Rows, error) { //call Read_Lock() - Read_Unlock()
	db.lastReadTicks = int64(OsTicks())
	return db.db.Query(query, params...)
}

func (db *DiskDb) Print() error {

	tables, err := db.GetTableInfo() //lock/unlock inside!
	if err != nil {
		return err
	}

	db.Lock()
	defer db.Unlock()

	for _, t := range tables {
		//table name
		fmt.Println(t.Name)

		//columns headers
		for _, c := range t.Columns {
			fmt.Print(c.Name, ";")
		}
		fmt.Println()

		rows, err := db.Read_unsafe("SELECT * FROM " + t.Name)
		if err != nil {
			return err
		}

		// out fields
		colTypes, err := rows.ColumnTypes()
		if err != nil {
			return err
		}

		values := make([]interface{}, len(colTypes))
		scanCallArgs := make([]interface{}, len(colTypes))
		for i := range colTypes {
			scanCallArgs[i] = &values[i]
		}

		for rows.Next() {
			//reset
			for i := range values {
				values[i] = nil
			}

			err = rows.Scan(scanCallArgs...)
			if err != nil {
				return err
			}

			for _, v := range values {
				fmt.Print(v)
			}
			fmt.Println()

		}
		fmt.Println("----------")
	}

	return nil
}

/*func (db *DiskDb) Attach(path string, alias string, inMemory bool) error {

	if inMemory {
		//ATTACH DATABASE 'file:memdb1?mode=memory&cache=shared' AS aux1;
		_, err := db.Write("ATTACH DATABASE 'file:" + path + "?mode=memory&cache=shared' AS '" + alias + "';")
		if err != nil {
			return fmt.Errorf("ATTACH_memory(%s) failed: %w", path, err)
		}

	} else {
		_, err := db.Write("ATTACH DATABASE 'file:" + path + "?&_journal_mode=WAL' AS '" + alias + "';")
		if err != nil {
			return fmt.Errorf("ATTACH_file(%s) failed: %w", path, err)
		}
	}
	return nil
}

func (db *DiskDb) Detach(alias string) error {
	_, err := db.Write("DETACH DATABASE '" + alias + "';")
	if err != nil {
		return fmt.Errorf("DETACH(%s) failed: %w", alias, err)
	}
	return nil
}*/

/*func (db *DiskDb) Begin() (*sql.Tx, error) {
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
	//db.cache = nil

	db.disk.ResetTick()
	return err
}
func (db *DiskDb) Rollback() error {
	if db.tx == nil {
		return nil
	}

	err := db.tx.Rollback()
	db.tx = nil

	//reset queries
	//db.cache = nil

	db.disk.ResetTick()
	return err
}*/

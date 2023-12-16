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
	disk     *Disk
	path     string
	inMemory bool

	db *sql.DB
	//tx *sql.Tx

	//cache          []*DiskDbCache
	lastWriteTicks int64
	lastReadTicks  int64
}

func NewDiskDb(path string, inMemory bool, disk *Disk) (*DiskDb, error) {
	var db DiskDb
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

	return &db, nil
}

func (db *DiskDb) Destroy() {

	db.db.Exec("PRAGMA wal_checkpoint(full);")

	//db.Commit()

	err := db.db.Close()
	if err != nil {
		fmt.Printf("db(%s).Destroy() failed: %v\n", db.path, err)
	}
}

func DiskDb_Vacuum(path string) error {
	db, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		return fmt.Errorf("sql.Open(%s) failed: %w", path, err)
	}
	defer db.Close()

	db.Exec("VACUUM;")

	return nil
}

func (db *DiskDb) Attach(path string, alias string, inMemory bool) error {

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
}

func (db *DiskDb) GetTableInfo() ([]*DiskIndexTable, error) {

	var tables []*DiskIndexTable

	rows, err := db.Read("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return nil, fmt.Errorf("Read() failed: %w", err)
	}
	for rows.Next() {
		var tname string
		err = rows.Scan(&tname)
		if err != nil {
			return nil, fmt.Errorf("Scan() failed: %w", err)
		}

		t := &DiskIndexTable{Name: tname}
		tables = append(tables, t)
		err = t.updateDb(db)
		if err != nil {
			return nil, fmt.Errorf("updateDb() failed: %w", err)
		}
	}

	return tables, nil
}

func (db *DiskDb) Tick() {

}

func (db *DiskDb) Vacuum() {
	db.db.Exec("VACUUM;")
}

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

func (db *DiskDb) Write(query string, params ...any) (sql.Result, error) {

	/*tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(query, params...)*/
	res, err := db.db.Exec(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query(%s) failed: %w", query, err)
	}

	db.lastWriteTicks = int64(OsTicks())

	return res, nil
}

func (db *DiskDb) ReadRow(query string, params ...any) *sql.Row {
	db.lastReadTicks = int64(OsTicks())
	return db.db.QueryRow(query, params...)
}

func (db *DiskDb) Read(query string, params ...any) (*sql.Rows, error) {
	db.lastReadTicks = int64(OsTicks())
	return db.db.Query(query, params...)
}

func (db *DiskDb) Print() error {

	tables, err := db.GetTableInfo()
	if err != nil {
		return err
	}

	for _, t := range tables {
		//table name
		fmt.Println(t.Name)

		//columns headers
		for _, c := range t.Columns {
			fmt.Print(c.Name, ";")
		}
		fmt.Println()

		rows, err := db.Read("SELECT * FROM " + t.Name)
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

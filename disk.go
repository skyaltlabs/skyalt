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
	"path/filepath"

	"github.com/mattn/go-sqlite3"
)

const SKYALT_MAX_DBS = 11

func InitSQLiteGlobal() error {
	sql.Register("sqlite3_skyalt",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				conn.SetLimit(sqlite3.SQLITE_LIMIT_ATTACHED, SKYALT_MAX_DBS) //wont go above 10? Recompile sqlite? ...
				//fmt.Println(conn.GetLimit(sqlite3.SQLITE_LIMIT_ATTACHED))
				return nil
			},
		})
	return nil
}

type DiskIndexColumn struct {
	Name string
	Type string
}

type DiskIndexTable struct {
	Name    string
	Columns []*DiskIndexColumn
}

func (indt *DiskIndexTable) updateDb(db *DiskDb) error {

	query := "pragma table_info(" + indt.Name + ");"
	rows, err := db.Read(query)
	if err != nil {
		return fmt.Errorf("Query(%s) failed: %w", query, err)
	}
	for rows.Next() {
		var cid int
		var cname, ctype string
		var pk int
		var notnull, dflt_value interface{}
		err = rows.Scan(&cid, &cname, &ctype, &notnull, &dflt_value, &pk)
		if err != nil {
			return fmt.Errorf("Scan(%s) failed: %w", db.path, err)
		}

		c := &DiskIndexColumn{Name: cname, Type: ctype}
		indt.Columns = append(indt.Columns, c)
	}

	return nil
}

type Disk struct {
	last_ticks int64

	dbs map[string]*DiskDb

	net *DiskNet
}

func NewDisk() (*Disk, error) {
	var disk Disk

	disk.dbs = make(map[string]*DiskDb)

	disk.net = NewDiskNet()

	return &disk, nil
}

func (disk *Disk) Destroy() {
	for _, db := range disk.dbs {
		db.Destroy()
	}

	disk.net.Destroy()
}

func (disk *Disk) OpenDb(path string) (*DiskDb, bool, error) {

	//find
	db, found := disk.dbs[path]

	if !found {
		folder := filepath.Dir(path)
		err := OsFolderCreate(folder)
		if err != nil {
			return nil, false, fmt.Errorf("OsFolderCreate(%s) failed: %w", folder, err)
		}

		//open
		db, err = NewDiskDb(path, false, disk)
		if err != nil {
			return nil, false, fmt.Errorf("NewDiskDb(%s) failed: %w", path, err)
		}

		//add
		disk.dbs[path] = db
	}

	return db, found, nil
}

func (disk *Disk) ResetTick() {
	disk.last_ticks = 0
}

func (disk *Disk) Tick() {
	/*if time.Now().UnixMilli() > disk.last_ticks+3000 {
		disk.UpdateIndex()
		disk.last_ticks = OsTicks()
	}*/

	//close innactive databases
	//..

}

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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

func (indt *DiskIndexTable) updateDb(db *sql.DB, tname string, path string) error {

	rows, err := db.Query("pragma table_info(" + tname + ");")
	if err != nil {
		return fmt.Errorf("indt.Query(%s) failed: %w", path, err)
	}
	for rows.Next() {
		var cid int
		var cname, ctype string
		var pk int
		var notnull, dflt_value interface{}
		err = rows.Scan(&cid, &cname, &ctype, &notnull, &dflt_value, &pk)
		if err != nil {
			return fmt.Errorf("indt.Scan(%s) failed: %w", path, err)
		}

		c := &DiskIndexColumn{Name: cname, Type: ctype}
		indt.Columns = append(indt.Columns, c)
	}

	return nil
}

type DiskIndexFile struct {
	Name      string
	Timestamp int64
	Tables    []*DiskIndexTable
	Size      int64
}

func (indf *DiskIndexFile) updateDb(folder string) error {

	path := folder + "/" + indf.Name

	db, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		return fmt.Errorf("sql.Open(%s) failed: %w", path, err)
	}
	defer db.Close()

	indf.Tables = nil

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return fmt.Errorf("indf.Query(%s) failed: %w", path, err)
	}
	for rows.Next() {
		var tname string
		err = rows.Scan(&tname)
		if err != nil {
			return fmt.Errorf("indf.Scan(%s) failed: %w", path, err)
		}

		c := &DiskIndexTable{Name: tname}
		indf.Tables = append(indf.Tables, c)
		err = c.updateDb(db, tname, path)
		if err != nil {
			return fmt.Errorf("updateDb(%s) failed: %w", path, err)
		}
	}

	return nil
}

func (indf *DiskIndexFile) Vacuum(folder string) error {

	path := folder + "/" + indf.Name
	db, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		return fmt.Errorf("sql.Open(%s) failed: %w", path, err)
	}
	defer db.Close()

	db.Exec("VACUUM;")

	return nil
}

type DiskIndexFolder struct {
	Name    string
	Folders []*DiskIndexFolder
	Files   []*DiskIndexFile

	Size int64
}

func (ind *DiskIndexFolder) findFile(name string) *DiskIndexFile {
	for i := range ind.Files {
		if ind.Files[i].Name == name {
			return ind.Files[i]
		}
	}
	return nil
}
func (ind *DiskIndexFolder) findFolder(name string) *DiskIndexFolder {
	for i := range ind.Folders {
		if ind.Folders[i].Name == name {
			return ind.Folders[i]
		}
	}
	return nil
}

func (ind *DiskIndexFolder) update(path string) {

	dir, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("ReadDir() failed: %v\n", err)
		return
	}

	sumSize := int64(0)

	for _, file := range dir {
		filePath := path + "/" + file.Name()

		if file.IsDir() {
			f := ind.findFolder(file.Name())
			if f == nil {
				f = &DiskIndexFolder{Name: file.Name()}
				ind.Folders = append(ind.Folders, f)
			}
			f.update(filePath)
			sumSize += f.Size

		} else {
			if strings.HasSuffix(file.Name(), "-shm") || strings.HasSuffix(file.Name(), "-wal") {
				continue //skip
			}
			if strings.EqualFold(file.Name(), "index.json") {
				continue //skip
			}

			f := ind.findFile(file.Name())
			if f == nil {
				f = &DiskIndexFile{Name: file.Name(), Timestamp: -1}
				ind.Files = append(ind.Files, f)
			}

			inf, err := file.Info()

			if err != nil {
				fmt.Printf("Info() failed: %v\n", err)
				continue
			}
			tm := inf.ModTime().Unix()
			if f.Timestamp != tm {
				err = f.updateDb(path)
				if err != nil {
					fmt.Printf("updateDb() failed: %v\n", err)
					continue
				}

				f.Timestamp = tm
				f.Size = inf.Size()
			}
			sumSize += f.Size
		}
	}

	//remove structures which don't have folder/files anymore
	for i := len(ind.Folders) - 1; i >= 0; i-- {
		if !OsFolderExists(path + "/" + ind.Folders[i].Name) {
			ind.Folders = append(ind.Folders[:i], ind.Folders[i+1:]...)
		}
	}

	for i := len(ind.Files) - 1; i >= 0; i-- {
		if !OsFileExists(path + "/" + ind.Files[i].Name) {
			ind.Files = append(ind.Files[:i], ind.Files[i+1:]...)
		}
	}

	ind.Size = sumSize
}

func (ind *DiskIndexFolder) Vacuum(path string) {
	for _, it := range ind.Files {
		err := it.Vacuum(path)
		if err != nil {
			fmt.Printf("Vacuum() failed: %v", err)
		}
	}

	for _, it := range ind.Folders {
		it.Vacuum(path + "/" + ind.Name)
	}
}

type Disk struct {
	folder string

	index DiskIndexFolder

	last_ticks int64
}

func NewDbs(folder string) (*Disk, error) {
	var disk Disk
	disk.folder = folder

	err := OsFolderCreate(folder)
	if err != nil {
		return nil, fmt.Errorf("OsFolderCreate() failed: %w", err)
	}
	if !OsFolderExists(folder) {
		return nil, fmt.Errorf("Folder(%s) not exist", folder)
	}

	//load index
	err = disk.loadIndex()
	if err != nil {
		return nil, fmt.Errorf("loadIndex() failed: %w", err)
	}

	disk.UpdateIndex()

	return &disk, nil
}

func (disk *Disk) Destroy() {

	err := disk.SaveIndex()
	if err != nil {
		fmt.Printf("SaveIndex() failed: %v\n", err)
	}
}

func (disk *Disk) GetIndexPath() string {
	return disk.folder + "/index.json"
}

func (disk *Disk) UpdateIndex() {
	disk.index.update(disk.folder)
}

func (disk *Disk) loadIndex() error {
	js, err := os.ReadFile(disk.GetIndexPath())
	if err == nil {
		err := json.Unmarshal([]byte(js), &disk.index)
		if err != nil {
			return fmt.Errorf("OsFolderCreate() failed: %w", err)
		}
	}
	return nil
}

func (disk *Disk) SaveIndex() error {
	js, err := json.MarshalIndent(&disk.index, "", "")
	if err != nil {
		return fmt.Errorf("MarshalIndent() failed: %w", err)
	} else {
		err = os.WriteFile(disk.GetIndexPath(), js, 0644)
		if err != nil {
			return fmt.Errorf("WriteFile() failed: %w", err)
		}
	}
	return nil
}

func (disk *Disk) OpenDb(path string) (*DiskDb, error) {

	db, err := sql.Open("sqlite3", "file:"+disk.folder+"/"+path+"?&_journal_mode=WAL") //sqlite3 -> sqlite3_skyalt
	if err != nil {
		return nil, fmt.Errorf("Open(%s) failed: %w", path, err)
	}

	return NewDiskDb(disk, db), nil
}

func (disk *Disk) OpenDbs(paths []string) (*DiskDb, error) {

	if len(paths) == 0 {
		return nil, fmt.Errorf("0 paths")
	}

	if len(paths) > SKYALT_MAX_DBS {
		fmt.Printf("cutting databases: maximum is %d\n", SKYALT_MAX_DBS)
		paths = paths[:SKYALT_MAX_DBS]
	}

	db, err := sql.Open("sqlite3", "file:"+disk.folder+"/"+paths[0]+"?&_journal_mode=WAL") //sqlite3 -> sqlite3_skyalt
	if err != nil {
		return nil, fmt.Errorf("Open(%s) failed: %w", paths[0], err)
	}

	for i := 1; i < len(paths); i++ {
		db.Exec("ATTACH '" + disk.folder + "/" + paths[i] + "' AS db" + strconv.Itoa(i))

	}

	return NewDiskDb(disk, db), nil
}

func (disk *Disk) Vacuum() {
	disk.index.Vacuum(disk.folder)
}

func (disk *Disk) ResetTick() {
	disk.last_ticks = 0
}

func (disk *Disk) Tick() {
	if time.Now().UnixMilli() > disk.last_ticks+3000 {
		disk.UpdateIndex()
		disk.last_ticks = OsTicks()
	}
}

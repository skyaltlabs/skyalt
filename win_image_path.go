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
	"fmt"
	"os"
	"strconv"
	"strings"
)

type WinMedia struct {
	blob      []byte
	blob_hash OsHash

	path string

	//optional(blob)
	table  string
	column string
	row    int
}

func InitWinMedia_blob(blob []byte, hash OsHash) WinMedia {
	var ip WinMedia
	ip.blob = blob
	ip.blob_hash = hash
	return ip
}

func InitWinMedia_url(url string) WinMedia {
	var ip WinMedia

	//get type + cut
	var isFile bool
	url, isFile = strings.CutPrefix(url, "file:")
	if !isFile {
		var isDbsPath bool
		url, isDbsPath = strings.CutPrefix(url, "db:")
		if !isDbsPath {
			fmt.Println("must start with 'file:' or 'db:'")
			return WinMedia{}
		}
	}

	d := strings.Index(url, ":")
	if d >= 0 {
		ip.path = url[:d]
		//optional(table/column/row)
		opt := url[d+1:]

		//table
		d := strings.Index(opt, "/")
		if d <= 0 {
			fmt.Println("table '/' is missing")
			return WinMedia{}
		}
		ip.table = opt[:d]
		opt = opt[d+1:]

		//column
		d = strings.Index(opt, "/")
		if d <= 0 {
			fmt.Println("column '/' is missing")
			return WinMedia{}
		}
		ip.column = opt[:d]
		opt = opt[d+1:]

		//row
		var err error
		ip.row, err = strconv.Atoi(opt)
		if err != nil {
			fmt.Printf("Atoi(%s) failed: %v", opt, err)
			return WinMedia{}
		}
	} else {
		ip.path = url
	}

	return ip
}

func (ip *WinMedia) IsBlob() bool {
	return len(ip.blob) > 0
}
func (ip *WinMedia) IsDb() bool {
	return len(ip.table) > 0
}
func (ip *WinMedia) IsFile() bool {
	return !ip.IsDb()
}

func (ip *WinMedia) GetString() string {
	return fmt.Sprintf("%s - %s/%s/%d", ip.path, ip.table, ip.column, ip.row)
}

func (a *WinMedia) Cmp(b *WinMedia) bool {
	if a.IsBlob() && b.IsBlob() {
		return a.blob_hash.Cmp(&b.blob_hash)
	}
	return a.path == b.path && a.table == b.table && a.column == b.column && a.row == b.row
}

func (ip *WinMedia) GetBlob(disk *Disk) ([]byte, error) {
	var data []byte
	var err error

	if ip.IsBlob() {
		return ip.blob, nil
	} else if ip.IsFile() {
		//file
		data, err = os.ReadFile(ip.path)
		if err != nil {
			return nil, fmt.Errorf("ReadFile(%s) failed: %w", ip.path, err)
		}
	} else if disk != nil {
		//db
		db, _, err := disk.OpenDb(ip.path)
		if err != nil {
			return nil, fmt.Errorf("OpenDb(%s) failed: %w", ip.path, err)
		}

		row := db.ReadRow(fmt.Sprintf("SELECT %s FROM %s WHERE rowid==%d", ip.column, ip.table, ip.row))
		err = row.Scan(&data)
		if err != nil {
			return nil, fmt.Errorf("QueryRow(%s) failed: %w", ip.path, err)
		}
	}

	return data, nil
}

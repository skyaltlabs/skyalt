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
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MediaPath struct {
	isAppPath bool
	path      string

	//optional(blob)
	table  string
	column string
	row    int
}

func MediaParseUrl(url string) (MediaPath, error) {
	var ip MediaPath

	//get type + cut
	url, ip.isAppPath = strings.CutPrefix(url, "app:")
	if !ip.isAppPath {
		var isDbsPath bool
		url, isDbsPath = strings.CutPrefix(url, "dbs:")
		if !isDbsPath {
			return ip, fmt.Errorf("must start with 'dbs:' or 'app:'")
		}
	}

	d := strings.Index(url, ":")
	if d >= 0 {
		ip.path = url
		//optional(table/column/row)
		opt := url[d+1:]

		//table
		d := strings.Index(opt, "/")
		if d <= 0 {
			return ip, errors.New("table '/' is missing")
		}
		ip.table = opt[:d]
		opt = opt[d+1:]

		//column
		d = strings.Index(opt, "/")
		if d <= 0 {
			return ip, errors.New("column '/' is missing")
		}
		ip.column = opt[:d]
		opt = opt[d+1:]

		//row
		var err error
		ip.row, err = strconv.Atoi(opt)
		if err != nil {
			return ip, fmt.Errorf("Atoi(%s) failed: %w", opt, err)
		}
	} else {
		ip.path = url
	}

	return ip, nil
}

func (ip *MediaPath) IsDb() bool {
	return len(ip.table) > 0
}
func (ip *MediaPath) IsFile() bool {
	return !ip.IsDb()
}

func (ip *MediaPath) GetString() string {
	return fmt.Sprintf("%s - %s/%s/%d", ip.path, ip.table, ip.column, ip.row)
}

func (a *MediaPath) Cmp(b *MediaPath) bool {
	return a.path == b.path && a.table == b.table && a.column == b.column && a.row == b.row
}

func (ip *MediaPath) GetFileBlob() ([]byte, error) {
	var data []byte
	var err error

	if ip.IsFile() {
		data, err = os.ReadFile(ip.path)
		if err != nil {
			return nil, fmt.Errorf("ReadFile(%s) failed: %w", ip.path, err)
		}
	}

	return data, nil
}

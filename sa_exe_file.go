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
	"os"
	"path/filepath"
	"strings"
)

func SAExe_File_dir(node *SANode) bool {

	pathAttr := node.GetAttrUi("path", "", SAAttrUi_DIR)
	full_path := node.GetAttrUi("full_path", "0", SAAttrUi_SWITCH).GetBool()
	exts := strings.Split(node.GetAttr("exts", "").GetString(), ";")
	extsType := node.GetAttrUi("exts_type", "0", SAAttrUi_COMBO("Include;Exclude", "")).GetBool()
	if len(exts) == 1 && exts[0] == "" {
		exts = nil
	}

	filesAttr := node.GetAttr("_files", []byte("[]"))
	dirsAttr := node.GetAttr("_dirs", []byte("[]"))

	path := pathAttr.GetString()
	if path == "" {
		pathAttr.SetErrorStr("empty")
		return false
	}
	dir, err := os.ReadDir(path)
	if err != nil {
		pathAttr.SetError(err)
		return false
	}

	filesStr := "["
	dirsStr := "["
	for _, f := range dir {
		nm := f.Name()
		ext := filepath.Ext(nm)
		ok := true

		if len(exts) > 0 {
			ok = OsTrnBool(!extsType, false, true)
			for _, e := range exts {
				if ext == e {
					ok = OsTrnBool(!extsType, true, false)
				}
			}
		}

		if ok {
			if full_path {
				nm = filepath.Join(path, nm)
			}
			if f.IsDir() {
				dirsStr += "\"" + nm + "\","
			} else {
				filesStr += "\"" + nm + "\","
			}
		}
	}
	filesStr, _ = strings.CutSuffix(filesStr, ",")
	dirsStr, _ = strings.CutSuffix(dirsStr, ",")
	filesStr += "]"
	dirsStr += "]"

	filesAttr.SetOutBlob([]byte(filesStr))
	dirsAttr.SetOutBlob([]byte(dirsStr))

	return true
}

func SAExe_File_write(node *SANode) bool {

	triggerAttr := node.GetAttrUi("trigger", 0, SAAttrUi_SWITCH)

	tp := node.GetAttrUi("type", 0, SAAttrUi_COMBO("Database;App;Disk", "")).GetInt()
	pathAttr := node.GetAttrUi("path", "", SAAttrUi_FILE)
	dataAttr := node.GetAttr("data", "")

	if triggerAttr.GetBool() {

		path := pathAttr.GetString()
		if tp <= 1 {
			path = OsTrnString(tp == 0, "databases/", "apps/"+node.app.Name+"/") + path
		}

		err := os.WriteFile(path, dataAttr.GetBlob().data, 0644)
		if err != nil {
			pathAttr.SetError(err)
			return false
		}

		//triggerAttr.AddSetAttr("0")
	}

	return true
}

func SAExe_File_read(node *SANode) bool {

	tp := node.GetAttrUi("type", 0, SAAttrUi_COMBO("Database;App;Disk", "")).GetInt()

	pathAttr := node.GetAttrUi("path", "", SAAttrUi_FILE)
	path := pathAttr.GetString()

	outputAttr := node.GetAttrUi("_out", []byte("[]"), SAAttrUi_BLOB)

	if path == "" {
		pathAttr.SetErrorStr("value is empty")
		return false
	}

	//get data
	var data []byte
	var err error
	if tp <= 1 {
		url := OsTrnString(tp == 0, "db:", "file:apps/"+node.app.Name+"/") + path
		m := InitWinMedia_url(url)
		data, err = m.GetBlob(node.app.base.ui.win.disk)
	} else {
		data, err = os.ReadFile(path)
	}

	//set
	if err != nil {
		pathAttr.SetError(err)
		return false
	}
	outputAttr.SetOutBlob(data)

	return true
}

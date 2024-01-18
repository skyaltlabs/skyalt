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
)

func (node *SANode) SAExe_File_write() bool {

	triggerAttr := node.GetAttrUi("trigger", "0", SAAttrUi_SWITCH)

	tp := node.GetAttrUi("type", "0", SAAttrUiValue{Fn: "combo", Prm: "Database;App;Disk"}).GetInt()
	pathAttr := node.GetAttr("path", "")
	dataAttr := node.GetAttr("json", "")

	if triggerAttr.GetBool() {

		path := pathAttr.GetString()
		if tp <= 1 {
			path = OsTrnString(tp == 0, "databases/", "apps/"+node.app.Name+"/") + path
		}

		err := os.WriteFile(path, dataAttr.GetBlob(), 0644)
		if err != nil {
			pathAttr.SetErrorExe(fmt.Sprintf("%v", err))
			return false
		}

		triggerAttr.SetExpBool(false)
	}

	return true
}

func (node *SANode) SAExe_File_read() bool {

	tp := node.GetAttrUi("type", "0", SAAttrUiValue{Fn: "combo", Prm: "Database;App;Disk"}).GetInt()

	pathAttr := node.GetAttr("path", "")
	path := pathAttr.GetString()

	outputAttr := node.GetAttrUi("_out", "", SAAttrUi_BLOB)
	outputAttr.result.SetBlob(nil) //reset

	if path == "" {
		pathAttr.SetErrorExe("value is empty")
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
		pathAttr.SetErrorExe(fmt.Sprintf("%v", err))
		return false
	}
	outputAttr.result.SetBlob(data)

	return true
}

func (node *SANode) SAExe_Vars() bool {
	//nothing here, it's all about RenderAttrs()
	return true
}

//neukládat jako JSON(ponechat save()), ale jako lines: ... kam dát Node.Pos,Bypass? ..............................
//edit = editbox(grid:[0, 0, 1, 1], grid_show:1, value:"hello")
//text = text(grid:[1, 2, 1, 1], grid_show:1, value: edit.value & "hi")

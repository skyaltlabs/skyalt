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

func (node *SANode) SAExe_IO_write() bool {

	triggerAttr := node.GetAttrUi("trigger", "0", SAAttrUi_SWITCH)
	fileAttr := node.GetAttr("file", "")
	jsonAttr := node.GetAttr("json", "")

	if triggerAttr.GetBool() {
		os.WriteFile(fileAttr.result.String(), jsonAttr.result.Blob(), 0644)

		triggerAttr.SetExpBool(false)
	}

	return true
}

func (w *SANode) SAExe_IO_blob() bool {

	tp := int(w.GetAttrUi("type", "0", SAAttrUiValue{Fn: "combo", Prm: "Database;App"}).result.Number())

	pathAttr := w.GetAttr("path", "")
	instr := pathAttr.instr.GetConst()
	value := instr.pos_attr.result.String()

	outputAttr := w.GetAttrUi("_out", "", SAAttrUi_BLOB)
	outputAttr.result.SetBlob(nil) //reset

	if value == "" {
		pathAttr.SetErrorExe("value is empty")
		return false
	}

	url := OsTrnString(tp == 0, "db:", "file:apps/"+w.app.Name+"/") + value
	m := InitWinMedia_url(url)
	data, err := m.GetBlob(w.app.base.ui.win.disk)
	if err != nil {
		pathAttr.SetErrorExe(fmt.Sprintf("GetBlob() failed: %v", err))
		return false
	}
	outputAttr.result.SetBlob(data)

	return true
}

func (w *SANode) SAExe_Medium() bool {
	//nothing here, it's all about RenderAttrs()
	return true
}

//neukládat jako JSON(ponechat save()), ale jako lines: ... kam dát Node.Pos,Bypass? ..............................
//edit = editbox(grid:[0, 0, 1, 1], grid_show:1, value:"hello")
//text = text(grid:[1, 2, 1, 1], grid_show:1, value: edit.value & "hi")

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

import "fmt"

func (w *SANode) ConstBlob() bool {

	tp := int(w.GetAttr("type", "uiCombo(0, \"Database;App\")").result.Number())

	pathAttr := w.GetAttr("path", "")
	instr := pathAttr.instr.GetConst()
	value := instr.pos_attr.result.String()

	outputAttr := w.GetAttrOutput("output", "uiBlob(0)")
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

func (w *SANode) ConstAttribute() bool {

	//nothing here, it's all about RenderAttrs()

	//mark attributes
	for _, it := range w.Attrs {
		it.useMark = true
	}

	return true
}

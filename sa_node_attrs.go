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
	"strconv"
)

func (node *SANode) GetGridShow() bool {
	return node.GetAttrBool("show", true)
}

func (node *SANode) SetGridStart(v OsV2) {
	node.Attrs["grid_x"] = v.X
	node.Attrs["grid_y"] = v.Y
}
func (node *SANode) SetGridSize(v OsV2) {
	node.Attrs["grid_w"] = v.X
	node.Attrs["grid_h"] = v.Y
}
func (node *SANode) SetGrid(grid OsV4) {
	node.SetAttrV4("grid", grid)
}
func (node *SANode) GetGrid() OsV4 {
	return node.GetAttrV4("grid", InitOsV4(0, 0, 1, 1))
}

func (node *SANode) SetAttrV4(namePrefix string, value OsV4) {
	node.Attrs[namePrefix+"_x"] = value.Start.X
	node.Attrs[namePrefix+"_y"] = value.Start.Y
	node.Attrs[namePrefix+"_w"] = value.Size.X
	node.Attrs[namePrefix+"_h"] = value.Size.Y
}
func (node *SANode) GetAttrV4(namePrefix string, defValue OsV4) OsV4 {
	var value OsV4

	value.Start.X = node.GetAttrInt(namePrefix+"_x", defValue.Start.X)
	value.Start.Y = node.GetAttrInt(namePrefix+"_y", defValue.Start.Y)
	value.Size.X = node.GetAttrInt(namePrefix+"_w", defValue.Size.X)
	value.Size.Y = node.GetAttrInt(namePrefix+"_h", defValue.Size.Y)

	return value
}

func (node *SANode) SetAttrCd(namePrefix string, value OsCd) {
	node.Attrs[namePrefix+"_r"] = int(value.R)
	node.Attrs[namePrefix+"_g"] = int(value.G)
	node.Attrs[namePrefix+"_b"] = int(value.B)
	node.Attrs[namePrefix+"_a"] = int(value.A)
}
func (node *SANode) GetAttrCd(namePrefix string, defValue OsCd) OsCd {
	var value OsCd

	value.R = byte(node.GetAttrInt(namePrefix+"_r", int(defValue.R)))
	value.G = byte(node.GetAttrInt(namePrefix+"_g", int(defValue.G)))
	value.B = byte(node.GetAttrInt(namePrefix+"_b", int(defValue.B)))
	value.A = byte(node.GetAttrInt(namePrefix+"_a", int(defValue.A)))

	return value
}

func (node *SANode) GetAttrBool(name string, defValue bool) bool {
	value, found := node.Attrs[name]
	if !found {
		node.Attrs[name] = defValue
		value = defValue
	}

	switch vv := value.(type) {
	case bool:
		return vv
	case int:
		return vv != 0
	case float64:
		return vv != 0
	default:
		node.Attrs[name] = defValue
		return defValue
	}
}
func (node *SANode) GetAttrInt(name string, defValue int) int {
	value, found := node.Attrs[name]
	if !found {
		node.Attrs[name] = defValue
		value = defValue
	}

	switch vv := value.(type) {
	case bool:
		return OsTrn(vv, 1, 0)
	case int:
		return vv
	case float64:
		return int(vv)
	default:
		node.Attrs[name] = defValue
		return defValue
	}
}
func (node *SANode) GetAttrFloat(name string, defValue float64) float64 {
	value, found := node.Attrs[name]
	if !found {
		node.Attrs[name] = defValue
		value = defValue
	}

	switch vv := value.(type) {
	case bool:
		return OsTrnFloat(vv, 1, 0)
	case int:
		return float64(vv)
	case float64:
		return vv
	default:
		node.Attrs[name] = defValue
		return defValue
	}
}
func (node *SANode) GetAttrString(name string, defValue string) string {
	value, found := node.Attrs[name]
	if !found {
		node.Attrs[name] = defValue
		value = defValue
	}

	switch vv := value.(type) {
	case bool:
		return OsTrnString(vv, "1", "0")
	case int:
		return strconv.Itoa(vv)
	case float64:
		return strconv.FormatFloat(vv, 'f', -1, 64)
	case string:
		return vv
	default:
		node.Attrs[name] = defValue
		return defValue
	}
}

func (node *SANode) showAttrName(grid *OsV4, name string, isDefault bool) {
	ui := node.app.base.ui
	if !isDefault {
		name = "**" + name + "**"
	}
	ui.Comp_text(grid.Start.X+0, grid.Start.Y, grid.Size.X, grid.Size.Y, name, 0)
}

func (node *SANode) ShowAttrV4(grid *OsV4, namePrefix string, defValue OsV4) OsV4 {
	ui := node.app.base.ui

	value := node.GetAttrV4(namePrefix, defValue)

	node.showAttrName(grid, namePrefix, value.Cmp(defValue))

	ui.Div_start(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(1, 100)
		ui.Div_colMax(2, 100)
		ui.Div_colMax(3, 100)
		_, _, _, fnshd1, _ := ui.Comp_editbox(0, 0, 1, 1, &value.Start.X, Comp_editboxProp().Ghost("x").Precision(0))
		_, _, _, fnshd2, _ := ui.Comp_editbox(1, 0, 1, 1, &value.Start.Y, Comp_editboxProp().Ghost("y").Precision(0))
		_, _, _, fnshd3, _ := ui.Comp_editbox(2, 0, 1, 1, &value.Size.X, Comp_editboxProp().Ghost("w").Precision(0))
		_, _, _, fnshd4, _ := ui.Comp_editbox(3, 0, 1, 1, &value.Size.Y, Comp_editboxProp().Ghost("h").Precision(0))
		if fnshd1 || fnshd2 || fnshd3 || fnshd4 {
			node.SetAttrV4(namePrefix, value)
			node.SetStructChange()
		}
	}
	ui.Div_end()

	grid.Start.Y += grid.Size.Y
	return value
}

func (node *SANode) ShowAttrCd(grid *OsV4, namePrefix string, defValue OsCd) OsCd {
	ui := node.app.base.ui

	value := node.GetAttrCd(namePrefix, defValue)

	node.showAttrName(grid, namePrefix, value.Cmp(defValue))

	if ui.comp_colorPicker(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, namePrefix+"_"+node.Name, "", true) {
		node.SetAttrCd(namePrefix, value)
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	return value
}

func (node *SANode) ShowAttrBool(grid *OsV4, name string, defValue bool) bool {
	ui := node.app.base.ui

	value := node.GetAttrBool(name, defValue)

	node.showAttrName(grid, name, value == defValue)

	if ui.Comp_switch(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, "", "", true) {
		node.Attrs[name] = value
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	return value
}

func (node *SANode) ShowAttrInt(grid *OsV4, name string, defValue int) int {
	ui := node.app.base.ui

	value := node.GetAttrInt(name, defValue)

	node.showAttrName(grid, name, value == defValue)

	_, _, _, fnshd1, _ := ui.Comp_editbox(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, Comp_editboxProp().Precision(0))
	if fnshd1 {
		node.Attrs[name] = value
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	return value
}
func (node *SANode) ShowAttrFloat(grid *OsV4, name string, defValue float64, prec int) float64 {
	ui := node.app.base.ui

	value := node.GetAttrFloat(name, defValue)

	node.showAttrName(grid, name, value == defValue)

	_, _, _, fnshd1, _ := ui.Comp_editbox(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, Comp_editboxProp().Precision(prec))
	if fnshd1 {
		node.Attrs[name] = value
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	return value
}

func (node *SANode) ShowAttrStringEx(grid *OsV4, name string, defValue string, multiLine bool, showName bool) string {
	ui := node.app.base.ui

	oldSizeY := grid.Size.Y
	if multiLine {
		grid.Size.Y += 3
	}

	value := node.GetAttrString(name, defValue)

	if showName {
		node.showAttrName(grid, name, value == defValue)
	}

	_, _, _, fnshd, _ := ui.Comp_editbox(grid.Start.X+OsTrn(showName, 1, 0), grid.Start.Y, grid.Size.X, grid.Size.Y, &value, Comp_editboxProp().Align(0, OsTrn(multiLine, 0, 1)).MultiLine(multiLine, true).Formating(false))
	if fnshd {
		node.Attrs[name] = value
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	grid.Size.Y = oldSizeY
	return value
}

func (node *SANode) ShowAttrString(grid *OsV4, name string, defValue string, multiLine bool) string {
	return node.ShowAttrStringEx(grid, name, defValue, multiLine, true)
}

func (node *SANode) ShowAttrIntCombo(grid *OsV4, name string, defValue int, options_names []string, options_values []string) int {
	ui := node.app.base.ui

	value := node.GetAttrInt(name, defValue)

	node.showAttrName(grid, name, value == defValue)

	valueStr := strconv.Itoa(value)
	if ui.Comp_combo(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, &valueStr, options_names, options_values, "", true, false) {
		node.Attrs[name], _ = strconv.Atoi(valueStr)
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	return value
}
func (node *SANode) ShowAttrStringCombo(grid *OsV4, name string, defValue string, options_names []string, options_values []string) string {
	ui := node.app.base.ui

	value := node.GetAttrString(name, defValue)

	node.showAttrName(grid, name, value == defValue)

	if ui.Comp_combo(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, options_names, options_values, "", true, false) {
		node.Attrs[name] = value
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	return value
}

func (node *SANode) ShowAttrFilePicker(grid *OsV4, name string, defValue string, selectFile bool, errWhenEmpty bool, dialogName string) string {
	ui := node.app.base.ui

	value := node.GetAttrString(name, defValue)

	node.showAttrName(grid, name, value == defValue)

	if ui.Comp_dirPicker(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, selectFile, errWhenEmpty, dialogName, true) {
		node.Attrs[name] = value
		node.SetStructChange()
	}

	grid.Start.Y += grid.Size.Y
	return value
}

func (node *SANode) RenderAttrs() {

	ui := node.app.base.ui

	ui.Div_colMax(0, 100)
	if node.HasError() {
		/*err_n := 0.0
		if node.errExe != nil {
			err_n++
		}
		if node.Code.file_err != nil {
			err_n++
		}
		if node.Code.exe_err != nil {
			err_n++
		}*/
		ui.Div_row(0, 1+3)
	}
	ui.Div_rowMax(1, 100)

	//rename + type
	ui.Div_start(0, 0, 1, 1)
	{
		pl := ui.win.io.GetPalette()
		ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.8), 0)

		ui.Div_colMax(0, 100)
		ui.Div_colMax(2, 4)

		old_path := NewSANodePath(node)
		_, _, _, fnshd, _ := ui.Comp_editbox_desc("Name", 0, 3, 0, 0, 1, 1, &node.Name, Comp_editboxProp())
		if fnshd {
			node.CheckUniqueName()
			node.GetRoot().RenameCodeSubDepends(old_path, NewSANodePath(node), node.IsTypeWithSubLayoutNodes())
		}

		//type
		node.Exe = node.app.ComboListOfNodes(2, 0, 1, 1, node.Exe)

		//context
		{
			dnm := "node_" + node.Name
			if ui.Comp_buttonIcon(3, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/context.png"), 0.3, "", Comp_buttonProp().Cd(CdPalette_B)) > 0 {
				ui.Dialog_open(dnm, 1)
			}
			if ui.Dialog_start(dnm) {
				ui.Div_colMax(0, 5)
				y := 0

				if ui.Comp_buttonMenu(0, y, 1, 1, ui.trns.DUPLICATE, false, Comp_buttonProp()) > 0 {
					nw := node.parent.AddNodeCopy(node)
					nw.SelectOnlyThis()
					ui.Dialog_close()
				}
				y++

				if ui.Comp_buttonMenu(0, y, 1, 1, ui.trns.REMOVE, false, Comp_buttonProp()) > 0 {
					node.Remove()
					ui.Dialog_close()
				}
				y++

				ui.Dialog_end()
			}
		}

		//error
		if node.HasError() {
			err_y := 1
			if node.errExe != nil {
				ui.Comp_textCd(0, err_y, 4, 1, "Error: "+node.errExe.Error(), 0, CdPalette_E)
				err_y++
			}

			if node.Code.file_err != nil {
				ui.Comp_textCd(0, err_y, 4, 1, "File Error: "+node.Code.file_err.Error(), 0, CdPalette_E)
				err_y++
			}
			if node.Code.exe_err != nil {
				ui.Comp_textCd(0, err_y, 4, 1, "Execute Error: "+node.Code.exe_err.Error(), 0, CdPalette_E)
				err_y++
			}
		}

	}
	ui.Div_end()

	gnd := node.app.base.node_groups.FindNode(node.Exe)
	if gnd != nil && gnd.attrs != nil {
		ui.Div_start(0, 1, 1, 1)
		ui.Div_colMax(0, 100)

		node.errExe = nil //reset
		gnd.attrs(node)
		ui.Div_end()
	}
}

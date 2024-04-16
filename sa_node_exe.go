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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-audio/wav"
)

func UiButton_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrIntCombo(&grid, "background", 1, []string{"Transparent", "Full", "Light"}, []string{"0", "1", "2"})
	node.ShowAttrIntCombo(&grid, "align", 1, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrFilePicker(&grid, "icon", "", true, false, "select_icon")
	node.ShowAttrFloat(&grid, "icon_margin", 0.15, 2)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrString(&grid, "confirmation", "", false)
	node.ShowAttrBool(&grid, "close_dialog", false)
}

func UiButton_render(node *SANode) {
	grid := node.GetGrid()
	background := node.GetAttrInt("background", 1)
	align := node.GetAttrInt("align", 1)
	label := node.GetAttrString("label", "")
	icon_path := node.GetAttrString("icon", "")
	icon_margin := node.GetAttrFloat("icon_margin", 0.15)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	confirmation := node.GetAttrString("confirmation", "")
	close_dialog := node.GetAttrBool("close_dialog", false)

	props := Comp_buttonProp().Enable(enable).Tooltip(tooltip).Align(align, 1)

	if icon_path != "" {
		icon := InitWinMedia_url("file:" + icon_path)
		props.Icon(&icon).ImgMargin(icon_margin)
	}

	if confirmation != "" {
		path := NewSANodePath(node)
		props.Confirmation(confirmation, "confirm_"+path.String())
	}

	switch background {
	case 0:
		props.DrawBack(false)
	case 1:
		props.DrawBack(true)
	case 2:
		props.DrawBackLight(true)
	}

	if node.app.base.ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, props) > 0 {
		node.SetChange([]SANodeCodeExePrm{{Node: node.Name, Attr: "triggered", Value: true}})

		if close_dialog {
			node.app.base.ui.Dialog_close()
		}

		if node.parent != nil && node.parent.parent != nil && node.parent.parent.IsTypeList() {
			list := node.parent.parent

			selected_button := list.GetAttrString("selected_button", "")
			if selected_button == node.Name {
				found_i := list.FindListSubNodePos(node.parent)
				if found_i >= 0 {
					list.Attrs["selected_index"] = found_i
				}
			}
		}
	}
}

func UiMenu_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrIntCombo(&grid, "background", 1, []string{"Transparent", "Full", "Light"}, []string{"0", "1", "2"})
	node.ShowAttrIntCombo(&grid, "align", 1, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrFilePicker(&grid, "icon", "", true, false, "select_icon")
	node.ShowAttrFloat(&grid, "icon_margin", 0.15, 2)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
}

func UiMenu_render(node *SANode) {
	grid := node.GetGrid()
	background := node.GetAttrInt("background", 1)
	align := node.GetAttrInt("align", 1)
	label := node.GetAttrString("label", "")
	icon_path := node.GetAttrString("icon", "")
	icon_margin := node.GetAttrFloat("icon_margin", 0.15)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)

	props := Comp_buttonProp().Enable(enable).Tooltip(tooltip).Align(align, 1)

	if icon_path != "" {
		icon := InitWinMedia_url("file:" + icon_path)
		props.Icon(&icon).ImgMargin(icon_margin)
	}

	switch background {
	case 0:
		props.DrawBack(false)
	case 1:
		props.DrawBack(true)
	case 2:
		props.DrawBackLight(true)
	}

	ui := node.app.base.ui

	dnm := "app_" + node.Name
	if node.parent != nil && node.parent.parent != nil && node.parent.parent.IsTypeList() {
		lay := node.parent
		list := node.parent.parent
		dnm = "app_" + list.Name + "_" + lay.Name + "_" + node.Name
	}

	//button
	if ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, props) > 0 {
		ui.Dialog_open(dnm, 1)
	}

	//dialog
	if ui.Dialog_start(dnm) {
		node.renderLayout()
		ui.Dialog_end()
	}
}

func UiText_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrString(&grid, "label", "", node.GetAttrBool("multi_line", false))
	node.ShowAttrIntCombo(&grid, "align_h", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrIntCombo(&grid, "align_v", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrBool(&grid, "multi_line", false)
	node.ShowAttrBool(&grid, "line_wrapping", true)
	node.ShowAttrBool(&grid, "formating", true)
	node.ShowAttrBool(&grid, "selection", true)
	node.ShowAttrBool(&grid, "show_border", false)
}

func UiText_render(node *SANode) {
	grid := node.GetGrid()
	label := node.GetAttrString("label", "")
	align_v := node.GetAttrInt("align_v", 0)
	align_h := node.GetAttrInt("align_h", 0)
	selection := node.GetAttrBool("selection", true)
	show_border := node.GetAttrBool("show_border", false)
	line_wrapping := node.GetAttrBool("line_wrapping", true)
	formating := node.GetAttrBool("formating", true)

	if node.GetAttrBool("multi_line", false) {
		node.app.base.ui.Comp_textSelectMulti(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, show_border, formating, line_wrapping)
	} else {
		node.app.base.ui.Comp_textSelect(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, formating, show_border)
	}
}

func UiEditbox_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrString(&grid, "value", "", node.GetAttrBool("multi_line", false))
	node.ShowAttrString(&grid, "ghost", "", false)
	node.ShowAttrIntCombo(&grid, "align_h", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrIntCombo(&grid, "align_v", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "multi_line", false)
	node.ShowAttrBool(&grid, "multi_line_enter_finish", false)
	node.ShowAttrBool(&grid, "line_wrapping", true)
	node.ShowAttrBool(&grid, "formating", true)

	node.ShowAttrBool(&grid, "temp_to_value", false)

	node.ShowAttrBool(&grid, "db_value", false)
}

func (node *SANode) extractDBpath() (string, string, string, int, error) {
	value := node.GetAttrString("value", "")

	parts := strings.Split(value, ":")
	if len(parts) != 4 {
		return "", "", "", 0, fmt.Errorf("invalid DB value format")
	}

	rowid, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", "", "", 0, err
	}

	return parts[0], parts[1], parts[2], rowid, nil
}

func _SANode_readValueIntoDb[V string | int | float64 | bool](node *SANode) (V, error) {
	var ret V

	path, table, column, rowid, err := node.extractDBpath()
	if err != nil {
		return ret, err
	}

	//open
	db, _, err := node.app.base.ui.win.disk.OpenDb(path)
	if err != nil {
		node.SetError(err)
		return ret, err
	}

	//read
	db.Lock()
	err = db.ReadRow_unsafe(fmt.Sprintf("SELECT %s FROM %s WHERE rowid=?", column, table), rowid).Scan(&ret)
	db.Unlock()
	if err != nil {
		node.SetError(err)
		return ret, err
	}

	return ret, nil
}

func _SANode_writeValueIntoDb[V string | int | float64 | bool](node *SANode, val V) error {
	path, table, column, rowid, err := node.extractDBpath()
	if err != nil {
		return err
	}

	//open
	db, _, err := node.app.base.ui.win.disk.OpenDb(path)
	if err != nil {
		node.SetError(err)
		return err
	}

	//write
	db.Lock()
	_, err = db.Write_unsafe(fmt.Sprintf("UPDATE %s SET %s=? WHERE rowid=?", table, column), val, rowid)
	db.Unlock()
	if err != nil {
		node.SetError(err)
		return err
	}
	return nil
}

func UiEditbox_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrString("value", "")
	ghost := node.GetAttrString("ghost", "")
	align_v := node.GetAttrInt("align_v", 0)
	align_h := node.GetAttrInt("align_h", 0)
	enable := node.GetAttrBool("enable", true)
	multi_line := node.GetAttrBool("multi_line", false)
	multi_line_enter_finish := node.GetAttrBool("multi_line_enter_finish", false)
	line_wrapping := node.GetAttrBool("line_wrapping", true)
	formating := node.GetAttrBool("formating", true)
	temp_to_value := node.GetAttrBool("temp_to_value", false)

	db_value := node.GetAttrBool("db_value", false)
	if db_value {
		value, _ = _SANode_readValueIntoDb[string](node)
	}

	origValue := value
	editedValue, active, _, fnshd, _ := node.app.base.ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, Comp_editboxProp().Ghost(ghost).MultiLine(multi_line, line_wrapping).MultiLineEnterFinish(multi_line_enter_finish).Formating(formating).Enable(enable).Align(align_h, align_v))

	if temp_to_value && active {
		if node.Attrs["value"] != editedValue {
			_SANode_writeValueIntoDb(node, editedValue)

			node.SetChange(nil)
		}
		node.Attrs["value"] = editedValue
	}

	if fnshd && origValue != value {
		if db_value {
			_SANode_writeValueIntoDb(node, value)
		} else {
			node.Attrs["value"] = value
		}

		node.SetChange(nil)
	}
}

func UiCheckbox_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "value", false)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "db_value", false)
}

func UiCheckbox_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrBool("value", false)
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)

	db_value := node.GetAttrBool("db_value", false)
	if db_value {
		value, _ = _SANode_readValueIntoDb[bool](node)
	}

	if node.app.base.ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, tooltip, enable) {
		if db_value {
			_SANode_writeValueIntoDb(node, value)
		} else {
			node.Attrs["value"] = value
		}
		node.SetChange(nil)
	}
}

func UiSwitch_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "value", false)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "db_value", false)
}

func UiSwitch_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrBool("value", false)
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)

	db_value := node.GetAttrBool("db_value", false)
	if db_value {
		value, _ = _SANode_readValueIntoDb[bool](node)
	}

	if node.app.base.ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, tooltip, enable) {
		if db_value {
			_SANode_writeValueIntoDb(node, value)
		} else {
			node.Attrs["value"] = value
		}
		node.SetChange(nil)
	}
}

func UiSlider_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFloat(&grid, "value", 0, 3)
	node.ShowAttrFloat(&grid, "min", 0, 3)
	node.ShowAttrFloat(&grid, "max", 10, 3)
	node.ShowAttrFloat(&grid, "step", 0, 3)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "db_value", false)
}

func UiSlider_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrFloat("value", 0)
	min := node.GetAttrFloat("min", 0)
	max := node.GetAttrFloat("max", 10)
	step := node.GetAttrFloat("step", 0)
	enable := node.GetAttrBool("enable", true)

	db_value := node.GetAttrBool("db_value", false)
	if db_value {
		value, _ = _SANode_readValueIntoDb[float64](node)
	}

	if node.app.base.ui.Comp_slider(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, min, max, step, enable) {
		if db_value {
			_SANode_writeValueIntoDb(node, value)
		} else {
			node.Attrs["value"] = value
		}
		node.SetChange(nil)
	}
}

func UiCombo_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrString(&grid, "value", "", false)
	node.ShowAttrString(&grid, "options_names", "a;b;c", false)
	node.ShowAttrString(&grid, "options_values", "0;1;2", false)
	node.ShowAttrBool(&grid, "search", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "db_value", false)
}

func UiCombo_render(node *SANode) {
	grid := node.GetGrid()

	value := node.GetAttrString("value", "")
	opts_names := node.GetAttrString("options_names", "a;b;c")
	opts_values := node.GetAttrString("options_values", "0;1;2")
	search := node.GetAttrBool("search", false)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)

	db_value := node.GetAttrBool("db_value", false)
	if db_value {
		value, _ = _SANode_readValueIntoDb[string](node)
	}

	if node.app.base.ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, strings.Split(opts_names, ";"), strings.Split(opts_values, ";"), tooltip, enable, search) {
		if db_value {
			_SANode_writeValueIntoDb(node, value)
		} else {
			node.Attrs["value"] = value
		}
		node.SetChange(nil)
	}
}

func UiColor_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrCd(&grid, "value", OsCd{127, 127, 127, 255})
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
}

func UiColor_render(node *SANode) {
	grid := node.GetGrid()

	value := node.GetAttrCd("value", OsCd{127, 127, 127, 255})
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)

	if node.app.base.ui.comp_colorPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, "color_picker_"+node.Name, tooltip, enable) {
		node.SetAttrCd("value", value)
		node.SetChange(nil)
	}
}

func UiDivider_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrIntCombo(&grid, "type", 0, []string{"Horizontal", "Vertical"}, []string{"0", "1"})
}

func UiDivider_render(node *SANode) {
	grid := node.GetGrid()

	tp := node.GetAttrInt("type", 0)

	if tp == 0 {
		node.app.base.ui.Div_SpacerRow(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	} else {
		node.app.base.ui.Div_SpacerCol(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	}
}

func UiTimer_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFloat(&grid, "time_sec", 60, 2)
	node.ShowAttrFloat(&grid, "start_sec", 0, 2)
	node.ShowAttrBool(&grid, "repeat", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	//node.ShowAttrBool(&grid, "triggered", false)

	if ui.Comp_button(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, "Reset", Comp_buttonProp()) > 0 {
		node.Attrs["start_sec"] = OsTime()
	}
}

func UiTimer_render(node *SANode) {
	ui := node.app.base.ui
	grid := node.GetGrid()

	time_secs := node.GetAttrFloat("time_sec", 60)
	start_sec := node.GetAttrFloat("start_sec", 0)
	repeat := node.GetAttrBool("repeat", false)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true) //also STOP!
	//triggered := node.GetAttrBool("triggered", false)

	dt := OsTime() - start_sec
	prc := OsTrnFloat(enable, dt/time_secs, 0)

	//draw it
	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(1, 2)
		ui.Div_rowMax(0, 100)

		node.app.base.ui.Comp_progress(0, 0, 1, 1, prc, 1, tooltip, enable)

		if ui.Comp_buttonLight(1, 0, 1, 1, "Reset", Comp_buttonProp()) > 0 {
			node.Attrs["start_sec"] = OsTime()
			prc = 0
		}
	}
	ui.Div_end()

	if enable && prc >= 1 {
		if start_sec > 0 { //if repeat==false, set 'done' only once
			//node.Attrs["done"] = true

			node.SetChange([]SANodeCodeExePrm{{Node: node.Name, Attr: "triggered", Value: true}})
		}
		if repeat {
			node.Attrs["start_sec"] = OsTime()
		} else {
			node.Attrs["start_sec"] = 0
		}
	}
}

func UiDate_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrInt(&grid, "value", 0)
	node.ShowAttrBool(&grid, "show_time", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "db_value", false)
}

func UiDate_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrInt("value", 0)
	show_time := node.GetAttrBool("show_time", false)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)

	db_value := node.GetAttrBool("db_value", false)
	if db_value {
		value, _ = _SANode_readValueIntoDb[int](node)
	}

	date := int64(value)
	if node.app.base.ui.Comp_CalendarDatePicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &date, show_time, "date_"+node.Name, tooltip, enable) {
		if db_value {
			_SANode_writeValueIntoDb(node, int(date))
		} else {
			node.Attrs["value"] = int(date)
		}
		node.SetChange(nil)
	}
}

func UiDiskDir_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFilePicker(&grid, "path", "", false, true, "disk_dir_"+node.Name)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "enable", true)
}

func UiDiskDir_render(node *SANode) {
	grid := node.GetGrid()
	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)

	if node.app.base.ui.Comp_dirPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &path, false, true, "dir_picker_"+node.Name, enable) {
		node.Attrs["path"] = path
		node.SetChange(nil)
	}
}

func UiDiskFile_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFilePicker(&grid, "path", "", true, true, "disk_file_"+node.Name)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "enable", true)
}

func UiDiskFile_render(node *SANode) {
	grid := node.GetGrid()
	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)

	if node.app.base.ui.Comp_dirPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &path, true, true, "dir_picker_"+node.Name, enable) {
		node.Attrs["path"] = path
		node.SetChange(nil)
	}
}

func UiMicrophone_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFilePicker(&grid, "path", "", true, true, "microphone_path_"+node.Name)
	node.ShowAttrBool(&grid, "enable", true)
}

func UiMicrophone_render(node *SANode) {
	grid := node.GetGrid()
	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)

	nodePath := NewSANodePath(node)
	rec_active := node.app.IsMicNodeRecording(nodePath)

	cd := CdPalette_B
	if rec_active {
		cd = CdPalette_P
	}
	ui := node.app.base.ui
	icon := InitWinMedia_url("file:apps/base/resources/mic.png")
	if ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, "", Comp_buttonProp().Icon(&icon).Tooltip("Enable/Disable audio recording").ImgMargin(0.15).Cd(cd).Enable(enable).DrawBack(rec_active)) > 0 {

		if !rec_active {
			//start
			if ui.win.io.ini.MicOff {
				node.SetError(errors.New("microphone is disabled in SkyAlt Settings"))
				return
			}
			node.app.AddMicNode(nodePath)
		} else {
			//stop
			node.app.RemoveMicNode(nodePath)

			//make WAV file
			{
				//encode
				//file := &OsWriterSeeker{}
				file, err := os.Create(path)
				if err != nil {
					node.SetError(err)
					return
				}

				buff := node.temp_mic_data
				enc := wav.NewEncoder(file, buff.Format.SampleRate, buff.SourceBitDepth, buff.Format.NumChannels, 1)
				err = enc.Write(&buff)
				if err != nil {
					enc.Close()
					file.Close()
					node.SetError(err)
					return
				}
				enc.Close()
				file.Close()

				//save
				//node.Attrs["data"] = file.buf.Bytes()

				//reset
				node.temp_mic_data.Data = nil
			}

			//set finished
			node.SetChange([]SANodeCodeExePrm{{Node: node.Name, Attr: "triggered", Value: true}})
		}
	}
}

func UiNet_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrString(&grid, "url", "", false)
}

func UiLayout_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "enable", true)
}

func UiLayout_render(node *SANode) {
	grid := node.GetGrid()

	ui := node.app.base.ui

	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		node.renderLayout()
	}
	ui.Div_end()
}

func UiDialog_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "enable", true)
	//....
}

func UiDialog_render(node *SANode) {
	grid := node.GetGrid()

	ui := node.app.base.ui

	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	//....
	ui.Div_end()
}

func UiList_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrIntCombo(&grid, "direction", 0, []string{"Vertical", "Horizonal"}, []string{"0", "1"})
	node.ShowAttrFloat(&grid, "max_width", 100, 1)
	node.ShowAttrFloat(&grid, "max_height", 1, 1)
	node.ShowAttrBool(&grid, "show_border", true)

	options := []string{""} //1st is empty
	for _, nd := range node.Subs {
		if nd.IsTypeButton() {
			options = append(options, nd.Name)
		}
	}
	node.ShowAttrStringCombo(&grid, "selected_button", "", options, options)
	node.ShowAttrInt(&grid, "selected_index", -1)
}

func UiList_render(node *SANode) {
	grid := node.GetGrid()

	direction := node.GetAttrInt("direction", 0)
	max_width := node.GetAttrFloat("max_width", 100)
	max_height := node.GetAttrFloat("max_height", 1)
	show_border := node.GetAttrBool("show_border", true)
	//selected_button := node.GetAttrString("selected_button", "")
	//selected_index := node.GetAttrInt("selected_index", -1)

	num_rows := len(node.listSubs)

	ui := node.app.base.ui

	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		if direction == 0 {
			//vertical
			ui.Div_colMax(0, max_width)
			for y := 0; y < num_rows; y++ {
				ui.Div_rowMax(y, max_height)
			}

			//visible rows
			lv := ui.GetCall()
			row_st := lv.call.data.scrollV.GetWheel() / ui.win.Cell()
			row_en := row_st + OsRoundUp(float64(lv.call.crop.Size.Y)/float64(ui.win.Cell())) + 1

			row_st = OsMin(row_st, num_rows)
			row_en = OsMin(row_en, num_rows)

			//draw items
			for i := row_st; i < row_en; i++ {
				it := node.listSubs[i]
				gr := node.app.base.node_groups.FindNode(it.Exe)

				it.Cols = node.Cols
				it.Rows = node.Rows
				it.SetGrid(InitOsV4(0, i, 1, 1))
				gr.render(it)
			}

		} else {
			//horizontal
			ui.Div_rowMax(0, max_height)
			for y := 0; y < num_rows; y++ {
				ui.Div_colMax(y, max_width)
			}

			//visible rows
			lv := ui.GetCall()
			row_st := lv.call.data.scrollH.GetWheel() / ui.win.Cell()
			row_en := row_st + OsRoundUp(float64(lv.call.crop.Size.X)/float64(ui.win.Cell())) + 1

			row_st = OsMin(row_st, num_rows)
			row_en = OsMin(row_en, num_rows)

			//draw items
			for i := row_st; i < row_en; i++ {
				it := node.listSubs[i]
				gr := node.app.base.node_groups.FindNode(it.Exe)

				it.Cols = node.Cols
				it.Rows = node.Rows
				it.SetGrid(InitOsV4(i, 0, 1, 1))
				gr.render(it)
			}
		}

		if show_border {
			pl := ui.win.io.GetPalette()
			ui.Paint_rect(0, 0, 1, 1, 0, pl.OnB, 0.03)
		}

	}
	ui.Div_end()
}

func UiChart_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrString(&grid, "values", "[]", true)
	typee := node.ShowAttrStringCombo(&grid, "typee", "lines", []string{"Lines", "Columns"}, []string{"lines", "columns"})

	pl := ui.win.io.GetPalette()
	node.ShowAttrCd(&grid, "cd", pl.P)

	node.ShowAttrFloat(&grid, "left_margin", 1.5, 2)
	node.ShowAttrFloat(&grid, "bottom_margin", 1.0, 2)

	if typee == "lines" {
		node.ShowAttrFloat(&grid, "point_rad", 0.15, 2)
		node.ShowAttrFloat(&grid, "line_thick", 0.06, 2)
		node.ShowAttrString(&grid, "x_unit", "", false)
		node.ShowAttrString(&grid, "y_unit", "", false)
		node.ShowAttrBool(&grid, "bound_x0", true)
		node.ShowAttrBool(&grid, "bound_y0", true)

	}
	if typee == "columns" {
		node.ShowAttrFloat(&grid, "column_margin", 0.1, 2)
		node.ShowAttrString(&grid, "y_unit", "", false)
		node.ShowAttrBool(&grid, "bound_y0", true)
	}
}

func UiChart_render(node *SANode) {
	ui := node.app.base.ui

	grid := node.GetGrid()
	//enable := node.GetAttrBool("enable", true)	//...
	typee := node.GetAttrString("typee", "lines")
	values := node.GetAttrString("values", "[]")

	left_margin := node.GetAttrFloat("left_margin", 1.5)
	bottom_margin := node.GetAttrFloat("bottom_margin", 1)
	right_margin := OsMinFloat(left_margin, 1)
	top_margin := OsMinFloat(bottom_margin, 1)

	point_rad := node.GetAttrFloat("point_rad", 0.15)
	line_thick := node.GetAttrFloat("line_thick", 0.06)
	column_margin := node.GetAttrFloat("column_margin", 0.1)

	x_unit := node.GetAttrString("x_unit", "")
	y_unit := node.GetAttrString("y_unit", "")

	bound_x0 := node.GetAttrBool("bound_x0", true)
	bound_y0 := node.GetAttrBool("bound_y0", true)

	pl := ui.win.io.GetPalette()
	cdAxis := pl.GetGrey(0)
	cdAxisGrey := pl.GetGrey(0.8)
	cd := node.GetAttrCd("cd", pl.P)

	var items []UiLayoutChartItem
	err := json.Unmarshal([]byte(values), &items)
	if err != nil {
		node.SetError(err)
		return
	}

	//bound
	min, max := UiLayoutChart_getBound(items, bound_x0, bound_y0)

	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{

		ui.Div_colMax(0, 100)
		ui.Div_rowMax(0, 100)

		if typee == "lines" {
			//axis
			_UiLayoutChart_drawAxisX(min, max, left_margin, right_margin, top_margin, bottom_margin, cdAxis, cdAxisGrey, ui, true, true, x_unit)
			_UiLayoutChart_drawAxisY(min, max, left_margin, right_margin, top_margin, bottom_margin, cdAxis, cdAxisGrey, ui, true, true, y_unit)

			//values
			_UiLayoutChart_drawLines(items, min, max, left_margin, right_margin, top_margin, bottom_margin, cd, point_rad, line_thick, ui)

		} else if typee == "columns" {
			//axis
			_UiLayoutChart_drawAxisX(min, max, left_margin, right_margin, top_margin, bottom_margin, cdAxis, cdAxisGrey, ui, true, false, x_unit)
			_UiLayoutChart_drawAxisY(min, max, left_margin, right_margin, top_margin, bottom_margin, cdAxis, cdAxisGrey, ui, false, true, y_unit)

			_UiLayoutChart_drawAxisXlabels(items, left_margin, right_margin, top_margin, bottom_margin, ui)

			//values
			_UiLayoutChart_drawColumns(items, min, max, left_margin, right_margin, top_margin, bottom_margin, column_margin, cd, ui)
		}
	}
	ui.Div_end()

}

func UiSQLite_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "enable", true)

	path := node.ShowAttrFilePicker(&grid, "path", "", true, true, "render_sqlite_"+node.Name)
	if !OsFileExists(path) {
		node.SetError(errors.New("file not exist"))
		return
	}
	db, _, err := node.app.base.ui.win.disk.OpenDb(path)
	if err != nil {
		node.SetError(err)
		return
	}

	grid.Start.Y++ //space

	node.ShowAttrBool(&grid, "show_path", true)
	node.ShowAttrBool(&grid, "show_table_list", true)
	{
		info, err := db.GetTableInfo()
		if err != nil {
			node.SetError(err)
			return
		}
		tablesList := DiskDbIndex_ListOfTables(info)
		node.ShowAttrStringCombo(&grid, "selected_table", "", tablesList, tablesList)
	}

	grid.Start.Y++ //space

	//init
	{
		dnm := "db_init_" + node.Name
		if ui.Comp_buttonLight(grid.Start.X+1, grid.Start.Y, 1, 1, "Initalization", Comp_buttonProp()) > 0 {
			ui.Dialog_open(dnm, 1)
		}
		grid.Start.Y++

		if ui.Dialog_start(dnm) {
			ui.Div_colMax(0, 20)
			if ui.Comp_button(0, 0, 1, 1, "Generate 'init_sql'", Comp_buttonProp().Tooltip("Create SQL structure command from current database.")) > 0 {
				info, err := db.GetTableInfo()
				if err != nil {
					node.SetError(err)
					return
				}

				ini := ""
				for _, t := range info {
					ini += fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", t.Name)
					for _, c := range t.Columns {
						ini += fmt.Sprintf("%s %s, ", c.Name, c.Type)
					}
					ini, _ = strings.CutSuffix(ini, ", ")
					ini += ");\n"
				}

				node.Attrs["init_sql"] = ini
			}

			gr := InitOsV4(0, 1, 1, 1)
			init_sql := node.ShowAttrStringEx(&gr, "init_sql", "", true, false)

			gr.Start.Y++ //space

			if ui.Comp_button(0, gr.Start.Y, 1, 1, "Re-initialize", Comp_buttonProp().Enable(init_sql != "").Tooltip("Create tables & columns structure. Data are kept.")) > 0 {
				_, err := db.Write(init_sql)
				if err != nil {
					node.SetError(err)
					return
				}
			}

			ui.Dialog_end()
		}
	}

	grid.Start.Y++ //space

	//Maintenance
	if ui.Comp_buttonLight(grid.Start.X+1, grid.Start.Y, 1, 1, "Vacuum", Comp_buttonProp().Tooltip("Run database maintenance")) > 0 {
		db.Vacuum()
	}
	grid.Start.Y++
}

func UiSQLite_render(node *SANode) {
	grid := node.GetGrid()

	ui := node.app.base.ui
	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		UiSQLite_renderEditor(node)
	}
	ui.Div_end()
}

var g_table_name string
var g_column_name string
var g_column_type string

func UiSQLite_renderEditor(node *SANode) {
	path := node.GetAttrString("path", "")
	show_path := node.GetAttrBool("show_path", true)
	show_table_list := node.GetAttrBool("show_table_list", true)
	selected_table := node.GetAttrString("selected_table", "")

	db, _, err := node.app.base.ui.win.disk.OpenDb(path)
	if err != nil {
		node.SetError(err)
		return
	}

	info, err := db.GetTableInfo()
	if err != nil {
		node.SetError(err)
		return
	}

	var tinfo *DiskDbIndexTable
	for _, t := range info {
		if t.Name == selected_table {
			tinfo = t
		}
	}

	var num_rows int
	var rows *sql.Rows
	if tinfo != nil {
		db.Lock()
		defer db.Unlock()

		row := db.ReadRow_unsafe("SELECT COUNT(*) FROM " + selected_table)
		row.Scan(&num_rows)

		rows, err = db.Read_unsafe("SELECT " + tinfo.ListOfColumnNames(true) + " FROM " + selected_table)
		if err != nil {
			node.SetError(err)
			return
		}
		defer rows.Close()
	}

	{
		ui := node.app.base.ui
		ui.Div_colMax(0, 100)
		ui.Div_rowMax(OsTrn(show_path, 1, 0)+OsTrn(show_table_list, 1, 0), 100)

		y := 0
		if show_path {
			ui.Comp_text(0, y, 1, 1, path, 1)
			y++
		}

		//list of tables
		if show_table_list {
			ui.Div_start(0, y, 1, 1)
			{
				ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)
				ui.DivInfo_set(SA_DIV_SET_scrollHnarrow, 1, 0)

				for i := range info {
					ui.Div_col(1+i, 2)
					ui.Div_colMax(1+i, 100)
				}

				//add table
				{
					dnm := "create_table_" + node.Name
					if ui.Comp_button(0, 0, 1, 1, "+", Comp_buttonProp().Tooltip("Create table")) > 0 {
						ui.Dialog_open(dnm, 1)
						g_table_name = ""
					}
					if ui.Dialog_start(dnm) {
						ui.Div_colMax(0, 7)
						ui.Div_colMax(1, 4)
						//name
						ui.Comp_editbox(0, 0, 1, 1, &g_table_name, Comp_editboxProp().TempToValue(true))
						//button
						if ui.Comp_button(1, 0, 1, 1, "Create Table", Comp_buttonProp().Enable(g_table_name != "")) > 0 {
							db.Write_unsafe("CREATE TABLE " + g_table_name + "(firstColumn TEXT);")
							node.Attrs["selected_table"] = g_table_name
							ui.Dialog_close()
						}
						ui.Dialog_end()
					}
				}

				//list of tables
				for i, t := range info {

					ui.Div_start(1+i, 0, 1, 1)
					{
						ui.Div_colMax(0, 100)

						dnm := "detail_table" + t.Name + node.Name
						if ui.Comp_buttonMenu(0, 0, 1, 1, t.Name, t.Name == selected_table, Comp_buttonProp()) > 0 {
							node.Attrs["selected_table"] = t.Name
						}
						icon := InitWinMedia_url("file:apps/base/resources/context.png")
						if ui.Comp_button(1, 0, 1, 1, "", Comp_buttonProp().Icon(&icon).ImgMargin(0.3).DrawBack(t.Name == selected_table)) > 0 {
							ui.Dialog_open(dnm, 1)
							g_table_name = t.Name
						}

						if ui.Dialog_start(dnm) {
							ui.Div_colMax(0, 7)
							//rename
							_, _, _, fnshd, _ := ui.Comp_editbox_desc("Name", 0, 3, 0, 0, 1, 1, &g_table_name, Comp_editboxProp())
							if fnshd {
								db.Write_unsafe(fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", t.Name, g_table_name))
								if selected_table == t.Name {
									node.Attrs["selected_table"] = g_table_name
								}
							}
							//delete
							if ui.Comp_button(0, 2, 1, 1, "Delete", Comp_buttonProp().SetError(true).Confirmation("Are you sure?", "confirm_delete_table_"+t.Name)) > 0 {
								db.Write_unsafe(fmt.Sprintf("DROP TABLE %s;", t.Name))

								//select close table
								if selected_table == t.Name {
									nm := ""
									if i+1 < len(info) {
										nm = info[i+1].Name //next
									} else if i > 0 {
										nm = info[i-1].Name //previous
									}
									node.Attrs["selected_table"] = nm
								}
							}
							ui.Dialog_end()
						}
					}
					ui.Div_end()
				}
			}
			ui.Div_end()
			y++
		}

		if tinfo != nil {
			//table(column+rows)
			ui.Div_start(0, y, 1, 1)
			{
				for i := range tinfo.Columns {
					ui.Div_colMax(i, float64(OsTrn(i == 0, 1, 10))) //rowid size is 1
				}
				ui.Div_col(len(tinfo.Columns), 0.5) //extra empty
				ui.Div_rowMax(1, 100)

				//add column
				{
					dnm := "create_column" + node.Name
					if ui.Comp_buttonLight(0, 0, 1, 1, "+", Comp_buttonProp().Tooltip("Create column")) > 0 {
						ui.Dialog_open(dnm, 1)
						g_column_name = ""
						g_column_type = "TEXT"
					}
					if ui.Dialog_start(dnm) {
						ui.Div_colMax(0, 7)
						ui.Div_colMax(1, 4)
						ui.Div_colMax(2, 4)
						//name
						ui.Comp_editbox(0, 0, 1, 1, &g_column_name, Comp_editboxProp().TempToValue(true))
						//type
						ui.Comp_combo(1, 0, 1, 1, &g_column_type, []string{"INTEGER", "REAL", "TEXT", "BLOB", "NUMERIC"}, []string{"INTEGER", "REAL", "TEXT", "BLOB", "NUMERIC"}, "Type", true, false)
						//button
						if ui.Comp_button(2, 0, 1, 1, "Create Column", Comp_buttonProp().Enable(g_column_name != "")) > 0 {

							def := "" //blob
							if strings.EqualFold(g_column_type, "TEXT") {
								def = "NOT NULL DEFAULT \"\"" //text
							} else if !strings.EqualFold(g_column_type, "BLOB") {
								def = "NOT NULL DEFAULT 0" //number
							}

							db.Write_unsafe(fmt.Sprintf("ALTER TABLE %s ADD %s %s %s;", tinfo.Name, g_column_name, g_column_type, def))
							ui.Dialog_close()
						}
						ui.Dialog_end()
					}
				}

				//list of columns
				for i, c := range tinfo.Columns {
					if i == 0 {
						continue //skip rowid
					}
					dnm := "column_detail" + c.Name + node.Name
					if ui.Comp_buttonLight(i, 0, 1, 1, fmt.Sprintf("%s(%s)", c.Name, c.Type), Comp_buttonProp()) > 0 {
						ui.Dialog_open(dnm, 1)
						g_column_name = c.Name
					}
					if ui.Dialog_start(dnm) {
						ui.Div_colMax(0, 7)
						//rename
						_, _, _, fnshd, _ := ui.Comp_editbox_desc("Name", 0, 3, 0, 0, 1, 1, &g_column_name, Comp_editboxProp())
						if fnshd {
							db.Write_unsafe(fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;", tinfo.Name, c.Name, g_column_name))
						}
						//delete
						if ui.Comp_button(0, 2, 1, 1, "Delete", Comp_buttonProp().SetError(true).Confirmation("Are you sure?", "confirm_delete_column_"+tinfo.Name+"_"+c.Name)) > 0 {
							db.Write_unsafe(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tinfo.Name, c.Name))
						}
						ui.Dialog_end()
					}
				}

				parentId := ui.DivInfo_get(SA_DIV_GET_uid, 0)
				ui.Div_start(0, 1, len(tinfo.Columns)+1, 1)
				{
					//copy cols from parent
					ui.DivInfo_set(SA_DIV_SET_copyCols, parentId, 0)
					ui.DivInfo_set(SA_DIV_SET_scrollOnScreen, 1, 0)
					ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

					//visible rows
					lv := ui.GetCall()
					row_st := lv.call.data.scrollV.GetWheel() / ui.win.Cell()
					row_en := row_st + OsRoundUp(float64(lv.call.crop.Size.Y)/float64(ui.win.Cell()))
					if num_rows > 0 {
						ui.Div_row(num_rows-1, 1) //set last
					}

					//prepare for scan()
					vals := make([][]byte, len(tinfo.Columns))
					valsPtrs := make([]interface{}, len(vals))
					for i := range vals {
						valsPtrs[i] = &vals[i]
					}

					r := 0
					for rows.Next() && r <= row_en {
						if r < row_st {
							r++
							continue //skip invisible rows
						}

						err := rows.Scan(valsPtrs...) //err ...
						if err != nil {
							continue
						}

						//rowid + detail dialog
						rowid := string(vals[0])
						{
							dnm := "row_detail_" + node.Name + selected_table + rowid
							if ui.Comp_buttonLight(0, r, 1, 1, rowid, Comp_buttonProp()) > 0 {
								ui.Dialog_open(dnm, 1)
							}
							if ui.Dialog_start(dnm) {
								ui.Div_colMax(0, 7)
								if ui.Comp_button(0, 0, 1, 1, "Delete", Comp_buttonProp().SetError(true).Confirmation("Are you sure?", "confirm_delete_row_"+tinfo.Name+"_"+rowid)) > 0 {
									db.Write_unsafe(fmt.Sprintf("DELETE FROM %s WHERE rowid=?", tinfo.Name), rowid)
								}
								ui.Dialog_end()
							}
						}

						//cells
						for c := 1; c < len(vals); c++ {
							if strings.EqualFold(tinfo.Columns[c].Type, "blob") {
								ui.Comp_text(c, r, 1, 1, "<blob>", 1)
								//image ......
							} else {
								_, _, _, fnshd, _ := ui.Comp_editbox(c, r, 1, 1, &vals[c], Comp_editboxProp())
								if fnshd {
									db.Write_unsafe(fmt.Sprintf("UPDATE %s SET %s=? WHERE rowid=?", tinfo.Name, tinfo.Columns[c].Name), string(vals[c]), rowid)
								}
							}
						}
						r++
					}
				}
				ui.Div_end()
			}
			ui.Div_end()
			y++

			//+row
			ui.Div_start(0, y, 1, 1)
			{
				ui.Div_colMax(0, 3)
				if ui.Comp_button(0, 0, 1, 1, "Add row", Comp_buttonProp()) > 0 {
					db.Write_unsafe(fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s);", tinfo.Name, tinfo.ListOfColumnNames(false), tinfo.ListOfColumnValues(false)))
				}
			}
			ui.Div_end()
			y++
		}

		//rect around
		pl := ui.win.io.GetPalette()
		ui.Paint_rect(0, 0, 1, 1, 0, pl.OnB, 0.03)
	}
}

func UiCode_AttrChat(node *SANode) {
	ui := node.app.base.ui

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(1, 100)

	//header
	ui.Div_start(0, 0, 1, 1)
	{
		ui.Div_colMax(0, 100)

		ui.Comp_text(0, 0, 1, 1, "Code chat", 1)

		dnm := "chat_" + node.Name
		if ui.Comp_buttonIcon(1, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/context.png"), 0.3, "", Comp_buttonProp().Cd(CdPalette_B)) > 0 {
			ui.Dialog_open(dnm, 1)
		}
		if ui.Dialog_start(dnm) {
			ui.Div_colMax(0, 5)
			y := 0

			if ui.Comp_buttonMenu(0, y, 1, 1, "Clear all", false, Comp_buttonProp().Confirmation("Are you sure?", "clear_chat")) > 0 {
				node.Code.Messages = nil
				node.Code.CheckLastChatEmpty()
				ui.Dialog_close()
			}
			y++

			ui.Dialog_end()
		}
	}
	ui.Div_end()

	ui.Div_start(0, 1, 1, 1)
	{
		ui.Div_colMax(0, 1.5)
		ui.Div_colMax(1, 100)

		node.Code.CheckLastChatEmpty()

		//set rows heights
		y := 0
		for i, str := range node.Code.Messages {
			isLast := (i+1 >= len(node.Code.Messages))
			isAssistRunning := (node.Code.job_oai != nil && node.Code.job_oai_index == i)

			//user
			{
				line_wrapping := true
				nlines := 1
				dd := ui.GetCall().call.FindFromGridPos(OsV2{1, y})
				if dd != nil {
					nlines = OsRoundUp(float64(ui._Paint_getMaxCellSize(dd, str.User, InitWinFontPropsDef(ui.win), true, line_wrapping).Y))
				}
				nlines = OsMax(2, nlines)

				ui.Div_row(y, float64(nlines))
				y++
			}

			y++ //model, button

			//y++ //space

			if !isLast || isAssistRunning {
				assist := str.Assistent
				if isAssistRunning {
					assist = node.Code.job_oai.wip_answer
					if node.Code.job_oai.done.Load() {
						//save
						node.Code.Messages[node.Code.job_oai_index].Assistent = string(node.Code.job_oai.output)
						node.Code.job_oai = nil
						node.Code.job_oai_index = -1
					}
				}

				line_wrapping := false
				nlines := 1
				dd := ui.GetCall().call.FindFromGridPos(OsV2{1, y})
				if dd != nil {
					sz := ui._Paint_getMaxCellSize(dd, assist, InitWinFontPropsDef(ui.win), true, line_wrapping)
					nlines = OsRoundUp(float64(sz.Y + 0.5))
				}
				ui.Div_row(y, float64(nlines))
				y++

				y++ //"Use this code"
			}

			y++ //space
		}

		//add UI
		y = 0
		for i := 0; i < len(node.Code.Messages); i++ {
			str := node.Code.Messages[i]
			//for i, str := range node.Code.Messages {
			isLast := (i+1 >= len(node.Code.Messages))
			isAssistRunning := (node.Code.job_oai != nil && node.Code.job_oai_index == i)

			//user
			{
				line_wrapping := true
				ui.Comp_textAlign(0, y, 1, 1, "User", 0, 0)
				ui.Comp_editbox(1, y, 1, 1, &node.Code.Messages[i].User, Comp_editboxProp().Align(0, 0).MultiLine(true, line_wrapping).TempToValue(true))
				y++
			}

			ui.Div_start(1, y, 1, 1)
			{
				ui.Div_colMax(0, 100)
				ui.Div_colMax(1, 4)
				ui.Div_colMax(2, 5)
				//ui.Div_colMax(3, 1)

				//model
				models := []string{"gpt-4-turbo", "gpt-3.5-turbo"} //https://platform.openai.com/docs/models/
				ui.Comp_combo(1, 0, 1, 1, &node.app.base.ui.win.io.ini.ChatModel, models, models, "Model", true, true)

				//send
				if ui.Comp_buttonLight(2, 0, 1, 1, OsTrnString(isLast, "Send", "Re-send"), Comp_buttonProp()) > 0 {
					node.Code.GetAnswer(i)
				}

				//delete
				if ui.Comp_buttonLight(3, 0, 1, 1, "X", Comp_buttonProp().Confirmation("Are you sure?", "delete_chat_item_"+strconv.Itoa(i))) > 0 {
					node.Code.Messages = append(node.Code.Messages[:i], node.Code.Messages[i+1:]...) //remove
					node.Code.CheckLastChatEmpty()
				}
			}
			ui.Div_end()
			y++

			//y++ //space

			//answer
			if !isLast || isAssistRunning {
				assist := str.Assistent
				if isAssistRunning {
					assist = node.Code.job_oai.wip_answer
				}

				{
					line_wrapping := false

					//background
					ui.Div_start(0, y, 2, 2)
					pl := ui.win.io.GetPalette()
					ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.85), 0)
					ui.Div_end()

					ui.Comp_textAlign(0, y, 1, 1, "Bot", 0, 0)
					ui.Comp_textSelectMulti(1, y, 1, 1, assist, OsV2{0, 0}, true, false, false, line_wrapping)
					y++

					ui.Div_start(1, y, 1, 1)
					{
						ui.Div_colMax(0, 100)
						ui.Div_colMax(1, 5)
						ui.Div_colMax(2, 5)
						if ui.Comp_buttonLight(1, 0, 1, 1, "Copy code", Comp_buttonProp().Enable(!isAssistRunning)) > 0 {
							node.Code.CopyCodeToClipboard(str.Assistent)
						}
						if ui.Comp_buttonLight(2, 0, 1, 1, "Use this code", Comp_buttonProp().Enable(!isAssistRunning)) > 0 {
							node.Code.UseCodeFromAnswer(str.Assistent)
						}
					}
					ui.Div_end()
					y++
				}
			}

			ui.Div_SpacerRow(0, y, 2, 1)
			y++ //space
		}
	}
	ui.Div_end()
}

func UiCode_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	_UiCode_attrs(node, &grid)

}
func _UiCode_attrs(node *SANode, grid *OsV4) {
	ui := node.app.base.ui

	ui.Div_rowMax(1, 100)                   //code
	ui.Div_rowResize(3, "output", 2, false) //output

	//bypass
	node.ShowAttrBool(grid, "bypass", false)

	//Code
	{
		ui.Comp_textAlign(0, 1, 1, 1, "Code", 0, 0)

		_, _, _, fnshd, _ := ui.Comp_editbox(1, 1, 1, 1, &node.Code.Code, Comp_editboxProp().Align(0, 0).MultiLine(true, false).Formating(false).TempToValue(true))
		if fnshd {
			node.Code.UpdateFile()
		}

		//run button
		if ui.Comp_button(1, 2, 1, 1, "Run", Comp_buttonProp()) > 0 {
			node.Code.Execute(nil)
		}
	}

	//output
	{
		ui.Comp_textAlign(0, 3, 1, 1, "Output", 0, 0)
		ui.Comp_textSelectMulti(1, 3, 1, 1, node.Code.cmd_output, OsV2{0, 0}, true, true, false, false)
	}
}

var g_whisper_formats = []string{"verbose_json", "json", "text", "srt", "vtt"}
var g_whisper_modelList = []string{"ggml-tiny.en", "ggml-tiny", "ggml-base.en", "ggml-base", "ggml-small.en", "ggml-small", "ggml-medium.en", "ggml-medium", "ggml-large-v1", "ggml-large-v2", "ggml-large-v3"}
var g_whisper_modelsFolder = "services/whisper.cpp/models/"

func UiWhisperCpp_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	ui.Div_start(0, 0, 2, 1)
	{
		//build model list
		var models []string
		for _, m := range g_whisper_modelList {
			if OsFileExists(filepath.Join(g_whisper_modelsFolder, m+".bin")) {
				models = append(models, m)
			}
		}

		ui.Div_colMax(0, 3)
		ui.Div_colMax(1, 100)
		ui.Div_colMax(2, 5)
		node.ShowAttrStringCombo(&grid, "model", OsTrnString(len(models) > 0, models[0], ""), models, models)
		ui.Comp_text(2, 0, 1, 1, "Use Whisper_downloader App", 0) //Launch it ...
		//if ui.Comp_buttonLight(2, 0, 1, 1, "Download", "", true) > 0 {
		//	ui.Dialog_open("models", 0)
		//}
	}
	ui.Div_end()

	node.ShowAttrInt(&grid, "offset_t", 0)
	node.ShowAttrInt(&grid, "offset_n", 0)
	node.ShowAttrInt(&grid, "duration", 0)
	node.ShowAttrInt(&grid, "max_context", -1)
	node.ShowAttrInt(&grid, "max_len", 0)
	node.ShowAttrInt(&grid, "best_of", 2)
	node.ShowAttrInt(&grid, "beam_size", -1)

	node.ShowAttrFloat(&grid, "word_thold", 0.01, 3)
	node.ShowAttrFloat(&grid, "entropy_thold", 2.4, 3)
	node.ShowAttrFloat(&grid, "logprob_thold", -1, 3)

	node.ShowAttrBool(&grid, "translate", false)
	node.ShowAttrBool(&grid, "diarize", false)
	node.ShowAttrBool(&grid, "tinydiarize", false)
	node.ShowAttrBool(&grid, "split_on_word", false)
	node.ShowAttrBool(&grid, "no_timestamps", false)

	node.ShowAttrString(&grid, "language", "", false)
	node.ShowAttrBool(&grid, "detect_language", false)

	node.ShowAttrFloat(&grid, "temperature", 0, 3)
	node.ShowAttrFloat(&grid, "temperature_inc", 0.2, 3)

	node.ShowAttrStringCombo(&grid, "response_format", "verbose_json", g_whisper_formats, g_whisper_formats)
}

var g_llama_modelsFolder = "services/llama.cpp/models/"

func UiLLamaCpp_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	ui.Div_start(0, 0, 2, 1)
	{
		//build model list
		var models []string
		modelFiles := OsFileListBuild(g_llama_modelsFolder, "", true)
		for _, m := range modelFiles.Subs {
			if !m.IsDir && !strings.HasPrefix(m.Name, "ggml-vocab") && !strings.HasSuffix(m.Name, ".temp") {
				models = append(models, m.Name)
			}
		}

		ui.Div_colMax(0, 3)
		ui.Div_colMax(1, 100)
		ui.Div_colMax(2, 3)
		node.ShowAttrStringCombo(&grid, "model", OsTrnString(len(models) > 0, models[0], ""), models, models)
		ui.Comp_text(2, 0, 1, 1, "Use LLama_downloader App", 0) //Launch it ...
		//if ui.Comp_buttonLight(2, 0, 1, 1, "Download", "", true) > 0 {
		//	ui.Dialog_open("models", 0)
		//}
	}
	ui.Div_end()

	//...
	/*stopAttr := node.GetAttr("stop", []byte(`["</s>", "Llama:", "User:"]`))
	err := json.Unmarshal(stopAttr.GetBlob().data, &props.Stop)
	if err != nil {
		stopAttr.SetError(err)
	}*/

	node.ShowAttrInt(&grid, "seed", -1)
	node.ShowAttrInt(&grid, "n_predict", 400)

	node.ShowAttrFloat(&grid, "temperature", 0.8, 3)
	node.ShowAttrFloat(&grid, "dynatemp_range", 0.0, 3)
	node.ShowAttrFloat(&grid, "dynatemp_exponent", 1.0, 3)
	node.ShowAttrInt(&grid, "repeat_last_n", 256)
	node.ShowAttrFloat(&grid, "repeat_penalty", 1.18, 3)

	node.ShowAttrInt(&grid, "top_k", 40)
	node.ShowAttrFloat(&grid, "top_p", 0.5, 3)
	node.ShowAttrFloat(&grid, "min_p", 0.05, 3)
	node.ShowAttrFloat(&grid, "tfs_z", 1.0, 3)
	node.ShowAttrFloat(&grid, "typical_p", 1.0, 3)
	node.ShowAttrFloat(&grid, "presence_penalty", 0.0, 3)
	node.ShowAttrFloat(&grid, "frequency_penalty", 0.0, 3)
	node.ShowAttrBool(&grid, "mirostat", false)
	node.ShowAttrFloat(&grid, "mirostat_tau", 5, 3)
	node.ShowAttrFloat(&grid, "mirostat_eta", 0.1, 3)
	//Grammar
	node.ShowAttrInt(&grid, "n_probs", 0)
	//Image_data
	node.ShowAttrBool(&grid, "cache_prompt", false)
	node.ShowAttrInt(&grid, "slot_id", -1)
}

var g_oia_modelList = []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo-preview"}

func UiOpenAI_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrStringCombo(&grid, "model", g_oia_modelList[0], g_oia_modelList, g_oia_modelList)
	//more ...............

	if node.app.base.ui.win.io.ini.OpenAI_key == "" {
		node.SetError(fmt.Errorf("openAI API key is not set. Fill it in Menu:Settings"))
	}
}

func UiMap_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "enable", true)

	node.ShowAttrFloat(&grid, "lon", 14.4071117049, -1)
	node.ShowAttrFloat(&grid, "lat", 50.0852013259, -1)
	node.ShowAttrFloat(&grid, "zoom", 5, -1)

	node.ShowAttrString(&grid, "file", "temp/maps/osm.sqlite", false)
	node.ShowAttrString(&grid, "url", "https://tile.openstreetmap.org/{z}/{x}/{y}.png", false)
	node.ShowAttrString(&grid, "copyright", "(c)OpenStreetMap contributors", false)
	node.ShowAttrString(&grid, "copyright_url", "https://www.openstreetmap.org/copyright", false)

	node.ShowAttrString(&grid, "locators", "", true)
	node.ShowAttrCd(&grid, "locators_cd", OsCd{50, 50, 200, 255})

	node.ShowAttrString(&grid, "segments", "", true)
	node.ShowAttrCd(&grid, "segments_cd", OsCd{200, 50, 50, 255})

}

func UiMap_render(node *SANode) {
	ui := node.app.base.ui

	grid := node.GetGrid()
	//enable := node.GetAttrBool("enable", true)	//......

	lon := node.GetAttrFloat("lon", 14.4071117049)
	lat := node.GetAttrFloat("lat", 50.0852013259)
	zoom := node.GetAttrFloat("zoom", 5)

	file := node.GetAttrString("file", "temp/maps/osm.sqlite")
	url := node.GetAttrString("url", "https://tile.openstreetmap.org/{z}/{x}/{y}.png")
	copyright := node.GetAttrString("copyright", "(c)OpenStreetMap contributors")
	copyright_url := node.GetAttrString("copyright_url", "https://www.openstreetmap.org/copyright")

	locators := node.GetAttrString("locators", "")   //`[{"label":"Example Title", "lon":14.4071117049, "lat":50.0852013259}, {"label":"2", "lon":14, "lat":50}]`
	segmentsIn := node.GetAttrString("segments", "") //`[{"label":"Example Title", "Trkpt":[{"lat":50,"lon":16,"ele":400,"time":"2020-04-15T09:05:20Z"},{"lat":50.4,"lon":16.1,"ele":400,"time":"2020-04-15T09:05:23Z"}]}]`
	segmentsIn = strings.TrimSpace(segmentsIn)

	locators_cd := node.GetAttrCd("locators_cd", OsCd{50, 50, 200, 255})
	segments_cd := node.GetAttrCd("segments_cd", OsCd{200, 50, 50, 255})

	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		ui.Div_colMax(0, 100)
		ui.Div_rowMax(0, 100)

		//map
		changed, err := ui.comp_map(&lon, &lat, &zoom, file, url, copyright, copyright_url)
		if err != nil {
			node.SetError(err)
		}
		if changed {
			//set back
			node.Attrs["lon"] = lon
			node.Attrs["lat"] = lat
			node.Attrs["zoom"] = zoom
		}

		//locators
		if locators != "" {
			var items []UiCompMapLocator
			err := json.Unmarshal([]byte(locators), &items)
			if err == nil {
				err = ui.comp_mapLocators(lon, lat, zoom, items, locators_cd, "map_"+node.Name)
				if err != nil {
					node.SetError(fmt.Errorf("comp_mapLocators() failed: %w", err))
				}
			} else {
				node.SetError(fmt.Errorf("locators Unmarshal() failed: %w", err))
			}
		}

		//segments
		if segmentsIn != "" {
			seg, err := ui.mapp.GetSegment(segmentsIn)

			if err == nil {
				err = ui.comp_mapSegments(lon, lat, zoom, seg.items, segments_cd) //slow ........ cache + redraw when zoom ....
				if err != nil {
					node.SetError(fmt.Errorf("comp_mapSegments() failed: %w", err))
				}
			} else {
				node.SetError(err)
			}
		}
	}
	ui.Div_end()
}

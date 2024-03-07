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
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func UiButton_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "clicked", false)
}

func UiButton_render(node *SANode) {
	grid := node.GetGrid()
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//clicked := node.GetAttrBool("clicked", false)

	if node.app.base.ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, tooltip, enable) > 0 {
		node.Attrs["clicked"] = true
	}
}

func UiText_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrString(&grid, "label", "", node.GetAttrBool("multi_line", false))
	node.ShowAttrIntCombo(&grid, "align_h", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrIntCombo(&grid, "align_v", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrBool(&grid, "multi_line", false)
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

	if node.GetAttrBool("multi_line", false) {
		node.app.base.ui.Comp_textSelectMulti(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, show_border)
	} else {
		node.app.base.ui.Comp_textSelect(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, show_border)
	}
}

func UiEditbox_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
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
	node.ShowAttrBool(&grid, "finished", false)
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
	//finished := node.GetAttrBool("finished", false)

	_, _, _, fnshd, _ := node.app.base.ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, Comp_editboxProp().Ghost(ghost).MultiLine(multi_line).MultiLineEnterFinish(multi_line_enter_finish).Enable(enable).Align(align_h, align_v))
	if fnshd {
		node.Attrs["value"] = value
		node.Attrs["finished"] = true
	}
}

func UiCheckbox_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "value", false)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiCheckbox_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrBool("value", false)
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, tooltip, enable) {
		node.Attrs["value"] = value
		node.Attrs["changed"] = true
	}
}

func UiSwitch_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "value", false)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiSwitch_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrBool("value", false)
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, tooltip, enable) {
		node.Attrs["value"] = value
		node.Attrs["changed"] = true
	}
}

func UiSlider_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFloat(&grid, "value", 0, 3)
	node.ShowAttrFloat(&grid, "min", 0, 3)
	node.ShowAttrFloat(&grid, "max", 10, 3)
	node.ShowAttrFloat(&grid, "step", 0, 3)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiSlider_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrFloat("value", 0)
	min := node.GetAttrFloat("min", 0)
	max := node.GetAttrFloat("max", 10)
	step := node.GetAttrFloat("step", 0)
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_slider(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, min, max, step, enable) {
		node.Attrs["value"] = value
		node.Attrs["changed"] = true
	}
}

func UiCombo_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrString(&grid, "value", "", false)
	node.ShowAttrString(&grid, "options_names", "a;b;c", false)
	node.ShowAttrString(&grid, "options_value", "0;1;2", false)
	node.ShowAttrBool(&grid, "search", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiCombo_render(node *SANode) {
	grid := node.GetGrid()

	value := node.GetAttrString("value", "")
	opts_names := node.GetAttrString("options_names", "a;b;c")
	opts_values := node.GetAttrString("options_value", "0;1;2")
	search := node.GetAttrBool("search", false)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	options_names := strings.Split(opts_names, ";")
	options_values := strings.Split(opts_values, ";")

	if node.app.base.ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, options_names, options_values, tooltip, enable, search) {
		node.Attrs["value"] = value
		node.Attrs["changed"] = true
	}
}

func UiColor_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrV4(&grid, "value", InitOsV4(127, 127, 127, 255))
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiColor_render(node *SANode) {
	grid := node.GetGrid()

	value := node.GetAttrV4("value", InitOsV4(127, 127, 127, 255))
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	cd := OsCd{R: byte(value.Start.X), G: byte(value.Start.Y), B: byte(value.Size.X), A: 255}
	if node.app.base.ui.comp_colorPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &cd, "color_picker_"+node.Name, tooltip, enable) {
		node.SetAttrV4("value", InitOsV4(int(cd.R), int(cd.G), int(cd.B), value.Size.Y))
		node.Attrs["changed"] = true
	}
}

func UiDivider_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
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
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFloat(&grid, "time_sec", 60, 2)
	node.ShowAttrFloat(&grid, "start_sec", 0, 2)
	node.ShowAttrBool(&grid, "repeat", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "done", false)

	if ui.Comp_button(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, "Reset", "", true) > 0 {
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
	//done := node.GetAttrBool("done", false)

	dt := OsTime() - start_sec
	prc := OsTrnFloat(enable, dt/time_secs, 0)

	//draw it
	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(1, 2)
		ui.Div_rowMax(0, 100)

		node.app.base.ui.Comp_progress(0, 0, 1, 1, prc, 1, tooltip, enable)

		if ui.Comp_buttonLight(1, 0, 1, 1, "Reset", "", true) > 0 {
			node.Attrs["start_sec"] = OsTime()
			prc = 0
		}
	}
	ui.Div_end()

	if enable && prc >= 1 {
		if start_sec > 0 { //if repeat==false, set 'done' only once
			node.Attrs["done"] = true
		}
		if repeat {
			node.Attrs["start_sec"] = OsTime()
		} else {
			node.Attrs["start_sec"] = 0
		}
	}
}

func UiDiskDir_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrFilePicker(&grid, "path", "", false, "disk_dir_"+node.Name)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiDiskDir_render(node *SANode) {
	grid := node.GetGrid()
	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_dirPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &path, false, "dir_picker_"+node.Name, enable) {
		node.Attrs["path"] = path
		node.Attrs["changed"] = true
	}
}

func UiDiskFile_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrFilePicker(&grid, "path", "", true, "disk_file_"+node.Name)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiDiskFile_render(node *SANode) {
	grid := node.GetGrid()
	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_dirPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &path, true, "dir_picker_"+node.Name, enable) {
		node.Attrs["path"] = path
		node.Attrs["changed"] = true
	}
}

func UiSQLite_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	path := node.ShowAttrFilePicker(&grid, "path", "", true, "render_sqlite_"+node.Name)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "changed", false) //...

	if !OsFileExists(path) {
		node.SetError(errors.New("file not exist"))
		return
	}

	db, _, err := node.app.base.ui.win.disk.OpenDb(path)
	if err != nil {
		node.SetError(err)
		return
	}

	if ui.Comp_button(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, "Vacuum", "Run database maintenance", true) > 0 {
		db.Vacuum()
	}
	grid.Start.Y++

	if ui.Comp_button(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, "Generate 'init_sql'", "Create SQL structure command from current database.", true) > 0 {
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
	grid.Start.Y++

	init_sql := node.ShowAttrString(&grid, "init_sql", "", true)
	if ui.Comp_button(grid.Start.X+1, grid.Start.Y, grid.Size.X, grid.Size.Y, "Re-initialize", "Create tables & columns structure. Data are kept.", init_sql != "") > 0 {
		_, err := db.Write(init_sql)
		if err != nil {
			node.SetError(err)
			return
		}
	}
	grid.Start.Y++
}

var g_table_name string
var g_column_name string
var g_column_type string

func UiSQLite_render(node *SANode) {
	grid := node.GetGrid()

	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)
	selected_table := node.GetAttrString("selected_table", "")
	//changed := node.GetAttrBool("changed", false)

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

		rows, err = db.Read_unsafe("SELECT rowid, " + tinfo.ListOfColumnNames() + " FROM " + selected_table)
		if err != nil {
			node.SetError(err)
			return
		}
		defer rows.Close()
	}

	ui := node.app.base.ui
	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		ui.Div_colMax(0, 100)
		ui.Div_rowMax(2, 100)

		if ui.Comp_dirPicker(0, 0, 1, 1, &path, true, "dir_picker_"+node.Name, enable) {
			node.Attrs["path"] = path
			node.Attrs["changed"] = true
		}

		//list of tables
		ui.Div_start(0, 1, 1, 1)
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
				if ui.Comp_button(0, 0, 1, 1, "+", "Create table", true) > 0 {
					ui.Dialog_open(dnm, 1)
					g_table_name = ""
				}
				if ui.Dialog_start(dnm) {
					ui.Div_colMax(0, 7)
					ui.Div_colMax(1, 4)
					//name
					ui.Comp_editbox(0, 0, 1, 1, &g_table_name, Comp_editboxProp().TempToValue(true))
					//button
					if ui.Comp_button(1, 0, 1, 1, "Create Table", "", g_table_name != "") > 0 {
						db.Write_unsafe("CREATE TABLE " + g_table_name + "(firstColumn TEXT);")
						ui.Dialog_close()
					}
					ui.Dialog_end()
				}
			}

			//list of tables
			for i, t := range info {

				dnm := "detail_table" + t.Name + node.Name
				cl := ui.Comp_buttonMenu(1+i, 0, 1, 1, t.Name, "", true, t.Name == selected_table)
				if cl == 1 {
					node.Attrs["selected_table"] = t.Name
				} else if cl == 2 {
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
					if ui.Comp_buttonError(0, 2, 1, 1, "Delete", "", true, true) > 0 {
						db.Write_unsafe(fmt.Sprintf("DROP TABLE %s;", t.Name))
					}
					ui.Dialog_end()
				}
			}
		}
		ui.Div_end()

		if tinfo != nil {
			//table(column+rows)
			ui.Div_start(0, 2, 1, 1)
			{
				for i := range tinfo.Columns {
					ui.Div_colMax(1+i, 10)
				}
				ui.Div_col(1+len(tinfo.Columns), 0.5) //extra empty
				ui.Div_rowMax(1, 100)

				//add column
				{
					dnm := "create_column" + node.Name
					if ui.Comp_buttonLight(0, 0, 1, 1, "+", "Create column", true) > 0 {
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
						if ui.Comp_button(2, 0, 1, 1, "Create Column", "", g_column_name != "") > 0 {
							db.Write_unsafe(fmt.Sprintf("ALTER TABLE %s ADD %s %s;", tinfo.Name, g_column_name, g_column_type))
							ui.Dialog_close()
						}
						ui.Dialog_end()
					}
				}

				//list of columns
				for i, c := range tinfo.Columns {
					dnm := "column_detail" + c.Name + node.Name
					if ui.Comp_buttonLight(1+i, 0, 1, 1, fmt.Sprintf("%s(%s)", c.Name, c.Type), "", true) > 0 {
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
						if ui.Comp_buttonError(0, 2, 1, 1, "Delete", "", true, true) > 0 {
							db.Write_unsafe(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tinfo.Name, c.Name))
						}
						ui.Dialog_end()
					}
				}

				parentId := ui.DivInfo_get(SA_DIV_GET_uid, 0)
				ui.Div_start(0, 1, 1+len(tinfo.Columns)+1, 1)
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
					vals := make([]string, 1+len(tinfo.Columns)) //1=rowid
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
						{
							dnm := "row_detail_" + node.Name + selected_table + vals[0]
							if ui.Comp_buttonLight(0, r, 1, 1, vals[0], "", true) > 0 {
								ui.Dialog_open(dnm, 1)
							}
							if ui.Dialog_start(dnm) {
								ui.Div_colMax(0, 7)
								if ui.Comp_buttonError(0, 0, 1, 1, "Delete", "", true, true) > 0 {
									db.Write_unsafe(fmt.Sprintf("DELETE FROM %s WHERE rowid=?", tinfo.Name), vals[0])
								}
								ui.Dialog_end()
							}
						}

						//cells
						for i := 1; i < len(vals); i++ {
							_, _, _, fnshd, _ := ui.Comp_editbox(i, r, 1, 1, &vals[i], Comp_editboxProp())
							if fnshd {
								db.Write_unsafe(fmt.Sprintf("UPDATE %s SET %s=? WHERE rowid=?", tinfo.Name, tinfo.Columns[i-1].Name), vals[i], vals[0])
							}
						}
						r++
					}
				}
				ui.Div_end()
			}
			ui.Div_end()

			//+row
			ui.Div_start(0, 3, 1, 1)
			{
				ui.Div_colMax(0, 3)
				if ui.Comp_button(0, 0, 1, 1, "Add row", "", true) > 0 {
					db.Write_unsafe(fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s);", tinfo.Name, tinfo.ListOfColumnNames(), tinfo.ListOfColumnValues()))
				}
			}
			ui.Div_end()
		}
	}
	ui.Div_end()

}

func UiCodeGo_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)
	ui.Div_colMax(2, 100)

	ui.Div_row(0, 1)
	ui.Div_row(2, 3)
	ui.Div_rowMax(0, 2)
	ui.Div_rowMax(2, 6)

	ui.Comp_text(0, 0, 1, 1, "Request", 0)
	ui.Comp_text(0, 2, 1, 1, "Answer", 0)

	ui.Comp_editbox(1, 0, 2, 1, &node.Code.TempCommand, Comp_editboxProp().Align(0, 0).MultiLine(true).TempToValue(true))

	if node.Code.TempCommand != node.Code.Command {
		ui.Comp_textCd(1, 1, 1, 1, "Warning: Re-generage answer", 0, CdPalette_E)
	}

	//generate button
	if ui.Comp_button(2, 1, 1, 1, "Generate", "", node.Code.TempCommand != node.Code.Command) > 0 {
		err := node.Code.GetAnswer()
		if err != nil {
			node.SetError(err)
		}
	}

	//answer
	ui.Comp_editbox(1, 2, 2, 1, &node.Code.Answer, Comp_editboxProp().Align(0, 0).MultiLine(true))

	//run button
	if ui.Comp_button(2, 3, 1, 1, "Run", "", true) > 0 {
		err := node.Code.Execute()
		if err != nil {
			node.SetError(err)
		}
	}

	//triggers
	ui.Comp_text(0, 4, 1, 1, "Triggers", 0)
	ui.Div_start(1, 4, 2, len(node.Code.Triggers))
	{
		ui.Div_colMax(1, 100)
		for i, tr := range node.Code.Triggers {
			if ui.Comp_button(0, i, 1, 1, "X", "Un-link", true) > 0 {
				node.Code.Triggers = append(node.Code.Triggers[:i], node.Code.Triggers[i+1:]...) //remove
			}
			ui.Comp_text(1, i, 1, 1, tr, 0)
		}
	}
	ui.Div_end()
}

var g_whisper_formats = []string{"verbose_json", "json", "text", "srt", "vtt"}
var g_whisper_modelList = []string{"ggml-tiny.en", "ggml-tiny", "ggml-base.en", "ggml-base", "ggml-small.en", "ggml-small", "ggml-medium.en", "ggml-medium", "ggml-large-v1", "ggml-large-v2", "ggml-large-v3"}
var g_whisper_modelsFolder = "services/whisper.cpp/models/"

func UiWhisperCpp_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	//build model list
	var models []string
	for _, m := range g_whisper_modelList {
		if m != "" { //1st is empty
			if OsFileExists(filepath.Join(g_whisper_modelsFolder, m+".bin")) {
				models = append(models, m)
			}
		}
	}

	node.ShowAttrStringCombo(&grid, "model", OsTrnString(len(models) > 0, models[0], ""), models, models)

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

	// downloader ...
}

var g_llama_modelsFolder = "services/llama.cpp/models/"

func UiLLamaCpp_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	//build model list
	var models []string
	modelFiles := OsFileListBuild(g_llama_modelsFolder, "", true)
	for _, m := range modelFiles.Subs {
		if !m.IsDir && !strings.HasPrefix(m.Name, "ggml-vocab") {
			models = append(models, m.Name)
		}
	}

	node.ShowAttrStringCombo(&grid, "model", OsTrnString(len(models) > 0, models[0], ""), models, models)

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
	node.ShowAttrInt(&grid, "mirostat", 0)
	node.ShowAttrFloat(&grid, "mirostat_tau", 5, 3)
	node.ShowAttrFloat(&grid, "mirostat_eta", 0.1, 3)
	//Grammar
	node.ShowAttrInt(&grid, "n_probs", 0)
	//Image_data
	node.ShowAttrBool(&grid, "cache_prompt", false)
	node.ShowAttrInt(&grid, "slot_id", -1)

	//downloader ...
}

var g_g4f_modelList = []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"}

func UiG4F_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrStringCombo(&grid, "model", g_g4f_modelList[0], g_g4f_modelList, g_g4f_modelList)
}

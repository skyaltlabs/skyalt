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
	"os"
	"path/filepath"
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
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "clicked", false)
	node.ShowAttrString(&grid, "confirmation", "", false)
}

func UiButton_render(node *SANode) {
	grid := node.GetGrid()
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	confirmation := node.GetAttrString("confirmation", "")
	//clicked := node.GetAttrBool("clicked", false)

	if node.app.base.ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, Comp_buttonProp().Enable(enable).Tooltip(tooltip).Confirmation(confirmation, "confirm_"+node.GetPath())) > 0 {
		node.Attrs["clicked"] = true
		node.changed = true
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
		node.app.base.ui.Comp_textSelectMulti(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, show_border, true)
	} else {
		node.app.base.ui.Comp_textSelect(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, show_border)
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
	//node.ShowAttrBool(&grid, "changed", false)
	node.ShowAttrBool(&grid, "temp_to_value", false)

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
	temp_to_value := node.GetAttrBool("temp_to_value", false)

	editedValue, active, _, fnshd, _ := node.app.base.ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, Comp_editboxProp().Ghost(ghost).MultiLine(multi_line).MultiLineEnterFinish(multi_line_enter_finish).Enable(enable).Align(align_h, align_v))

	if temp_to_value && active {
		old_changed := node.changed //node.GetAttrBool("changed", false)
		if !old_changed {
			//node.Attrs["changed"] = (node.Attrs["value"] != editedValue)
			node.changed = (node.Attrs["value"] != editedValue)
		}
		node.Attrs["value"] = editedValue
	}

	if fnshd {
		//node.Attrs["changed"] = (node.Attrs["value"] != value)
		node.changed = (node.Attrs["value"] != value)
		node.Attrs["value"] = value
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
	//node.ShowAttrBool(&grid, "changed", false)
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
		//node.Attrs["changed"] = true
		node.changed = true
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
	//node.ShowAttrBool(&grid, "changed", false)
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
		//node.Attrs["changed"] = true
		node.changed = true
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
		//node.Attrs["changed"] = true
		node.changed = true
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
	//node.ShowAttrBool(&grid, "changed", false)
}

func UiCombo_render(node *SANode) {
	grid := node.GetGrid()

	value := node.GetAttrString("value", "")
	opts_names := node.GetAttrString("options_names", "a;b;c")
	opts_values := node.GetAttrString("options_values", "0;1;2")
	search := node.GetAttrBool("search", false)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	options_names := strings.Split(opts_names, ";")
	options_values := strings.Split(opts_values, ";")

	if node.app.base.ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, options_names, options_values, tooltip, enable, search) {
		node.Attrs["value"] = value
		//node.Attrs["changed"] = true
		node.changed = true
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
	//node.ShowAttrBool(&grid, "changed", false)
}

func UiColor_render(node *SANode) {
	grid := node.GetGrid()

	value := node.GetAttrCd("value", OsCd{127, 127, 127, 255})
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.comp_colorPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, "color_picker_"+node.Name, tooltip, enable) {
		node.SetAttrCd("value", value)
		//node.Attrs["changed"] = true
		node.changed = true
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
	node.ShowAttrBool(&grid, "done", false)

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

		if ui.Comp_buttonLight(1, 0, 1, 1, "Reset", Comp_buttonProp()) > 0 {
			node.Attrs["start_sec"] = OsTime()
			prc = 0
		}
	}
	ui.Div_end()

	if enable && prc >= 1 {
		if start_sec > 0 { //if repeat==false, set 'done' only once
			node.Attrs["done"] = true
			node.changed = true
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
	//node.ShowAttrBool(&grid, "changed", false)
}

func UiDate_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrInt("value", 0)
	show_time := node.GetAttrBool("show_time", false)
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	date := int64(value)
	if node.app.base.ui.Comp_CalendarDatePicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &date, show_time, "date_"+node.Name, tooltip, enable) {
		node.Attrs["value"] = int(date)
		//node.Attrs["changed"] = true
		node.changed = true
	}
}

func UiDiskDir_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFilePicker(&grid, "path", "", false, "disk_dir_"+node.Name)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "enable", true)
	//node.ShowAttrBool(&grid, "changed", false)
}

func UiDiskDir_render(node *SANode) {
	grid := node.GetGrid()
	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_dirPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &path, false, "dir_picker_"+node.Name, enable) {
		node.Attrs["path"] = path
		//node.Attrs["changed"] = true
		node.changed = true
	}
}

func UiDiskFile_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFilePicker(&grid, "path", "", true, "disk_file_"+node.Name)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "enable", true)
	//node.ShowAttrBool(&grid, "changed", false)
}

func UiDiskFile_render(node *SANode) {
	grid := node.GetGrid()
	path := node.GetAttrString("path", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_dirPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &path, true, "dir_picker_"+node.Name, enable) {
		node.Attrs["path"] = path
		//node.Attrs["changed"] = true
		node.changed = true
	}
}

func UiMicrophone_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrFilePicker(&grid, "path", "", true, "microphone_path_"+node.Name)
	node.ShowAttrBool(&grid, "enable", true)
	//node.ShowAttrBool(&grid, "changed", false)
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
			//node.Attrs["changed"] = true
			node.changed = true
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
	node.ShowAttrBool(&grid, "write", false)
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

func UiCopy_Attrs(node *SANode) {
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
	//node.ShowAttrBool(&grid, "changed", false) //...
}

func UiCopy_render(node *SANode) {
	grid := node.GetGrid()

	direction := node.GetAttrInt("direction", 0)
	max_width := node.GetAttrFloat("max_width", 100)
	max_height := node.GetAttrFloat("max_height", 1)
	show_border := node.GetAttrBool("show_border", true)

	ui := node.app.base.ui

	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	{
		num_rows := len(node.copySubs)

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
				it := node.copySubs[i]
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
				it := node.copySubs[i]
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

func UiSQLite_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)
	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "show", true)
	node.ShowAttrBool(&grid, "write", false)
	node.ShowAttrBool(&grid, "enable", true)
	//node.ShowAttrBool(&grid, "changed", false) //...

	path := node.ShowAttrFilePicker(&grid, "path", "", true, "render_sqlite_"+node.Name)
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
							db.Write_unsafe(fmt.Sprintf("ALTER TABLE %s ADD %s %s;", tinfo.Name, g_column_name, g_column_type))
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
					vals := make([]string, len(tinfo.Columns))
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
							if ui.Comp_buttonLight(0, r, 1, 1, vals[0], Comp_buttonProp()) > 0 {
								ui.Dialog_open(dnm, 1)
							}
							if ui.Dialog_start(dnm) {
								ui.Div_colMax(0, 7)
								if ui.Comp_button(0, 0, 1, 1, "Delete", Comp_buttonProp().SetError(true).Confirmation("Are you sure?", "confirm_delete_row_"+tinfo.Name+"_"+vals[0])) > 0 {
									db.Write_unsafe(fmt.Sprintf("DELETE FROM %s WHERE rowid=?", tinfo.Name), vals[0])
								}
								ui.Dialog_end()
							}
						}

						//cells
						for i := 1; i < len(vals); i++ {
							_, _, _, fnshd, _ := ui.Comp_editbox(i, r, 1, 1, &vals[i], Comp_editboxProp())
							if fnshd {
								db.Write_unsafe(fmt.Sprintf("UPDATE %s SET %s=? WHERE rowid=?", tinfo.Name, tinfo.Columns[i].Name), vals[i], vals[0])
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

func UiCodeGo_AttrChat(node *SANode) {
	ui := node.app.base.ui

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(1, 100)

	//header
	ui.Div_start(0, 0, 1, 1)
	{
		ui.Div_colMax(1, 100)

		if ui.Comp_buttonLight(0, 0, 1, 1, "X", Comp_buttonProp()) > 0 {
			node.ShowCodeChat = false
		}
		ui.Comp_text(1, 0, 1, 1, "Code Chat", 1)

		dnm := "chat_" + node.Name
		if ui.Comp_buttonIcon(2, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/context.png"), 0.3, "", Comp_buttonProp().Cd(CdPalette_B)) > 0 {
			ui.Dialog_open(dnm, 1)
		}
		if ui.Dialog_start(dnm) {
			ui.Div_colMax(0, 5)
			y := 0

			if ui.Comp_buttonMenu(0, y, 1, 1, "Clear all", false, Comp_buttonProp()) > 0 {
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
		ui.Div_colMax(2, 100)

		node.Code.CheckLastChatEmpty()

		y := 0
		for i, str := range node.Code.Messages {
			if i+1 < len(node.Code.Messages) {
				nlines := WinFontProps_NumRows(str.User)
				ui.Comp_text(0, y, 1, 1, "User", 0)
				ui.Comp_textSelectMulti(1, y, 2, nlines, str.User, OsV2{0, 0}, true, false, false)
				y += nlines

				nlines = WinFontProps_NumRows(str.Assistent)
				ui.Div_start(0, y, 3, nlines+1)
				{
					ui.Div_colMax(0, 1.5)
					ui.Div_colMax(1, 100)
					ui.Div_colMax(2, 100)

					pl := ui.win.io.GetPalette()
					ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.85), 0)

					ui.Comp_text(0, 0, 1, 1, "Bot", 0)
					ui.Comp_textSelectMulti(1, 0, 2, nlines, str.Assistent, OsV2{0, 0}, true, false, false)

					if ui.Comp_buttonLight(2, nlines, 1, 1, "Use this code", Comp_buttonProp()) > 0 {
						node.Code.UseCodeFromAnswer(str.Assistent)
					}
				}
				ui.Div_end()
				y += nlines + 1

				//ui.Div_SpacerRow(0, y, 3, 1)
				y += 2 //space
			} else {
				//last one is only user editbox

				nlines := WinFontProps_NumRows(str.User) + 2
				ui.Comp_editbox(0, y, 3, nlines, &node.Code.Messages[i].User, Comp_editboxProp().MultiLine(true).Align(0, 0).Formating(false))
				y += nlines

				//or ctrl+enter ..........
				if ui.Comp_buttonLight(2, y, 1, 1, "Send", Comp_buttonProp()) > 0 {
					node.Code.GetAnswer()
				}

			}
		}

		//delete last one ...
	}
	ui.Div_end()
}

func UiCodeGo_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	y := 0

	if node.Code.ans_err != nil {
		ui.Comp_textCd(0, y, 2, 1, "Answer Error: "+node.Code.ans_err.Error(), 0, CdPalette_E)
		y++
		node.SetError(fmt.Errorf("Answer"))
	}
	if node.Code.file_err != nil {
		ui.Comp_textCd(0, y, 2, 1, "File Error: "+node.Code.file_err.Error(), 0, CdPalette_E)
		y++
		node.SetError(fmt.Errorf("File"))
	}
	if node.Code.exe_err != nil {
		ui.Comp_textCd(0, y, 2, 1, "Execute Error: "+node.Code.exe_err.Error(), 0, CdPalette_E)
		y++
		node.SetError(fmt.Errorf("Execute"))
	}

	//bypass
	{
		gr := InitOsV4(0, y, 1, 1)
		node.ShowAttrBool(&gr, "bypass", false)
		y += 1
	}

	ui.Div_SpacerRow(0, y, 2, 1)
	y++

	//triggers
	{
		ui.Comp_text(0, y, 1, 1, "Triggers", 0)
		nTrigs := len(node.Code.Triggers)
		ui.Div_start(1, y, 1, nTrigs+1)
		{
			ui.Div_colMax(1, 100)
			for i, tr := range node.Code.Triggers {
				if ui.Comp_buttonLight(0, i, 1, 1, "X", Comp_buttonProp().Tooltip("Un-link")) > 0 {
					node.Code.Triggers = append(node.Code.Triggers[:i], node.Code.Triggers[i+1:]...) //remove
				}
				ui.Comp_text(1, i, 1, 1, tr, 0)
			}

			ui.Div_start(0, nTrigs, 2, 1)
			{
				ui.Div_colMax(1, 5)
				ui.Comp_text(0, 0, 1, 1, "+", 1)
				var pick_node string
				if ui.Comp_combo(1, 0, 1, 1, &pick_node, node.app.all_triggers_str, node.app.all_triggers_str, "", true, true) {
					node.Code.addTrigger(pick_node)
				}
			}
			ui.Div_end()

		}
		ui.Div_end()
		y += nTrigs + 1

		if len(node.Code.Triggers) == 0 {
			node.SetError(fmt.Errorf("no trigger(s)"))
		}
	}

	ui.Div_SpacerRow(0, y, 2, 1)
	y++

	//Features
	/*{
		ui.Comp_text(0, y, 1, 1, "Features", 0)
		for i, str := range node.Code.Messages {
			if str.User != "" && i+1 < len(node.Code.Messages) { //must exist, not last one(because it's without answer yet)
				nlines := WinFontProps_NumRows(str.User)
				ui.Comp_textSelectMulti(1, y, 1, nlines, str.User, OsV2{0, 0}, true, false)
				y += nlines
			}
		}

		str := OsTrnString(node.ShowCodeChat, "Close Assistent", "Open Assistent")
		if ui.Comp_buttonLight(1, y, 1, 1, str, "", true) > 0 {
			node.ShowCodeChat = !node.ShowCodeChat
		}
		y++
	}

	ui.Div_SpacerRow(0, y, 2, 1)
	y++*/

	// Open chat
	{
		str := OsTrnString(node.ShowCodeChat, "Close Code chat", "Open Code chat")
		if ui.Comp_buttonLight(1, y, 1, 1, str, Comp_buttonProp()) > 0 {
			node.ShowCodeChat = !node.ShowCodeChat
		}
		y++
	}

	//Imports
	/*{
		ui.Comp_text(0, y, 1, 1, "Imports", 0)
		n := len(node.Code.Imports)
		ui.Div_start(1, y, 1, n+1)
		{
			ui.Div_colMax(1, 3)
			ui.Div_colMax(2, 100)
			//ui.Div_colMax(1, 100)
			for i, im := range node.Code.Imports {
				if ui.Comp_buttonLight(0, i, 1, 1, "X", "Remove", true) > 0 {
					node.Code.Imports = append(node.Code.Imports[:i], node.Code.Imports[i+1:]...) //remove
				}
				ui.Comp_text(1, i, 1, 1, im.Name, 0)
				ui.Comp_text(2, i, 1, 1, im.Path, 0)
			}
		}
		ui.Div_end()
		y += OsMax(1, n)
	}*/

	//Code
	{
		ui.Comp_text(0, y, 1, 1, "Code", 0)

		nlines := WinFontProps_NumRows(node.Code.Code)
		_, _, _, fnshd, _ := ui.Comp_editbox(1, y, 1, nlines, &node.Code.Code, Comp_editboxProp().Align(0, 0).MultiLine(true).Formating(false))
		if fnshd {
			node.Code.UpdateFile()
		}
		y += nlines

		//run button
		if ui.Comp_button(1, y, 1, 1, "Run", Comp_buttonProp()) > 0 {
			node.Code.Execute()
		}
		y++
	}

	ui.Div_SpacerRow(0, y, 2, 1)
	y++

	//output
	{
		ui.Comp_text(0, y, 1, 1, "Output", 0)
		nlines := OsClamp(WinFontProps_NumRows(node.Code.output), 2, 5)
		ui.Comp_textSelectMulti(1, y, 1, nlines, node.Code.output, OsV2{0, 0}, true, true, false)
		y += nlines
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
}

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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/go-audio/wav"
)

func SAExe_Render_Layout(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		w.renderLayout()
		ui.Div_end()
	}
}

func SAExe_Render_Dialog(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && ui != nil

	triggerAttr := w.GetAttrUi("trigger", 0, SAAttrUi_SWITCH)
	typeAttr := w.GetAttrUi("type", 0, SAAttrUi_COMBO("Center;Relative", ""))

	dnm := w.getPath()
	if triggerAttr.GetBool() {
		ui.Dialog_open(dnm, uint8(OsClamp(typeAttr.GetInt(), 0, 2)))
		//triggerAttr.AddSetAttr("0")
	}

	if showIt {
		if ui.Dialog_start(dnm) {
			w.renderLayout()
			ui.Dialog_end()
		}
	}
}

func SAExe_Render_Button(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool()
	tp := w.GetAttrUi("type", 0, SAAttrUi_COMBO("Classic;Light;Menu", "")).GetInt()
	labelAttr := w.GetAttr("label", "")
	label := labelAttr.GetString()
	clickedAttr := w.GetAttrUi("clicked", 0, SAAttrUi_SWITCH) //can't be _output, because it's set when render(not graph execution)

	clicked := false
	switch tp {
	case 0:
		if showIt {
			clicked = ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, "", enable) > 0
		}
	case 1:
		if showIt {
			clicked = ui.Comp_buttonLight(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, "", enable) > 0
		}
	case 2:
		selectedAttr := w.GetAttrUi("selected", 0, SAAttrUi_SWITCH)

		if showIt {
			clicked = ui.Comp_buttonMenu(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, "", enable, selectedAttr.GetBool()) > 0
			if clicked {
				selectedAttr.AddSetAttr(OsTrnString(selectedAttr.GetBool(), "0", "1")) //reverse
			}

			w.app.flamingo.TryAddItemFromAttr(labelAttr)
		}
	}

	if clicked {
		clickedAttr.AddSetAttr("1")
		clickedAttr.AddSetAttrExe()
		clickedAttr.AddSetAttr("0")
	}
}

func SAExe_Render_Text(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	labelAttr := w.GetAttr("label", "")
	label := labelAttr.GetString()
	align_h := w.GetAttrUi("align_h", 0, SAAttrUi_COMBO("Left;Center;Right", "")).GetInt()
	align_v := w.GetAttrUi("align_v", 1, SAAttrUi_COMBO("Top;Center;Bottom", "")).GetInt()
	selection := w.GetAttrUi("selection", 1, SAAttrUi_SWITCH).GetBool()
	multi_line := w.GetAttrUi("multi_line", 0, SAAttrUi_SWITCH).GetBool()
	drawBorder := w.GetAttrUi("draw_border", 0, SAAttrUi_SWITCH).GetBool()

	if showIt {
		if multi_line {
			ui.Comp_textSelectMulti(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, drawBorder)
		} else {
			ui.Comp_textSelect(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, drawBorder)
		}

		w.app.flamingo.TryAddItemFromAttr(labelAttr)
	}
}

func SAExe_Render_Switch(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	label := w.GetAttr("label", "").GetString()

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool() && valueInstr != nil

	if showIt {
		if ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, "", enable) {
			valueInstr.LineReplace(value, false)
		}

		w.app.flamingo.TryAddItemFromAttr(valueAttr)
	}
}

func SAExe_Render_Checkbox(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	label := w.GetAttr("label", "").GetString()

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool() && valueInstr != nil

	if showIt {
		if ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, "", enable) {
			valueInstr.LineReplace(value, false)
		}

		w.app.flamingo.TryAddItemFromAttr(valueAttr)
	}
}

func SAExe_Render_Slider(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	min := w.GetAttr("min", 0).GetFloat()
	max := w.GetAttr("max", 10).GetFloat()
	step := w.GetAttr("step", 1).GetFloat()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool() && valueInstr != nil

	if showIt {
		if ui.Comp_slider(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, min, max, step, enable) {
			valueInstr.LineReplace(value, false)
		}

		w.app.flamingo.TryAddItemFromAttr(valueAttr)
	}
}

func SAExe_Render_Combo(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool() && valueInstr != nil
	options_names := w.GetAttr("options_names", "a;b;c").GetString()
	options_values := w.GetAttr("options_values", "a;b;c").GetString()
	search := w.GetAttrUi("search", 0, SAAttrUi_SWITCH).GetBool()

	if showIt {
		if ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, strings.Split(options_names, ";"), strings.Split(options_values, ";"), "", enable, search) {
			valueInstr.LineReplace(value, false)
		}

		w.app.flamingo.TryAddItemFromAttr(valueAttr)
	}
}

func SAExe_Render_Editbox(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	prop := Comp_editboxProp()

	prop.align.X = w.GetAttrUi("align_h", 0, SAAttrUi_COMBO("Left;Center;Right", "")).GetInt()
	prop.align.Y = w.GetAttrUi("align_v", 1, SAAttrUi_COMBO("Top;Center;Bottom", "")).GetInt()
	prop.enable = w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool() && valueInstr != nil
	prop.tempToValue = w.GetAttrUi("tempToValue", 0, SAAttrUi_SWITCH).GetBool()
	prop.value_precision = w.GetAttr("precision", 2).GetInt()
	prop.ghost = w.GetAttr("ghost", "").GetString()
	prop.multi_line = w.GetAttrUi("multi_line", 0, SAAttrUi_SWITCH).GetBool()
	prop.multi_line_enter_finish = w.GetAttrUi("multi_line_enter_finish", 0, SAAttrUi_SWITCH).GetBool()
	finishedAttr := w.GetAttrUi("finished", 0, SAAttrUi_SWITCH)
	enter_finishedAttr := w.GetAttrUi("enter_finished", 0, SAAttrUi_SWITCH)
	isEmptyAttr := w.GetAttrUi("isEmpty", 0, SAAttrUi_SWITCH)

	//temp, tempChanged? ...

	if showIt {
		editedValue, _, chngd, fnshd, _ := ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, prop)
		if fnshd || (prop.tempToValue && chngd) {
			valueInstr.LineReplace(value, false)
		}
		if fnshd {
			finishedAttr.AddSetAttr("1")
			finishedAttr.AddSetAttrExe()
			finishedAttr.AddSetAttr("0")

			if ui.win.io.keys.enter {
				enter_finishedAttr.AddSetAttr("1")
				enter_finishedAttr.AddSetAttrExe()
				enter_finishedAttr.AddSetAttr("0")
			}
		}

		isEmptyAttr.AddSetAttr(OsTrnString(editedValue == "", "1", "0"))
		//_isEmptyAttr.GetResult().SetBool(editedValue == "")

		w.app.flamingo.TryAddItemFromAttr(valueAttr)
	}

}

func SAExe_Render_Divider(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	tp := w.GetAttrUi("type", 0, SAAttrUi_COMBO("Column;Row", "")).GetInt()

	if showIt {
		switch tp {
		case 0:
			ui.Div_SpacerCol(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		case 1:
			ui.Div_SpacerRow(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		}
	}
}

func SAExe_Render_ColorPalette(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	cdAttr := w.GetAttrUi("cd", []byte("[0, 0, 0, 255]"), SAAttrUi_COLOR)
	cd := cdAttr.GetCd()

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			if ui.comp_colorPalette(&cd) {
				cdAttr.ReplaceCd(cd)
			}
		}
		ui.Div_end()

		w.app.flamingo.TryAddItemFromAttr(cdAttr)
	}
}

func SAExe_Render_ColorPicker(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool()
	cdAttr := w.GetAttrUi("cd", []byte("[0, 0, 0, 255]"), SAAttrUi_COLOR)
	cd := cdAttr.GetCd()

	if showIt {
		if ui.comp_colorPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &cd, w.getPath(), enable) {
			cdAttr.ReplaceCd(cd)
		}

		w.app.flamingo.TryAddItemFromAttr(cdAttr)
	}
}

func _SAExe_Render_FileAndDirPicker(selectFile bool, dialogName string, w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	uiV := SAAttrUi_DIR
	if selectFile {
		uiV = SAAttrUi_FILE
	}

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool()
	pathAttr := w.GetAttrUi("path", "", uiV)
	path := pathAttr.GetString()

	if showIt {
		if ui.comp_dirPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &path, selectFile, dialogName, enable) {
			pathAttr.AddSetAttr(path)
		}

		w.app.flamingo.TryAddItemFromAttr(pathAttr)
	}
}

func SAExe_Render_FilePicker(w *SANode, renderIt bool) {
	_SAExe_Render_FileAndDirPicker(true, w.getPath(), w, renderIt)
}

func SAExe_Render_FolderPicker(w *SANode, renderIt bool) {
	_SAExe_Render_FileAndDirPicker(false, w.getPath(), w, renderIt)
}

func SAExe_Render_Calendar(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	valueAttr := w.GetAttrUi("value", 0, SAAttrUi_DATE)
	pageAttr := w.GetAttrUi("page", 0, SAAttrUi_DATE)

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := valueAttr.GetInt64()
			page := pageAttr.GetInt64()

			changed := ui.Comp_Calendar(&value, &page, 100, 100)
			if changed {
				//set back
				valueAttr.AddSetAttr(strconv.Itoa(int(value)))
				pageAttr.AddSetAttr(strconv.Itoa(int(page)))
			}
		}
		ui.Div_end()

		w.app.flamingo.TryAddItemFromAttr(valueAttr)
	}
}

func SAExe_Render_Date(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool()
	valueAttr := w.GetAttrUi("value", 0, SAAttrUi_DATE)
	show_time := w.GetAttrUi("show_time", 0, SAAttrUi_SWITCH).GetBool()

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := valueAttr.GetInt64()
			if ui.Comp_CalendarDataPicker(&value, show_time, w.getPath(), enable) {
				valueAttr.AddSetAttr(strconv.Itoa(int(value)))
			}
		}
		ui.Div_end()

		w.app.flamingo.TryAddItemFromAttr(valueAttr)
	}
}

func SAExe_Render_Image(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	margin := w.GetAttr("margin", 0).GetFloat()
	cd := w.GetAttrUi("cd", []byte("[255, 255, 255, 255]"), SAAttrUi_COLOR).GetCd()
	background := w.GetAttrUi("background", 0, SAAttrUi_SWITCH).GetBool()

	alignV := w.GetAttrUi("alignV", 1, SAAttrUi_COMBO("Left;Center;Right", "")).GetInt()
	alignH := w.GetAttrUi("alignH", 1, SAAttrUi_COMBO("Left;Center;Right", "")).GetInt()
	fill := w.GetAttrUi("fill", 0, SAAttrUi_SWITCH).GetBool()

	blobAttr := w.GetAttrUi("blob", "", SAAttrUi_BLOB)

	if !renderIt {
		if !blobAttr.IsBlob() {
			blobAttr.SetErrorStr("Not BLOB")
			return
		}
	}

	if background {
		w.z_depth = 0.5
	}

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			blob := blobAttr.GetBlob()
			path := InitWinMedia_blob(blob)
			ui.Paint_file(0, 0, 1, 1, margin, path, cd, alignV, alignH, fill, background)
		}
		ui.Div_end()

		w.app.flamingo.TryAddItemFromAttr(blobAttr)
	}
}

func SAExe_Render_FileDrop(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	pathAttr := w.GetAttr("path", "")
	instr := pathAttr.instr.GetConst()
	value := instr.pos_attr.GetString()

	outputAttr := w.GetAttrUi("_out", "", SAAttrUi_BLOB)

	if showIt {
		div := ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			ui.Div_colMax(0, 100)
			ui.Div_rowMax(0, 100)
			ui.Div_start(0, 0, 1, 1)
			ui.Paint_rect(0, 0, 1, 1, 0.03, OsCd{0, 0, 0, 255}, 0.03)
			ui.Div_end()
			ui.Comp_text(0, 0, 1, 1, "Drag file & drop it here", 1)

			_, _, _, fnshd, _ := ui.Comp_editbox(0, 1, 1, 1, &value, Comp_editboxProp().Ghost("path"))
			if fnshd {
				instr.LineReplace(value, false)
			}

			if div.IsOver(ui) {
				value = ui.win.io.touch.drop_path //rewrite 'value'!
				if value != "" {
					instr.LineReplace(value, false)
				}
			}

			w.app.flamingo.TryAddItemFromAttr(pathAttr)
		}
		ui.Div_end()
	}

	if !renderIt {
		outputAttr.SetOutBlob(nil) //reset

		if value != "" {
			data, err := os.ReadFile(value)
			if err == nil {
				outputAttr.SetOutBlob(data)
			} else {
				pathAttr.SetError(fmt.Errorf("ReadFile(%s) failed: %w", value, err))
			}
		}
	}
}

func SAExe_Render_Map(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	file := w.GetAttr("file", "maps/osm").GetString()
	url := w.GetAttr("url", "https://tile.openstreetmap.org/{z}/{x}/{y}.png").GetString()
	copyright := w.GetAttr("copyright", "(c)OpenStreetMap contributors").GetString()
	copyright_url := w.GetAttr("copyright_url", "https://www.openstreetmap.org/copyright").GetString()

	cam_lonAttr := w.GetAttr("lon", 14.4071117049)
	cam_latAttr := w.GetAttr("lat", 50.0852013259)
	cam_zoomAttr := w.GetAttr("zoom", 5)

	//locators
	locatorsAttr := w.GetAttr("locators", []byte(`[{"label":"1", "lon":14.4071117049, "lat":50.0852013259}, {"label":"2", "lon":14, "lat":50}]`))

	//segments
	segmentsAttr := w.GetAttr("segments", []byte(`[{"label":"ABC", "Trkpt":[{"lat":50,"lon":16,"ele":400,"time":"2020-04-15T09:05:20Z"},{"lat":50.4,"lon":16.1,"ele":400,"time":"2020-04-15T09:05:23Z"}]}]`))

	//Locators and Path can be Array or single Item(1x map)! .......

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			ui.Div_colMax(0, 100)
			ui.Div_rowMax(0, 100)

			file = "disk/" + file

			cam_lon := cam_lonAttr.GetFloat()
			cam_lat := cam_latAttr.GetFloat()
			cam_zoom := cam_zoomAttr.GetFloat()

			changed, err := ui.comp_map(&cam_lon, &cam_lat, &cam_zoom, file, url, copyright, copyright_url)
			if err != nil {
				w.SetError(err)
			}

			if changed {
				//set back
				cam_lonAttr.AddSetAttr(strconv.FormatFloat(cam_lon, 'f', -1, 64))
				cam_latAttr.AddSetAttr(strconv.FormatFloat(cam_lat, 'f', -1, 64))
				cam_zoomAttr.AddSetAttr(strconv.Itoa(int(cam_zoom)))
			}

			//locators
			locatorsBlob := locatorsAttr.GetBlob()
			if locatorsBlob.Len() > 0 {
				var locators []UiCompMapLocator
				err := json.Unmarshal(locatorsBlob.data, &locators)
				if err == nil {
					err = ui.comp_mapLocators(cam_lon, cam_lat, cam_zoom, locators, w.getPath())
					if err != nil {
						locatorsAttr.SetError(fmt.Errorf("comp_mapLocators() failed: %w", err))
					}
				} else {
					locatorsAttr.SetError(fmt.Errorf("Unmarshal() failed: %w", err))
				}
			}

			//paths
			segmentsBlob := segmentsAttr.GetBlob()
			if segmentsBlob.Len() > 0 {
				var segments []UiCompMapSegments
				err := json.Unmarshal(segmentsBlob.data, &segments)
				if err == nil {
					err = ui.comp_mapSegments(cam_lon, cam_lat, cam_zoom, segments)
					if err != nil {
						segmentsAttr.SetError(fmt.Errorf("comp_mapSegments() failed: %w", err))
					}
				} else {
					segmentsAttr.SetError(fmt.Errorf("Unmarshal() failed: %w", err))
				}
			}
			ui.Div_end()

			//w.app.flamingo.tryAddItemFromAttr(valueAttr)	? //add every item alone = attr + index .......
		}
	}
}

func SAExe_Render_List(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	itemsAttr := w.GetAttr("items", []byte("[0, 1, 2, 3, 4, 5]"))
	multiSelect := w.GetAttrUi("multi_select", 1, SAAttrUi_CHECKBOX).GetBool()
	direction := w.GetAttrUi("direction", 0, SAAttrUi_COMBO("Vertical;Horizonal", "")).GetBool()
	maxItemSize := w.GetAttr("max_item_size", []byte("[100, 1]")).GetV2f()

	single_selectedAttr := w.GetAttr("single_selected", -1)             //input, but can replace(instr)
	multi_selectedAttr := w.GetAttr("multi_selected", []byte("[2, 3]")) //input, but can replace(instr)

	var items []interface{}
	err := json.Unmarshal(itemsAttr.GetBlob().data, &items)
	if err != nil {
		itemsAttr.SetError(err)
		return
	}

	var single_selected int
	var multi_selected []int
	if multiSelect {
		err = json.Unmarshal(multi_selectedAttr.GetBlob().data, &multi_selected)
		if err != nil {
			multi_selectedAttr.SetError(err)
			return
		}
		sort.Ints(multi_selected)
	} else {
		single_selected = single_selectedAttr.GetInt()
	}

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)

		lv := ui.GetCall().call
		canvas_w := float32(OsMax(0, lv.canvas.Size.X-lv.data.scrollV._GetWidth(ui.win)))
		canvas_h := float32(OsMax(0, lv.canvas.Size.Y-lv.data.scrollV._GetWidth(ui.win)))

		var num_rows, num_cols int
		if !direction {
			lay_cols := canvas_w / float32(ui.win.Cell())
			maxItemSize.X = OsMinFloat32(maxItemSize.X, float32(lay_cols))

			num_cols = int(lay_cols / float32(maxItemSize.X))
			num_cols = OsMax(num_cols, 1)
			num_rows = OsRoundUp(float64(len(items)) / float64(num_cols))

			//vertical
			for i := 0; i < num_cols; i++ {
				if maxItemSize.X < lay_cols {
					ui.Div_col(i, float64(maxItemSize.X))
				}
				ui.Div_colMax(i, 100)
			}
			for i := 0; i < num_rows; i++ {
				ui.Div_row(i, float64(maxItemSize.Y))
			}
		} else {
			lay_rows := canvas_h / float32(ui.win.Cell())
			maxItemSize.Y = OsMinFloat32(maxItemSize.Y, float32(lay_rows))
			num_rows = int(float64(lay_rows) / float64(maxItemSize.Y))
			num_rows = OsMax(num_rows, 1)

			lay_cols := canvas_w / float32(ui.win.Cell())
			num_cols = int(lay_cols / float32(maxItemSize.X))

			mx_rows := OsRoundUp(float64(len(items)) / float64(num_rows))
			num_cols = OsMax(num_cols, mx_rows)

			//horizontal
			for i := 0; i < num_cols; i++ {
				ui.Div_col(i, float64(maxItemSize.X))
			}
			for i := 0; i < num_rows; i++ {
				if maxItemSize.Y < lay_rows {
					ui.Div_row(i, float64(maxItemSize.Y))
				}
				ui.Div_rowMax(i, 100)
			}
		}

		i := 0
		for y := 0; y < num_rows; y++ {
			for x := 0; x < num_cols; x++ {
				if i >= len(items) {
					break
				}

				var isSelected bool
				var msel_i int
				if multiSelect {
					msel_i = sort.SearchInts(multi_selected, i)
					isSelected = msel_i < len(multi_selected) && multi_selected[msel_i] == i
				} else {
					isSelected = single_selected == i
				}

				label := ""
				switch vv := items[i].(type) {
				case string:
					label = vv
				case float64:
					label = strconv.FormatFloat(vv, 'f', -1, 64)
				}

				clicked := ui.Comp_buttonMenu(x, y, 1, 1, label, "", true, isSelected) > 0
				if clicked {
					if multiSelect {
						if isSelected {
							multi_selected = append(multi_selected[:msel_i], multi_selected[msel_i+1:]...) //remove
						} else {
							multi_selected = append(multi_selected, i) //add
							sort.Ints(multi_selected)
						}

						str := "["
						for _, s := range multi_selected {
							str += strconv.Itoa(s) + ","
						}
						str, _ = strings.CutSuffix(str, ",")
						str += "]"
						multi_selectedAttr.AddSetAttrArr(str)
					} else {
						single_selectedAttr.AddSetAttr(strconv.Itoa(OsTrn(isSelected, -1, i)))
					}
				}

				i++
			}
		}

		ui.Div_end()
	}
}

func SAExe_Render_Table(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	resizable := w.GetAttrUi("resizable", 0, SAAttrUi_SWITCH).GetBool()
	columnsAttr := w.GetAttr("columns", []byte(`["a", "b"]`))
	rowsAttr := w.GetAttr("rows", []byte(`[{"a":1, "b":2}, {"a":10, "b":20}, {"a":100, "b":200}]`))

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)

		//rows
		var rows []map[string]interface{}
		err := json.Unmarshal([]byte(rowsAttr.GetString()), &rows)
		if err != nil {
			rowsAttr.SetError(err)
		}

		//columns
		var columnNames []string
		if columnsAttr.GetString() != "" {
			err := json.Unmarshal([]byte(columnsAttr.GetString()), &columnNames)
			if err != nil {
				columnsAttr.SetError(err)
			}
		} else {
			if len(rows) > 0 {
				for key := range rows[0] {
					columnNames = append(columnNames, key)
				}
			}
		}
		sort.Strings(columnNames)

		for c, name := range columnNames {
			if resizable {
				ui.Div_colResize(1+c, name, 4, false)
			} else {
				ui.Div_colMax(1+c, 100)
			}
		}
		ui.Div_col(1+len(columnNames), 0.5) //extra empty

		ui.Div_rowMax(1, 100)

		//show columns
		ui.Comp_text(0, 0, 1, 1, "#", 1)
		for c, col := range columnNames {
			ui.Comp_buttonLight(1+c, 0, 1, 1, col, "", true)
		}

		parentId := ui.DivInfo_get(SA_DIV_GET_uid, 0)
		ui.Div_start(0, 1, 1+len(columnNames)+1, 1)
		{
			//copy cols from parent
			ui.DivInfo_set(SA_DIV_SET_copyCols, parentId, 0)
			ui.DivInfo_set(SA_DIV_SET_scrollOnScreen, 1, 0)
			ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

			//visible rows
			lv := ui.GetCall()
			row_st := lv.call.data.scrollV.GetWheel() / ui.win.Cell()
			row_en := row_st + OsRoundUp(float64(lv.call.crop.Size.Y)/float64(ui.win.Cell()))
			if len(rows) > 0 {
				ui.Div_row(len(rows)-1, 1) //set last
			}

			//show rows(only visible)
			for r := row_st; r <= row_en && r < len(rows); r++ {

				ui.Comp_text(0, r, 1, 1, strconv.Itoa(r), 1) //row #

				for c, col := range columnNames {
					item := rows[r][col]
					switch vv := item.(type) {
					case int:
						ui.Comp_text(1+c, r, 1, 1, strconv.Itoa(vv), 1) //+1 => header
					case float64:
						ui.Comp_text(1+c, r, 1, 1, strconv.FormatFloat(vv, 'f', -1, 64), 1)
					case string:
						ui.Comp_text(1+c, r, 1, 1, vv, 1)
					}
				}
			}
		}
		ui.Div_end()

		//rect around
		pl := ui.win.io.GetPalette()
		ui.Paint_rect(0, 0, 1, 1, 0, pl.P, 0.03)

		ui.Div_end()
	}
}

func SAExe_Render_Microphone(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()

	enable := w.GetAttrUi("enable", 1, SAAttrUi_SWITCH).GetBool()
	fileAttr := w.GetAttr("temp_file", "temp_mic.wav")
	outAttr := w.GetAttr("_out", "")
	finishedAttr := w.GetAttrUi("finished", 0, SAAttrUi_SWITCH)

	if fileAttr.GetString() == "" {
		fileAttr.SetErrorStr("empty")
		return
	}

	if showIt {

		nodePath := NewSANodePath(w)
		rec_active := w.app.IsMicNodeRecording(nodePath)

		cd := CdPalette_B
		if rec_active {
			cd = CdPalette_P
		}
		if ui.Comp_buttonIcon(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, InitWinMedia_url("file:apps/base/resources/mic.png"), 0.15, "Enable/Disable audio recording", cd, enable, rec_active) > 0 {

			if !rec_active {
				//start
				if w.app.base.ui.win.io.ini.MicOff {
					outAttr.SetErrorStr("Microphone is disabled in SkyAlt Settings")
					return
				}
				w.app.AddMicNode(nodePath)
			} else {
				//stop
				w.app.RemoveMicNode(nodePath)

				//make WAV file
				{
					file, err := os.Create(fileAttr.GetString())
					if err != nil {
						w.SetError(err)
						return
					}
					buff := w.temp_mic_data
					enc := wav.NewEncoder(file, buff.Format.SampleRate, buff.SourceBitDepth, buff.Format.NumChannels, 1)
					err = enc.Write(&buff)
					if err != nil {
						enc.Close()
						file.Close()
						w.SetError(err)
						return
					}
					enc.Close()
					file.Close()

					w.temp_mic_data.Data = nil
				}

				renderIt = false //load it into _out

				//set finished
				finishedAttr.AddSetAttr("1")
				finishedAttr.AddSetAttrExe()
				finishedAttr.AddSetAttr("0")
			}
		}
	}

	//read file
	if !renderIt { //waste of resources to load file every drawFrame() ..........

		wavData, err := os.ReadFile(fileAttr.GetString())
		if err != nil {
			w.SetError(err)
			return
		}
		outAttr.SetOutBlob(wavData)
	}

}

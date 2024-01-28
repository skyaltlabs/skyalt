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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func SAExe_Render_Layout(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		w.renderLayout()
		ui.Div_end()
	}
}

func SAExe_Render_Dialog(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && ui != nil

	triggerAttr := w.GetAttrUi("trigger", "0", SAAttrUi_SWITCH)
	typeAttr := w.GetAttrUi("type", "0", SAAttrUi_COMBO("Center;Relative", ""))

	if showIt {
		dnm := w.getPath()
		if triggerAttr.GetBool() {
			ui.Dialog_open(dnm, uint8(OsClamp(typeAttr.GetInt(), 0, 2)))
			triggerAttr.SetExpBool(false)
		}

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
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool()
	tp := w.GetAttrUi("type", "0", SAAttrUi_COMBO("Classic;Light;Menu", "")).GetInt()
	label := w.GetAttr("label", "").GetString()

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
		selected := w.GetAttrUi("selected", "0", SAAttrUi_SWITCH).GetBool()

		if showIt {
			clicked = ui.Comp_buttonMenu(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, "", enable, selected) > 0
			if clicked {
				sel := w.findAttr("selected")
				sel.SetExpBool(!selected)
			}
		}
	}

	if clicked {
		w.app.clickedAttr = w.GetAttrUi("clicked", "0", SAAttrUi_SWITCH)
	}
}

func SAExe_Render_Text(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	label := w.GetAttr("label", "").GetString()
	align := w.GetAttrUi("align", "0", SAAttrUi_COMBO("Left;Center;Right", "")).GetInt()

	if showIt {
		ui.Comp_text(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, align)
	}
}

func SAExe_Render_Switch(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	label := w.GetAttr("label", "").GetString()

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool() && valueInstr != nil

	if showIt {
		if ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, "", enable) {
			valueInstr.LineReplace(value, false)
		}
	}
}

func SAExe_Render_Checkbox(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	label := w.GetAttr("label", "").GetString()

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool() && valueInstr != nil

	if showIt {
		if ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, "", enable) {
			valueInstr.LineReplace(value, false)
		}
	}
}

func SAExe_Render_Combo(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool() && valueInstr != nil
	options_names := w.GetAttr("options_names", "\"a;b;c\")").GetString()
	options_values := w.GetAttr("options_values", "\"a;b;c\")").GetString()
	search := w.GetAttrUi("search", "0", SAAttrUi_SWITCH).GetBool()

	if showIt {
		if ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, strings.Split(options_names, ";"), strings.Split(options_values, ";"), "", enable, search) {
			valueInstr.LineReplace(value, false)
		}
	}
}

func SAExe_Render_Editbox(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	valueAttr := w.GetAttr("value", "")
	valueInstr := valueAttr.instr.GetConst()
	value := valueAttr.GetString()

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool() && valueInstr != nil
	tmpToValue := w.GetAttrUi("tempToValue", "0", SAAttrUi_SWITCH).GetBool()
	precision := w.GetAttr("precision", "2").GetInt()
	ghost := w.GetAttr("ghost", "").GetString()

	if showIt {
		_, _, chngd, fnshd, _ := ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, precision, nil, ghost, false, tmpToValue, enable)
		if fnshd || (tmpToValue && chngd) {
			valueInstr.LineReplace(value, false)
		}
	}
}

func SAExe_Render_Divider(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	tp := w.GetAttrUi("type", "0", SAAttrUi_COMBO("Column;Row", "")).GetInt()

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
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	cdAttr := w.GetAttrUi("cd", "[0, 0, 0, 255]", SAAttrUi_COLOR)
	cd := cdAttr.GetCd()

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			if ui.comp_colorPalette(&cd) {
				cdAttr.ReplaceCd(cd)
			}
		}
		ui.Div_end()
	}
}

func SAExe_Render_Color(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool()
	cdAttr := w.GetAttrUi("cd", "[0, 0, 0, 255]", SAAttrUi_COLOR)
	cd := cdAttr.GetCd()

	if showIt {
		if ui.comp_colorPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &cd, w.getPath(), enable) {
			cdAttr.ReplaceCd(cd)
		}
	}
}

func SAExe_Render_Calendar(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	valueAttr := w.GetAttrUi("value", "0", SAAttrUi_DATE)
	pageAttr := w.GetAttrUi("page", "0", SAAttrUi_DATE)

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := valueAttr.GetInt64()
			page := pageAttr.GetInt64()

			changed := ui.Comp_Calendar(&value, &page, 100, 100)

			if changed {
				//set back
				valueAttr.SetExpInt(int(value))
				pageAttr.SetExpInt(int(page))
			}
		}
		ui.Div_end()
	}
}

func SAExe_Render_Date(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool()
	valueAttr := w.GetAttrUi("value", "0", SAAttrUi_DATE)
	show_time := w.GetAttrUi("show_time", "0", SAAttrUi_SWITCH).GetBool()

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := valueAttr.GetInt64()
			if ui.Comp_CalendarDataPicker(&value, show_time, w.getPath(), enable) {
				valueAttr.SetExpInt(int(value))
			}
		}
		ui.Div_end()
	}
}

func SAExe_Render_Image(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	margin := w.GetAttr("margin", "0").GetFloat()
	cd := w.GetAttrUi("cd", "[255, 255, 255, 255]", SAAttrUi_COLOR).GetCd()
	background := w.GetAttrUi("background", "0", SAAttrUi_SWITCH).GetBool()

	alignV := w.GetAttrUi("alignV", "1", SAAttrUi_COMBO("Left;Center;Right", "")).GetInt()
	alignH := w.GetAttrUi("alignH", "1", SAAttrUi_COMBO("Left;Center;Right", "")).GetInt()
	fill := w.GetAttrUi("fill", "0", SAAttrUi_SWITCH).GetBool()

	blobAttr := w.GetAttrUi("blob", "", SAAttrUi_BLOB)

	if !renderIt {
		if !blobAttr.IsBlob() {
			blobAttr.SetErrorExe("Not BLOB")
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
	}
}

func SAExe_Render_FileDrop(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

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

			_, _, _, fnshd, _ := ui.Comp_editbox(0, 1, 1, 1, &value, 0, nil, "path", false, false, true)
			if fnshd {
				instr.LineReplace(value, false)
			}

			if div.IsOver(ui) {
				value = ui.win.io.touch.drop_path //rewrite 'value'!
				if value != "" {
					instr.LineReplace(value, false)
				}
			}
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
				pathAttr.SetErrorExe(fmt.Sprintf("ReadFile(%s) failed: %v", value, err))
			}
		}
	}
}

func SAExe_Render_Map(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	file := w.GetAttr("file", "\"maps/osm\"").GetString()
	url := w.GetAttr("url", "\"https://tile.openstreetmap.org/{z}/{x}/{y}.png\"").GetString()
	copyright := w.GetAttr("copyright", "\"(c)OpenStreetMap contributors\"").GetString()
	copyright_url := w.GetAttr("copyright_url", "\"https://www.openstreetmap.org/copyright\"").GetString()

	cam_lonAttr := w.GetAttr("lon", "14.4071117049")
	cam_latAttr := w.GetAttr("lat", "50.0852013259")
	cam_zoomAttr := w.GetAttr("zoom", "5")

	//locators
	locatorsAttr := w.GetAttr("locators", "[{\"label\":\"1\", \"lon\":14.4071117049, \"lat\":50.0852013259}, {\"label\":\"2\", \"lon\":14, \"lat\":50}]")

	//segments
	segmentsAttr := w.GetAttr("segments", "[{\"label\":\"ABC\", \"Trkpt\":[{\"lat\":50,\"lon\":16,\"ele\":400,\"time\":\"2020-04-15T09:05:20Z\"},{\"lat\":50.4,\"lon\":16.1,\"ele\":400,\"time\":\"2020-04-15T09:05:23Z\"}]}]")

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
				w.errExe = err
			}

			if changed {
				//set back
				cam_lonAttr.SetExpFloat(cam_lon)
				cam_latAttr.SetExpFloat(cam_lat)
				cam_zoomAttr.SetExpInt(int(cam_zoom))
			}

			//locators
			locatorsBlob := locatorsAttr.GetBlob()
			if locatorsBlob.Len() > 0 {
				var locators []UiCompMapLocator
				err := json.Unmarshal(locatorsBlob.data, &locators)
				if err == nil {
					err = ui.comp_mapLocators(cam_lon, cam_lat, cam_zoom, locators, w.getPath())
					if err != nil {
						locatorsAttr.SetErrorExe(fmt.Sprintf("comp_mapLocators() failed: %v", err))
					}
				} else {
					locatorsAttr.SetErrorExe(fmt.Sprintf("Unmarshal() failed: %v", err))
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
						segmentsAttr.SetErrorExe(fmt.Sprintf("comp_mapSegments() failed: %v", err))
					}
				} else {
					segmentsAttr.SetErrorExe(fmt.Sprintf("Unmarshal() failed: %v", err))
				}
			}
			ui.Div_end()
		}
	}
}

func SAExe_Render_List(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	itemsAttr := w.GetAttr("items", "[0, 1, 2, 3, 4, 5]")
	multiSelect := w.GetAttrUi("multi_select", "1", SAAttrUi_CHECKBOX).GetBool()
	direction := w.GetAttrUi("direction", "0", SAAttrUi_COMBO("Vertical;Horizonal", "")).GetBool()
	maxItemSize := w.GetAttr("max_item_size", "[100, 1]").GetV2f()

	single_selectedAttr := w.GetAttr("single_selected", "-1")   //input, but can replace(instr)
	multi_selectedAttr := w.GetAttr("multi_selected", "[2, 3]") //input, but can replace(instr)

	var items []interface{}
	err := json.Unmarshal(itemsAttr.GetBlob().data, &items)
	if err != nil {
		itemsAttr.SetErrorExe(err.Error())
		return
	}

	var single_selected int
	var multi_selected []int
	if multiSelect {
		err = json.Unmarshal(multi_selectedAttr.GetBlob().data, &multi_selected)
		if err != nil {
			multi_selectedAttr.SetErrorExe(err.Error())
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
						multi_selectedAttr.SetExpString(str, true)
					} else {
						single_selectedAttr.SetExpInt(OsTrn(isSelected, -1, i))
					}
				}

				i++
			}
		}

		ui.Div_end()
	}
}

func SAExe_Render_Microphone(w *SANode, renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttrUi("enable", "1", SAAttrUi_SWITCH).GetBool()

	activeAttr := w.GetAttrUi("active", "0", SAAttrUi_SWITCH)
	audioAttr := w.GetAttr("_audio", "")
	audioAttr.SetOutBlob(nil) //reset

	if showIt {
		active := activeAttr.GetBool()
		cd := CdPalette_B
		if active {
			cd = CdPalette_P
		}
		if ui.Comp_buttonIcon(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, InitWinMedia_url("file:apps/base/resources/mic.png"), 0.3, "Enable/Disable audio recording", cd, enable, active) > 0 {
			active = !active
			activeAttr.SetExpBool(active)
		}

		if active {
			if w.app.base.ui.win.io.ini.MicOff {
				audioAttr.SetErrorExe("Microphone is disabled in SkyAlt Settings")
				return
			}

			w.app.base.mic_actived = true

			buf := new(bytes.Buffer)
			err := binary.Write(buf, binary.LittleEndian, w.app.base.mic_data)
			if err == nil {
				audioAttr.SetOutBlob(buf.Bytes())
			}

			ui.buff.ResetHost() //no sleep - is it working? .....
		}
	}

}

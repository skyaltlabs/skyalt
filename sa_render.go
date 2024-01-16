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
	"strings"
)

func (w *SANode) SARender_Dialog(renderIt bool) bool {
	ui := w.app.base.ui
	showIt := renderIt && ui != nil

	triggerAttr := w.GetAttr("trigger", "uiSwitch(0)")
	typeAttr := w.GetAttr("type", "uiCombo(0, \"Center;Relative\")")

	if showIt {
		if triggerAttr.GetBool() {
			ui.Dialog_open(w.Name, uint8(OsClamp(typeAttr.GetInt(), 0, 2)))
			triggerAttr.SetExpBool(false)
		}

		if ui.Dialog_start(w.Name) {
			w.renderLayout()
			ui.Dialog_end()
		}
	}

	return true
}

func (w *SANode) SARender_Button(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttr("enable", "uiSwitch(1)").GetBool()
	tp := w.GetAttr("type", "uiCombo(0, \"Classic;Light;Menu;Segments\")").GetInt()
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
		selected := w.GetAttr("selected", "uiSwitch(0)").GetBool()

		if showIt {
			clicked = ui.Comp_buttonMenu(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, "", enable, selected) > 0
			if clicked {
				sel := w.findAttr("selected")
				sel.SetExpBool(!selected)
			}
		}

	case 3:
		labels := label
		butts := strings.Split(labels, ";")
		selected := w.GetAttr("selected", fmt.Sprintf("uiCombo(0, %s)", labels)).GetInt()

		if showIt {
			ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
			{
				for i := range butts {
					ui.Div_colMax(i*2+0, 100)
					if i+1 < len(butts) {
						ui.Div_col(i*2+1, 0.1)
					}
				}
				for i, it := range butts {
					clicked = ui.Comp_buttonText(i*2+0, 0, 1, 1, it, "", "", enable, selected == i) > 0
					if clicked {
						sel := w.findAttr("selected")
						sel.SetExpInt(i)
					}
					if i+1 < len(butts) {
						ui.Div_SpacerCol(i*2+1, 0, 1, 1)
					}
				}
				//ui.Paint_rect(0, 0, 1, 1, 0, ui.buff.win.io.GetPalette().GetGrey(0.5), 0.03)
			}
			ui.Div_end()
		}
	}

	w.GetAttr("clicked", "uiSwitch(0)").GetBool()
	cl := w.findAttr("clicked")
	cl.Value = OsTrnString(clicked, "1", "0")
}

func (w *SANode) SARender_Text(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	label := w.GetAttr("label", "").GetString()
	align := w.GetAttr("align", "uiCombo(0, \"Left;Center;Right\")").GetInt()

	if showIt {
		ui.Comp_text(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, align)
	}
}

func (w *SANode) SARender_Switch(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	label := w.GetAttr("label", "").GetString()
	instr := w.GetAttr("value", "").instr.GetConst()
	value := instr.pos_attr.result.String()
	enable := w.GetAttr("enable", "uiSwitch(1)").GetBool() && instr != nil

	if showIt {
		if ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, "", enable) {
			instr.LineReplace(value)
		}
	}
}

func (w *SANode) SARender_Checkbox(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	label := w.GetAttr("label", "").GetString()
	instr := w.GetAttr("value", "").instr.GetConst()
	value := instr.pos_attr.result.String()
	enable := w.GetAttr("enable", "uiSwitch(1)").GetBool() && instr != nil

	if showIt {
		if ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, "", enable) {
			instr.LineReplace(value)
		}
	}
}

func (w *SANode) SARender_Combo(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	instr := w.GetAttr("value", "").instr.GetConst()
	value := instr.pos_attr.result.String()
	enable := w.GetAttr("enable", "uiSwitch(1)").GetBool() && instr != nil
	options := w.GetAttr("options", "\"a;b;c\")").GetString()
	search := w.GetAttr("search", "uiSwitch(0)").GetBool()

	if showIt {
		if ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, options, "", enable, search) {
			instr.LineReplace(value)
		}
	}
}

func (w *SANode) SARender_Editbox(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	instr := w.GetAttr("value", "").instr.GetConst()
	value := instr.pos_attr.result.String()
	enable := w.GetAttr("enable", "uiSwitch(1)").GetBool() && instr != nil
	tmpToValue := w.GetAttr("tempToValue", "uiSwitch(0)").GetBool()
	precision := w.GetAttr("precision", "2").GetInt()
	ghost := w.GetAttr("ghost", "").GetString()

	if showIt {
		_, _, chngd, fnshd, _ := ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, precision, nil, ghost, false, tmpToValue, enable)
		if fnshd || (tmpToValue && chngd) {
			instr.LineReplace(value)
		}
	}
}

func (w *SANode) SARender_Divider(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	tp := w.GetAttr("type", "uiCombo(0, \"Column;Row\"").GetInt()

	if showIt {
		switch tp {
		case 0:
			ui.Div_SpacerCol(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		case 1:
			ui.Div_SpacerRow(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		}
	}
}

func (w *SANode) SARender_ColorPalette(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	cdAttr := w.GetAttr("cd", "uiColor([0, 0, 0, 255])")
	cd := cdAttr.result.GetCd()

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

func (w *SANode) SARender_Color(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttr("enable", "uiSwitch(1)").GetBool()
	cdAttr := w.GetAttr("cd", "uiColor([0, 0, 0, 255])")
	cd := cdAttr.result.GetCd()

	if showIt {
		if ui.comp_colorPicker(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &cd, w.Name, enable) {
			cdAttr.ReplaceCd(cd)
		}
	}
}

func (w *SANode) SARender_Calendar(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	valueAttr := w.GetAttr("value", "uiDate(0)")
	pageAttr := w.GetAttr("page", "uiDate(0)")

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := valueAttr.GetInt64()
			page := pageAttr.GetInt64()

			ui.Comp_Calendar(&value, &page, 100, 100)

			valueAttr.SetExpInt(int(value))
			pageAttr.SetExpInt(int(page))
		}
		ui.Div_end()
	}
}

func (w *SANode) SARender_Date(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttr("enable", "uiSwitch(1)").GetBool()
	valueAttr := w.GetAttr("value", "uiDate(0)")
	show_time := w.GetAttr("show_time", "uiSwitch(0)").GetBool()

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := valueAttr.GetInt64()
			if ui.Comp_CalendarDataPicker(&value, show_time, w.Name, enable) {
				valueAttr.SetExpInt(int(value))
			}
		}
		ui.Div_end()
	}
}

func (w *SANode) SARender_Map(renderIt bool) {
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
	segmentsAttr := w.GetAttr("segments", "[{\"Trkpt\":[{\"lat\":50,\"lon\":16,\"ele\":400,\"time\":\"2020-04-15T09:05:20Z\"},{\"lat\":50.4,\"lon\":16.1,\"ele\":400,\"time\":\"2020-04-15T09:05:23Z\"}]}]")

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

			err := ui.comp_map(&cam_lon, &cam_lat, &cam_zoom, file, url, copyright, copyright_url)
			if err != nil {
				w.errExe = err
			}

			//set back
			cam_lonAttr.SetExpFloat(cam_lon)
			cam_latAttr.SetExpFloat(cam_lat)
			cam_zoomAttr.SetExpInt(int(cam_zoom))

			//locators
			locatorsBlob := locatorsAttr.result.Blob()
			if len(locatorsBlob) > 0 {
				var locators []UiCompMapLocator
				err := json.Unmarshal(locatorsBlob, &locators)
				if err == nil {
					err = ui.comp_mapLocators(cam_lon, cam_lat, cam_zoom, locators)
					if err != nil {
						locatorsAttr.SetErrorExe(fmt.Sprintf("comp_mapLocators() failed: %v", err))
					}
				} else {
					locatorsAttr.SetErrorExe(fmt.Sprintf("Unmarshal() failed: %v", err))
				}
			}

			//paths
			segmentsBlob := segmentsAttr.result.Blob()
			if len(segmentsBlob) > 0 {
				var segments []UiCompMapSegments
				err := json.Unmarshal(segmentsBlob, &segments)
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

func (w *SANode) SARender_Image(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	margin := w.GetAttr("margin", "0").result.Number()
	cd := w.GetAttr("cd", "uiColor([255, 255, 255, 255])").result.GetCd()
	background := w.GetAttr("background", "uiSwitch(0)").GetBool()

	alignV := w.GetAttr("alignV", "uiCombo(1, \"Left;Center;Right\")").GetInt()
	alignH := w.GetAttr("alignH", "uiCombo(1, \"Left;Center;Right\")").GetInt()
	fill := w.GetAttr("fill", "uiSwitch(0)").GetBool()

	blobAttr := w.GetAttr("blob", "uiBlob(0)")

	if !renderIt {
		if !blobAttr.result.IsBlob() {
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
			blob := blobAttr.result.Blob()
			path := InitWinMedia_blob(blob)
			ui.Paint_file(0, 0, 1, 1, margin, path, cd, alignV, alignH, fill, background)
		}
		ui.Div_end()
	}
}

func (w *SANode) SARender_File(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	pathAttr := w.GetAttr("path", "")
	instr := pathAttr.instr.GetConst()
	value := instr.pos_attr.result.String()

	outputAttr := w.GetAttrOutput("output", "uiBlob(0)")

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
				instr.LineReplace(value)
			}

			if div.IsOver(ui) {
				value = ui.win.io.touch.drop_path //rewrite 'value'!
				if value != "" {
					instr.LineReplace(value)
				}
			}
		}
		ui.Div_end()
	}

	if !renderIt {
		outputAttr.result.SetBlob(nil) //reset

		if value != "" {
			data, err := os.ReadFile(value)
			if err == nil {
				outputAttr.result.SetBlob(data)
			} else {
				pathAttr.SetErrorExe(fmt.Sprintf("ReadFile(%s) failed: %v", value, err))
			}
		}
	}
}

func (w *SANode) SARender_Layout(renderIt bool) {
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

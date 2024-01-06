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
	"strings"
)

func (w *SANode) SARender_Dialog(renderIt bool) bool {
	ui := w.app.base.ui
	showIt := renderIt && ui != nil

	triggerAttr := w.GetAttr("trigger", "bool(0)")
	typeAttr := w.GetAttr("type", "combo(0, \"Center;Relative\")")

	if showIt {
		if triggerAttr.GetBool() {
			ui.Dialog_open(w.Name, uint8(OsClamp(typeAttr.GetInt(), 0, 2)))
			triggerAttr.SetExpBool(false)
		}

		if ui.Dialog_start(w.Name) {
			w.RenderLayout()
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

	enable := w.GetAttr("enable", "bool(1)").GetBool()
	tp := w.GetAttr("type", "combo(0, \"Classic;Light;Menu;Segments\")").GetInt()
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
		selected := w.GetAttr("selected", "bool(0)").GetBool()

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
		selected := w.GetAttr("selected", fmt.Sprintf("combo(0, %s)", labels)).GetInt()

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

	w.GetAttr("clicked", "bool(0)").GetBool()
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
	align := w.GetAttr("align", "combo(0, \"Left;Center;Right\")").GetInt()

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
	value := instr.pos_attr.finalValue.String()
	enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil

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
	value := instr.pos_attr.finalValue.String()
	enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil

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
	value := instr.pos_attr.finalValue.String()
	enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil
	options := w.GetAttr("options", "\"a;b;c\")").GetString()
	search := w.GetAttr("search", "bool(0)").GetBool()

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
	value := instr.pos_attr.finalValue.String()
	enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil
	tmpToValue := w.GetAttr("tempToValue", "bool(0)").GetBool()
	precision := w.GetAttr("precision", "2").GetInt()
	ghost := w.GetAttr("ghost", "").GetString()

	if showIt {
		_, _, chngd, fnshd, _ := ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, precision, "", ghost, false, tmpToValue, enable)
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

	tp := w.GetAttr("type", "combo(0, \"Column;Row\"").GetInt()

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

	cdAttr := w.GetAttr("cd", "color([0, 0, 0, 255])")
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

func (w *SANode) SARender_Color(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	enable := w.GetAttr("enable", "bool(1)").GetBool()
	cdAttr := w.GetAttr("cd", "color([0, 0, 0, 255])")
	cd := cdAttr.GetCd()

	if showIt {
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			if ui.comp_colorPicker(&cd, w.Name, enable) {
				cdAttr.ReplaceCd(cd)
			}
		}
		ui.Div_end()
	}
}

func (w *SANode) SARender_Calendar(renderIt bool) {
	ui := w.app.base.ui
	showIt := renderIt && w.CanBeRenderOnCanvas() && w.GetGridShow() && ui != nil

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	valueAttr := w.GetAttr("value", "date(0)")
	pageAttr := w.GetAttr("page", "date(0)")

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

	enable := w.GetAttr("enable", "bool(1)").GetBool()
	valueAttr := w.GetAttr("value", "date(0)")
	show_time := w.GetAttr("show_time", "bool(0)").GetBool()

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
	locatorsAttr := w.GetAttr("locators", "{\"lon;lat;label\", 14.4071117049, 50.0852013259, \"1\", 14, 50, \"2\"}")
	locrs := locatorsAttr.finalValue.Table()
	lon_i := locrs.FindName("lon")
	lat_i := locrs.FindName("lat")
	label_i := locrs.FindName("label")
	if lon_i < 0 {
		locatorsAttr.SetErrorExe("'lon' column not found")
	}
	if lat_i < 0 {
		locatorsAttr.SetErrorExe("'lat' column not found")
	}
	if label_i < 0 {
		locatorsAttr.SetErrorExe("'label_i' column not found")
	}

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
			{
				var locators []UiCompMapLocator
				for r := 0; r < locrs.NumRows(); r++ {
					locators = append(locators, UiCompMapLocator{lon: locrs.Get(lon_i, r).Number(), lat: locrs.Get(lat_i, r).Number(), label: locrs.Get(label_i, r).String()})
				}

				err := ui.comp_mapLocators(cam_lon, cam_lat, cam_zoom, locators)
				if err != nil {
					locatorsAttr.errExe = err
				}
			}
			ui.Div_end()
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
		w.RenderLayout()
		ui.Div_end()
	}
}

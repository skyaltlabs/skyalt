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

// rename Flamingo -> Assistant .....

// most natural would be just talk to mic and point(select rect) with mouse ......

type SAFlamingo struct {
	app *SAApp

	drawActive  bool
	drawStart   OsV2
	buildCoord  OsV4
	buildActive bool

	items []*SANodeAttr

	doc    string
	ask    string
	result string

	//history? ...
}

func NewSAFlamingo(app *SAApp) *SAFlamingo {
	flg := &SAFlamingo{}
	flg.app = app
	return flg
}

func (flg *SAFlamingo) Destroy() {

}

func (flg *SAFlamingo) GetCd() OsCd {
	return InitOsCd32(3, 232, 252, 255)
}

func (flg *SAFlamingo) addItem(attr *SANodeAttr) {
	//find
	for _, it := range flg.items {
		if it == attr {
			return
		}
	}

	//add
	flg.items = append(flg.items, attr)
}

func (flg *SAFlamingo) tryAddItem(attr *SANodeAttr, crop OsV4) {
	if !flg.buildActive {
		return
	}

	selectArea := float64(flg.buildCoord.Area())
	nodeArea := float64(crop.Area())
	interArea := float64(flg.buildCoord.GetIntersect(crop).Area())

	MX := 0.8 //80%
	if selectArea > 0 && nodeArea > 0 && interArea/selectArea > MX || interArea/nodeArea > MX {
		flg.addItem(attr)
	}
}

func (flg *SAFlamingo) TryAddItemFromDiv(div *UiLayoutDiv, attr *SANodeAttr) {
	if !flg.buildActive {
		return
	}

	if !div.enableInput {
		return //not top level
	}

	flg.tryAddItem(attr, div.crop)
}

func (flg *SAFlamingo) TryAddItemFromAttr(attr *SANodeAttr) {
	if !flg.buildActive {
		return
	}

	grid := attr.node.GetGrid()
	ui := flg.app.base.ui

	div := ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	flg.TryAddItemFromDiv(div, attr)
	ui.Div_end()
}

func (flg *SAFlamingo) buildDoc() {

	//uložit node atd., abych mohl později promítnout result ...

	flg.doc = "" //...
}

func (flg *SAFlamingo) Tick() {
	ui := flg.app.base.ui
	keys := ui.win.io.keys
	touch := ui.win.io.touch

	if flg.buildActive {
		flg.buildDoc()
		flg.buildActive = false
		ui.Dialog_open("flamingo_settings", 0)
	}

	//dialog
	if ui.Dialog_start("flamingo_settings") {

		br := 1.0
		ui.Div_col(0, br)
		ui.Div_colMax(1, 20)
		ui.Div_col(2, br)

		ui.Div_row(0, 1) //title
		ui.Div_rowMax(1, 10)
		ui.Div_row(2, br)

		ui.Comp_text(1, 0, 1, 1, "Flamingo", 1)

		ui.Div_start(1, 1, 1, 1) //mid
		{
			ui.Div_colMax(0, 2)
			ui.Div_colMax(1, 100)
			ui.Div_rowMax(0, 100)
			ui.Div_rowMax(3, 100)

			ui.Comp_textSelect(0, 0, 1, 1, "Ask", OsV2{0, 0}, true, false)
			ui.Comp_textSelect(0, 3, 1, 1, "Result", OsV2{0, 0}, true, false)

			ui.Comp_editbox(1, 0, 1, 1, &flg.ask, Comp_editboxProp().MultiLine(true).Align(0, 0).Ghost("prompt")) //prompt

			if ui.Comp_button(1, 1, 1, 1, "Execute", "", true) > 0 {
				//g4f := flg.app.base.services.GetG4F()
				//run ...
			}
			ui.Comp_progress(1, 2, 1, 1, 0.0, 0, "", false)

			ui.Comp_editbox(1, 3, 1, 1, flg.result, Comp_editboxProp().MultiLine(true).Align(0, 0).Ghost("result")) //result

			//send to app
			if ui.Comp_button(1, 4, 1, 1, "Apply", "", true) > 0 {
				//result -> SkyAlt ...
			}

		}
		ui.Div_end()

		ui.Dialog_end()
	} else if ui.Dialog_startEx("flamingo", false, false) {
		ui.Div_colMax(0, 100)
		ui.Div_rowMax(0, 100)

		cd := flg.GetCd()

		if !flg.drawActive && touch.start {
			//start selection
			flg.drawActive = true
			flg.drawStart = touch.pos
		}

		if flg.drawActive {
			//continue selection
			coord := InitOsV4ab(flg.drawStart, touch.pos)
			rad := ui.win.Cell() / 8

			cd.A = 20
			ui.buff.AddRectRound(coord, rad, cd, 0) //back
			cd.A = 255
			ui.buff.AddRectRound(coord, rad, cd, ui.CellWidth(0.03)) //border

			//end selection
			if touch.end {
				flg.ask = ""
				flg.result = ""
				ui.Dialog_close()
				flg.drawActive = false
				flg.buildActive = true
				flg.buildCoord = coord
				flg.items = nil
			}
		}

		ui.Paint_rect(0, 0, 1, 1, 0.06, cd, 0.03) //border

		if !keys.ctrl {
			ui.Dialog_close()
		}

		ui.Dialog_end()
	} else {
		if keys.ctrl {
			ui.Dialog_open("flamingo", 0)
		}
	}

}

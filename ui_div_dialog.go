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

import "errors"

func (ui *Ui) Dialog_close() {
	ui.CloseAndAbove(ui.GetCall())
}

func (ui *Ui) Dialog_end() {

	//fmt.Println(OsTime() - a)

	lv := ui.GetCall()

	//close dialog
	if lv.call.enableInput {
		winRect, _ := ui.win.GetScreenCoord()
		outside := winRect.Inside(ui.win.io.touch.pos) && !lv.base.canvas.Inside(ui.win.io.touch.pos)
		if (ui.win.io.touch.end && outside) || ui.win.io.keys.esc {
			ui.touch.Reset()
			ui.CloseAndAbove(lv)
			ui.win.io.keys.esc = false
		}
	}

	ui.renderEnd(true)

	err := ui.buff.DialogEnd()
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	}
	/*err := ui.buff.EndLevel()
	if err != nil {
		app.AddLogErr(err)
	}*/

	err = ui.EndCall()
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	}

	lv = ui.GetCall()
	ui.buff.AddCrop(lv.call.CropWithScroll(ui.win))
}

func (ui *Ui) Dialog_open(name string, tp uint8) bool {
	lv := ui.GetCall()

	//name
	if len(name) == 0 {
		return false
	}

	//find
	act := ui.Find(name)
	if act != nil {
		lv.call.data.app.AddLogErr(errors.New("dialog already opened"))
		return false //already open
	}

	//coord
	var src_coordMoveCut OsV4
	switch tp {
	case 1:
		if lv.call.lastChild != nil {
			src_coordMoveCut = lv.call.lastChild.crop
		} else {
			src_coordMoveCut = lv.call.crop
		}
	case 2:
		src_coordMoveCut = OsV4{Start: ui.win.io.touch.pos, Size: OsV2{1, 1}}
	}

	//add
	ui.AddDialog(name, src_coordMoveCut, ui.win)
	ui.touch.Reset()
	ui.win.io.ResetTouchAndKeys()
	ui.edit.setFirstEditbox = true

	return true
}

func (ui *Ui) Dialog_start(name string) bool {
	lv := ui.GetCall()

	//name
	if len(name) == 0 {
		return false
	}

	//find
	lev := ui.Find(name)
	if lev == nil {
		return false //dialog not open, which is NOT error
	}

	if lev.use == 1 {
		lv.call.data.app.AddLogErr(errors.New("dialog already drawn into"))
		return false
	}
	lev.use = 1

	/*if lev.src_coordMoveCut.Size.X > 1 || lev.src_coordMoveCut.Size.Y > 1 { //for tp==1
		if lv.call.lastChild != nil {
			lev.src_coordMoveCut = lv.call.lastChild.crop
		} else {
			lev.src_coordMoveCut = lv.call.crop
		}
	}*/

	//coord
	winRect, _ := ui.win.GetScreenCoord()

	coord := lev.base.GetLevelSize(winRect, ui.win)
	coord = lev.GetCoord(coord, winRect)
	lev.base.canvas = coord
	lev.base.crop = coord

	ui.StartCall(lev)

	ui.renderStart(0, 0, 1, 1, true)
	//a = OsTime()

	err := ui.buff.DialogStart(coord) //rewrite buffer with background
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	}

	return true //active/open
}

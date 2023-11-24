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

func (levels *UiLayoutLevels) Dialog_close() {
	levels.CloseAndAbove(levels.GetCall())
}

func (levels *UiLayoutLevels) Dialog_end() {

	//fmt.Println(OsTime() - a)

	lv := levels.GetCall()

	//close dialog
	if lv.call.enableInput {
		winRect, _ := levels.win.GetScreenCoord()
		outside := winRect.Inside(levels.win.io.touch.pos) && !lv.base.canvas.Inside(levels.win.io.touch.pos)
		if (levels.win.io.touch.end && outside) || levels.win.io.keys.esc {
			levels.touch.Reset()
			levels.CloseAndAbove(lv)
			levels.win.io.keys.esc = false
		}
	}

	levels.renderEnd(true)

	err := levels.buff.DialogEnd()
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	}
	/*err := levels.buff.EndLevel()
	if err != nil {
		app.AddLogErr(err)
	}*/

	err = levels.EndCall()
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	}

}

func (levels *UiLayoutLevels) Dialog_open(name string, tp uint8) bool {
	lv := levels.GetCall()

	//name
	if len(name) == 0 {
		return false
	}

	//find
	act := levels.Find(name)
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
		src_coordMoveCut = OsV4{Start: levels.win.io.touch.pos, Size: OsV2{1, 1}}
	}

	//add
	levels.AddDialog(name, src_coordMoveCut, levels.win)
	levels.touch.Reset()
	levels.win.io.ResetTouchAndKeys()
	levels.edit.setFirstEditbox = true

	return true
}

func (levels *UiLayoutLevels) Dialog_start(name string) bool {
	lv := levels.GetCall()

	//name
	if len(name) == 0 {
		return false
	}

	//find
	lev := levels.Find(name)
	if lev == nil {
		return false //dialog not open, which is NOT error
	}

	if lev.use == 1 {
		lv.call.data.app.AddLogErr(errors.New("dialog already drawn into"))
		return false
	}
	lev.use = 1

	if lev.src_coordMoveCut.Size.X > 1 || lev.src_coordMoveCut.Size.Y > 1 { //for tp==1
		if lv.call.lastChild != nil {
			lev.src_coordMoveCut = lv.call.lastChild.crop
		} else {
			lev.src_coordMoveCut = lv.call.crop
		}
	}

	//coord
	winRect, _ := levels.win.GetScreenCoord()

	coord := lev.base.GetLevelSize(winRect, levels.win)
	coord = lev.GetCoord(coord, winRect)
	lev.base.canvas = coord
	lev.base.crop = coord

	levels.StartCall(lev)

	levels.renderStart(0, 0, 1, 1, true)
	//a = OsTime()

	err := levels.buff.DialogStart(coord) //rewrite buffer with background
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	}

	return true //active/open
}

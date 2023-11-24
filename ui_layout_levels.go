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

import "fmt"

type UiLayoutEdit struct {
	uid, next_uid        *UiLayoutDiv
	setFirstEditbox, tab bool

	temp, orig string
	start, end OsV2

	last_edit string //for every SA_Editbox call

	tempToValue bool
}

type UiLayoutDrag struct {
	div   *UiLayoutDiv
	group string
	id    uint64 //maybe string? ...
}

type UiLayoutLevels struct {
	win *Win

	dialogs []*UiLayoutLevel
	calls   []*UiLayoutLevel

	buff *WinPaintBuff

	tile UiLayoutTile

	edit_history UiPaintTextHistoryArray
	edit         UiLayoutEdit
	drag         UiLayoutDrag
	touch        UiLayoutTouch

	app *UiLayoutApp
}

func NewUiLayoutLevels(app *UiLayoutApp, win *Win) (*UiLayoutLevels, error) {
	var levels UiLayoutLevels
	levels.win = win
	levels.app = app

	levels.buff = NewWinPaintBuff(win)

	levels.AddDialog("", OsV4{}, win)

	return &levels, nil
}

func (levels *UiLayoutLevels) Destroy() {
	levels.buff.Destroy()

	for _, l := range levels.dialogs {
		l.Destroy()
	}
	levels.dialogs = nil
	levels.calls = nil
}

func (levels *UiLayoutLevels) CellWidth(width float64) int {
	t := int(width * float64(levels.win.Cell())) // cell is ~34
	if width > 0 && t <= 0 {
		t = 1 //at least 1px
	}
	return t
}

func (levels *UiLayoutLevels) Save() {
	for _, l := range levels.dialogs {
		l.base.Save()
	}
}

func (levels *UiLayoutLevels) AddDialog(name string, src_coordMoveCut OsV4, win *Win) {

	newDialog := NewUiLayoutLevel(name, src_coordMoveCut, levels.app, win)
	levels.dialogs = append(levels.dialogs, newDialog)

	//disable bottom dialogs
	for _, l := range levels.calls {
		enabled := (l == newDialog)
		div := l.call
		for div != nil {
			div.enableInput = enabled
			div = div.parent
		}
	}

}

func (levels *UiLayoutLevels) StartCall(lev *UiLayoutLevel) {
	//init level
	lev.call = lev.base

	//add
	levels.calls = append(levels.calls, lev)
}
func (levels *UiLayoutLevels) EndCall() error {

	n := len(levels.calls)
	if n > 1 {
		levels.calls = levels.calls[:n-1]
		return nil
	}

	return fmt.Errorf("trying to EndCall from root level")
}

func (levels *UiLayoutLevels) isSomeClose() bool {
	for _, l := range levels.dialogs {
		if l.use == 0 || l.close {
			return true
		}
	}
	return false
}

func (levels *UiLayoutLevels) Maintenance() {

	levels.GetBaseDialog().use = 1 //base level is always use

	//remove unused or closed
	if levels.isSomeClose() {
		var lvls []*UiLayoutLevel
		for _, l := range levels.dialogs {
			if l.use != 0 && !l.close {
				lvls = append(lvls, l)
			}
		}
		levels.dialogs = lvls

	}

	//layout
	for _, l := range levels.dialogs {
		l.base.Maintenance()
		l.use = 0
	}
}

func (levels *UiLayoutLevels) ResetGridLocks() {
	for _, l := range levels.dialogs {
		l.base.ResetGridLock()
	}
}

func (levels *UiLayoutLevels) CloseAndAbove(dialog *UiLayoutLevel) {

	found := false
	for _, l := range levels.dialogs {
		if l == dialog {
			found = true
		}
		if found {
			l.close = true
		}
	}
}
func (levels *UiLayoutLevels) CloseAll() {

	if len(levels.dialogs) > 1 {
		levels.CloseAndAbove(levels.dialogs[1])
	}
}

func (levels *UiLayoutLevels) GetBaseDialog() *UiLayoutLevel {
	return levels.dialogs[0]
}

func (levels *UiLayoutLevels) GetCall() *UiLayoutLevel {
	return levels.calls[len(levels.calls)-1] //last call
}

func (levels *UiLayoutLevels) IsStackTop() bool {
	return levels.dialogs[len(levels.dialogs)-1] == levels.GetCall() //last dialog
}

func (levels *UiLayoutLevels) ResetStack() {
	levels.calls = nil
	levels.StartCall(levels.GetBaseDialog())
}

func (levels *UiLayoutLevels) Find(name string) *UiLayoutLevel {

	for _, l := range levels.dialogs {
		if l.name == name {
			return l
		}
	}
	return nil
}

func (levels *UiLayoutLevels) UpdateTile(win *Win) bool {

	redraw := false

	if levels.tile.NeedsRedrawFromSleep(win.io.touch.pos) {
		redraw = true
	}
	levels.tile.NextTick()

	return redraw
}

func (levels *UiLayoutLevels) RenderTile(win *Win) {

	if levels.tile.IsActive(win.io.touch.pos) {
		err := win.RenderTile(levels.tile.text, levels.tile.coord, levels.tile.priorUp, levels.tile.cd, win.fonts.Get(SKYALT_FONT_PATH))
		if err != nil {
			fmt.Printf("RenderTile() failed: %v\n", err)
		}
	}

}

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
)

type UiLayoutEdit struct {
	uid, next_uid        *UiLayoutDiv
	setFirstEditbox, tab bool

	temp, orig string
	start, end OsV2

	last_edit string //for every SA_Editbox call

	tempToValue bool
}

func (edit *UiLayoutEdit) IsActive() bool {
	return edit.uid != nil
}

type UiLayoutDrag struct {
	div   *UiLayoutDiv
	group string
	id    int //maybe string? ...
}

type Ui struct {
	win *Win

	dialogs []*UiLayoutLevel
	calls   []*UiLayoutLevel

	buff *WinPaintBuff

	tile UiLayoutTile

	edit_history UiPaintTextHistoryArray
	edit         UiLayoutEdit
	drag         UiLayoutDrag
	touch        UiLayoutTouch

	base_app  *UiLayoutApp
	app_calls []*UiLayoutApp

	trns UiTranslations

	date_page int64

	mapp *UiLayoutMap
}

func NewUi(win *Win, base_app_layout_path string) (*Ui, error) {
	var ui Ui
	ui.win = win

	ui.mapp = NewUiLayoutMap()

	ui.base_app = NewUiLayoutApp("base", base_app_layout_path)

	ui.buff = NewWinPaintBuff(win)

	ui.AddDialog("", OsV4{}, win)

	//translations
	err := ui.reloadTranslations()
	if err != nil {
		return nil, fmt.Errorf("reloadTranslations() failed: %w", err)
	}

	return &ui, nil
}

func (ui *Ui) Destroy() {

	ui.mapp.Destroy()

	ui.base_app.Destroy()

	ui.buff.Destroy()

	for _, l := range ui.dialogs {
		l.Destroy()
	}
	ui.dialogs = nil
	ui.calls = nil
}

func (ui *Ui) reloadTranslations() error {
	js, err := UiTranslations_fromJsonFile("apps/base/translations.json", ui.win.io.ini.Languages)
	if err != nil {
		return fmt.Errorf("reloadTranslations() failed: %w", err)
	}
	err = json.Unmarshal(js, &ui.trns)
	if err != nil {
		fmt.Printf("Unmarshal() failed: %v\n", err)
	}
	return nil
}

func (ui *Ui) CellWidth(width float64) int {
	t := int(width * float64(ui.win.Cell())) // cell is ~34
	if width > 0 && t <= 0 {
		t = 1 //at least 1px
	}
	return t
}

func (ui *Ui) Save(base_app_layout_path string) {
	for _, l := range ui.dialogs {
		l.base.Save()
	}

	js, err := ui.base_app.Save()
	if err == nil {
		err = os.WriteFile(base_app_layout_path, js, 0644)
		if err != nil {
			fmt.Printf("WriteFile() failed: %v\n", err)
		}
	} else {
		fmt.Printf("Save() failed: %v\n", err)
	}

}

func (ui *Ui) AddDialog(name string, src_coordMoveCut OsV4, win *Win) {

	newDialog := NewUiLayoutLevel(name, src_coordMoveCut, ui.base_app, win)
	ui.dialogs = append(ui.dialogs, newDialog)

	//disable bottom dialogs
	for _, l := range ui.calls {
		enabled := (l == newDialog)
		div := l.call
		for div != nil {
			div.enableInput = enabled
			div = div.parent
		}
	}

}

func (ui *Ui) StartCall(lev *UiLayoutLevel) {
	//init level
	lev.call = lev.base

	//add
	ui.calls = append(ui.calls, lev)
}
func (ui *Ui) EndCall() error {

	n := len(ui.calls)
	if n > 1 {
		ui.calls = ui.calls[:n-1]
		return nil
	}

	return fmt.Errorf("trying to EndCall from root level")
}

func (ui *Ui) isSomeClose() bool {
	for _, l := range ui.dialogs {
		if l.use == 0 || l.close {
			return true
		}
	}
	return false
}

func (ui *Ui) Maintenance() {

	ui.GetBaseDialog().use = 1 //base level is always use

	//remove unused or closed
	if ui.isSomeClose() {
		var lvls []*UiLayoutLevel
		for _, l := range ui.dialogs {
			if l.use != 0 && !l.close {
				lvls = append(lvls, l)
			}
		}
		ui.dialogs = lvls

		ui.edit.setFirstEditbox = false
	}

	//layout
	for _, l := range ui.dialogs {
		l.base.Maintenance()
		l.use = 0
	}
}

func (ui *Ui) ResetGridLocks() {
	for _, l := range ui.dialogs {
		l.base.ResetGridLock()
	}
}

func (ui *Ui) CloseAndAbove(dialog *UiLayoutLevel) {

	found := false
	for _, l := range ui.dialogs {
		if l == dialog {
			found = true
		}
		if found {
			l.close = true
		}
	}
}
func (ui *Ui) CloseAll() {

	if len(ui.dialogs) > 1 {
		ui.CloseAndAbove(ui.dialogs[1])
	}
}

func (ui *Ui) GetBaseDialog() *UiLayoutLevel {
	return ui.dialogs[0]
}

func (ui *Ui) GetCall() *UiLayoutLevel {
	return ui.calls[len(ui.calls)-1] //last
}

func (ui *Ui) GetLastApp() *UiLayoutApp {
	return ui.app_calls[len(ui.app_calls)-1] //last
}

func (ui *Ui) IsStackTop() bool {
	return ui.dialogs[len(ui.dialogs)-1] == ui.GetCall() //last dialog
}

func (ui *Ui) ResetStack() {
	ui.calls = nil
	ui.StartCall(ui.GetBaseDialog())

	ui.app_calls = nil
	ui.app_calls = append(ui.app_calls, ui.base_app)
}

func (ui *Ui) Find(name string) *UiLayoutLevel {

	for _, l := range ui.dialogs {
		if l.name == name {
			return l
		}
	}
	return nil
}

func (ui *Ui) UpdateTile(win *Win) bool {

	redraw := false

	if ui.tile.NeedsRedrawFromSleep(win.io.touch.pos) {
		redraw = true
	}
	ui.tile.NextTick()

	return redraw
}

func (ui *Ui) RenderTile(win *Win) {

	if ui.tile.IsActive(win.io.touch.pos) {
		err := win.RenderTile(ui.tile.text, ui.tile.coord, ui.tile.priorUp, ui.tile.cd)
		if err != nil {
			fmt.Printf("RenderTile() failed: %v\n", err)
		}
	}
}

func (ui *Ui) StartRender() {
	winRect, _ := ui.win.GetScreenCoord()
	ui.GetBaseDialog().base.canvas = winRect
	ui.GetBaseDialog().base.crop = winRect

	// close all levels
	if ui.win.io.keys.shift && ui.win.io.keys.esc {
		ui.touch.Reset()
		ui.CloseAll()
		ui.win.io.keys.esc = false
	}

	ui.ResetStack()

	lv := ui.GetCall()
	ui.buff.Prepare(lv.call.canvas, true)

	if ui.win.io.touch.start {
		ui.touch.Reset()
	}
}

func (ui *Ui) EndRender() {

	if ui.win.io.touch.end {
		ui.touch.Reset()
		ui.drag.group = ""
	}

	// tile - redraw If mouse is over tile
	if ui.tile.IsActive(ui.win.io.touch.pos) {
		err := ui.win.RenderTile(ui.tile.text, ui.tile.coord, ui.tile.priorUp, ui.tile.cd)
		if err != nil {
			fmt.Printf("RenderTile() failed: %v\n", err)
		}
	}

	ui.Maintenance()
	ui.buff.FinalDraw()
}

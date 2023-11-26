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
	"strconv"
)

type SABase_translations struct {
	SAVE            string
	SETTINGS        string
	ZOOM            string
	WINDOW_MODE     string
	FULLSCREEN_MODE string
	ABOUT           string
	QUIT            string
	SEARCH          string

	COPYRIGHT string
	WARRANTY  string

	TIME_ZONE string

	DATE_FORMAT      string
	DATE_FORMAT_EU   string
	DATE_FORMAT_US   string
	DATE_FORMAT_ISO  string
	DATE_FORMAT_TEXT string

	THEME       string
	THEME_OCEAN string
	THEME_RED   string
	THEME_BLUE  string
	THEME_GREEN string
	THEME_GREY  string

	DPI        string
	SHOW_STATS string
	SHOW_GRID  string
	LANGUAGE   string
	LANGUAGES  string

	NAME        string
	REMOVE      string
	RENAME      string
	DUPLICATE   string
	VACUUM      string
	CREATE_FILE string
	CHANGE_APP  string

	SETUP_DB string

	ALREADY_EXISTS string
	EMPTY_FIELD    string
	INVALID_NAME   string

	IN_USE string

	ADD_APP   string
	CREATE_DB string

	DEVELOPERS    string
	CREATE_APP    string
	PACKAGE_APP   string
	REINSTALL_APP string
	VACUUM_DBS    string

	REPO    string
	PACKAGE string

	SIZE string
	LOGS string
}

type SABaseSettings struct {
	Apps     []string
	Selected int

	NewAppName string
}

func (sts *SABaseSettings) HasApp() bool {
	if sts.Selected >= len(sts.Apps) {
		sts.Selected = len(sts.Apps) - 1
	}
	return sts.Selected >= 0
}

func (sts *SABaseSettings) findApp(name string) int {
	for i, a := range sts.Apps {
		if a == name {
			return i
		}
	}
	return -1
}

func (sts *SABaseSettings) Refresh() {
	files := OsFileListBuild("apps", "", true)

	//add(new on disk)
	for _, f := range files.Subs {
		if !f.IsDir || f.Name == "base" {
			continue

		}
		if sts.findApp(f.Name) < 0 {
			sts.Apps = append(sts.Apps, f.Name)
		}
	}

	//remove(deleted from disk)
	for i := len(sts.Apps) - 1; i >= 0; i-- {
		if files.FindInSubs(sts.Apps[i], true) < 0 {
			sts.Apps = append(sts.Apps[:i], sts.Apps[i+1:]...)
		}
	}

	//remove duplicity
	for i := len(sts.Apps) - 1; i >= 0; i-- {

		if sts.findApp(sts.Apps[i]) != i {
			sts.Apps = append(sts.Apps[:i], sts.Apps[i+1:]...)
		}
	}
}

type SABase struct {
	ui_app *UiLayoutApp

	settings SABaseSettings

	exit bool

	trns SABase_translations
}

func NewSABase(ui *Ui) (*SABase, error) {
	var base SABase

	base.ui_app = ui.base_app

	//open
	{
		OsFolderCreate("apps")
		OsFolderCreate("apps/base")

		js, err := os.ReadFile(base.getPath())
		if err == nil {
			err := json.Unmarshal([]byte(js), &base.settings)
			if err != nil {
				fmt.Printf("warnning: Unmarshal() failed: %v\n", err)
			}
		}
	}

	//translations
	err := base.reloadTranslations(ui)
	if err != nil {
		return nil, fmt.Errorf("reloadTranslations() failed: %w", err)
	}

	base.settings.Refresh()

	return &base, nil
}

func (base *SABase) reloadTranslations(ui *Ui) error {

	js, err := SATranslations_fromJsonFile("apps/base/translations.json", ui.win.io.ini.Languages)
	if err != nil {
		return fmt.Errorf("SATranslations_fromJsonFile() failed: %w", err)
	}
	err = json.Unmarshal(js, &base.trns)
	if err != nil {
		fmt.Printf("warnning: Unmarshal() failed: %v\n", err)
	}
	return nil
}

func (base *SABase) Destroy() {

	//save
	{
		js, err := json.MarshalIndent(&base.settings, "", "")
		if err != nil {
			fmt.Printf("MarshalIndent() failed: %v\n", err)
		} else {
			err = os.WriteFile(base.getPath(), js, 0644)
			if err != nil {
				fmt.Printf("WriteFile() failed: %v\n", err)
			}
		}
	}

	base.ui_app.Destroy()
}

func (base *SABase) getPath() string {
	return "apps/base/app.json"
}

func (base *SABase) Render(ui *Ui) bool {

	base.settings.HasApp() //fix range

	ui.renderStart(0, 0, 1, 1, true)

	//hard/soft render ...
	base.drawFrame(ui)

	ui.renderEnd(true)

	return !base.exit
}

func (base *SABase) drawFrame(ui *Ui) {

	icon_rad := 1.7

	ui.Div_rowMax(0, 100)
	ui.Div_col(0, icon_rad)
	ui.Div_colMax(1, 100)
	ui.Div_colResize(2, "settings", 10)

	ui.Div_start(0, 0, 1, 1)
	base.drawIcons(ui, icon_rad)
	ui.Div_end()

	if base.settings.HasApp() {
		ui.Div_startName(1, 0, 1, 1, base.settings.Apps[base.settings.Selected])
		base.drawApp(ui)
		ui.Div_end()
	}

	ui.Div_start(2, 0, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_rowResize(0, "parameters", 10)
		ui.Div_rowMax(1, 100)

		ui.Div_start(0, 0, 1, 1)
		base.drawParameters(ui)
		ui.Div_end()

		ui.Div_start(0, 1, 1, 1)
		base.drawNetwork(ui)
		ui.Div_end()
	}
	ui.Div_end()
}

func (base *SABase) drawMenu(ui *Ui) {

	ui.Div_colMax(0, 8)
	ui.Div_row(1, 0.2)
	ui.Div_row(3, 0.2)
	ui.Div_row(5, 0.2)
	ui.Div_row(7, 0.2)
	ui.Div_row(9, 0.2)

	iconMargin := 0.22
	ini := &ui.win.io.ini
	y := 0
	//save
	if ui.Comp_buttonMenuIcon(0, y, 1, 1, base.trns.SAVE, "file:apps/base/resources/save.png", iconMargin, "", true, false) > 0 {
		//...
		ui.Dialog_close()
	}
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	//settings
	if ui.Comp_buttonMenuIcon(0, y, 1, 1, base.trns.SETTINGS, "file:apps/base/resources/settings.png", iconMargin, "", true, false) > 0 {
		ui.Dialog_close()
		ui.Dialog_open("settings", 0)
	}
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	//zoom
	ui.Div_start(0, y, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(2, 2)

		ui.Comp_textIcon(0, 0, 1, 1, base.trns.ZOOM, "file:apps/base/resources/zoom.png", iconMargin)

		if ui.Comp_buttonOutlined(1, 0, 1, 1, "+", "", true, false) > 0 {
			ini.Dpi += 3
		}

		dpiV := int(float64(ini.Dpi) / float64(ini.Dpi_default) * 100)
		ui.Comp_text(2, 0, 1, 1, strconv.Itoa(dpiV)+"%", 1)

		if ui.Comp_buttonOutlined(3, 0, 1, 1, "-", "", true, false) > 0 {
			ini.Dpi -= 3
		}
	}
	ui.Div_end()
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	//window/fullscreen switch
	{
		ff := base.trns.WINDOW_MODE
		icon := "file:apps/base/resources/window_mode.png"
		if !ini.Fullscreen {
			ff = base.trns.FULLSCREEN_MODE
			icon = "file:apps/base/resources/fullscreen_mode.png"
		}
		if ui.Comp_buttonMenuIcon(0, y, 1, 1, ff, icon, iconMargin, "", true, false) > 0 {
			ini.Fullscreen = !ini.Fullscreen
		}
	}
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	if ui.Comp_buttonMenuIcon(0, y, 1, 1, base.trns.ABOUT, "file:apps/base/resources/about.png", iconMargin, "", true, false) > 0 {
		ui.Dialog_close()
		ui.Dialog_open("about", 0)
	}
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	if ui.Comp_buttonMenuIcon(0, y, 1, 1, base.trns.QUIT, "file:apps/base/resources/quit.png", iconMargin, "", true, false) > 0 {
		base.exit = true
		ui.Dialog_close()
	}
	y++
}

// https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes
const g_langs = "English|Chinese(中文)|Hindi(हिंदी)|Spanish(Español)|Russian(Руштина)|Czech(Česky)"

var g_lang_codes = []string{"en", "zh", "hi", "es", "ru", "cs"}

func _SABase_FindLangCode(lng string) int {
	for ii, cd := range g_lang_codes {
		if cd == lng {
			return ii
		}
	}
	return 0
}

func (base *SABase) drawMenuDialogs(ui *Ui) {
	if ui.Dialog_start("about") {
		ui.Div_colMax(0, 15)
		ui.Div_row(1, 3)

		ui.Comp_text(0, 0, 1, 1, base.trns.ABOUT, 1)

		ui.Comp_image(0, 1, 1, 1, "file:apps/base/resources/logo.png", OsCd{A: 255}, 0, 1, 1, false)

		ui.Comp_text(0, 2, 1, 1, "v0.4", 1)

		ui.Comp_buttonText(0, 3, 1, 1, "www.skyalt.com", "https://www.skyalt.com", "", true, false)
		ui.Comp_buttonText(0, 4, 1, 1, "github.com/skyaltlabs/skyalt/", "https://github.com/skyaltlabs/skyalt/", "", true, false)

		ui.Comp_text(0, 5, 1, 1, base.trns.COPYRIGHT, 1)
		ui.Comp_text(0, 6, 1, 1, base.trns.WARRANTY, 1)
		ui.Dialog_end()
	}

	if ui.Dialog_start("settings") {
		ui.Div_colMax(1, 12)
		ui.Div_colMax(2, 1)

		y := 0

		ui.Comp_text(1, 0, 1, 1, base.trns.SETTINGS, 1)
		y++

		ini := &ui.win.io.ini

		//languages
		{
			ui.Comp_text(1, y, 1, 1, base.trns.LANGUAGES, 0)
			y++
			for i, lng := range ini.Languages {

				lang_id := _SABase_FindLangCode(lng)
				ui.Div_start(1, y, 1, 1)
				{
					ui.Div_colMax(2, 100)

					ui.Comp_text(0, 0, 1, 1, strconv.Itoa(i+1)+".", 1)

					ui.Div_start(1, 0, 1, 1)
					{
						ui.Div_drag("lang", i)
						src, pos, done := ui.Div_drop("lang", true, false, false)
						if done {
							Div_DropMoveElement(&ini.Languages, &ini.Languages, src, i, pos)
							base.reloadTranslations(ui)
						}
						ui.Comp_image(0, 0, 1, 1, "file:apps/base/resources/reorder.png", OsCd{A: 255}, 0.15, 1, 1, false)
					}
					ui.Div_end()

					if ui.Comp_combo(2, 0, 1, 1, &lang_id, g_langs, "", true, true) {
						ini.Languages[i] = g_lang_codes[lang_id]
						base.reloadTranslations(ui)
					}

					if ui.Comp_buttonLight(3, 0, 1, 1, "X", "", len(ini.Languages) > 1 || i > 0) > 0 {
						ini.Languages = append(ini.Languages[:i], ini.Languages[i+1:]...)
						base.reloadTranslations(ui)
					}
				}
				ui.Div_end()
				i++
				y++
			}

			ui.Div_start(1, y, 1, 1)
			if ui.Comp_buttonLight(0, 0, 1, 1, "+", "", true) > 0 {
				ini.Languages = append(ini.Languages, "en")
				base.reloadTranslations(ui)
			}
			y++
			ui.Div_end()

			y++ //space
		}

		ui.Comp_combo_desc(base.trns.DATE_FORMAT, 0, 4, 1, y, 1, 2, &ini.DateFormat, base.trns.DATE_FORMAT_EU+"|"+base.trns.DATE_FORMAT_US+"|"+base.trns.DATE_FORMAT_ISO+"|"+base.trns.DATE_FORMAT_TEXT, "", true, true)
		y += 2

		ui.Comp_combo_desc(base.trns.THEME, 0, 4, 1, y, 1, 2, &ini.Theme, base.trns.THEME_OCEAN+"|"+base.trns.THEME_RED+"|"+base.trns.THEME_BLUE+"|"+base.trns.THEME_GREEN+"|"+base.trns.THEME_GREY, "", true, true)
		y += 2

		ui.Comp_editbox_desc(base.trns.DPI, 0, 4, 1, y, 1, 2, &ini.Dpi, 0, "", "", false, false)
		y += 2

		ui.Comp_switch(1, y, 1, 1, &ini.Stats, base.trns.SHOW_STATS, "", true)
		y++

		ui.Dialog_end()
	}

}

func (base *SABase) drawIcons(ui *Ui, icon_rad float64) {

	ui.Paint_rect(0, 0, 1, 1, 0, ui.win.io.GetPalette().GetGrey(0.7), 0)

	ui.DivInfo_set(SA_DIV_SET_scrollVnarrow, 1, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

	ui.Div_colMax(0, 100)

	for i := 0; i < 2+len(base.settings.Apps)+1; i++ {
		ui.Div_row(i, icon_rad)
	}
	ui.Div_row(0, 1)   //spacer
	ui.Div_row(1, 0.1) //spacer

	y := 0
	if ui.Comp_buttonIcon(0, y, 1, 1, "file:apps/base/resources/logo_small.png", "", true) > 0 {
		ui.Dialog_open("menu", 1)
	}
	if ui.Dialog_start("menu") {
		base.drawMenu(ui)
		ui.Dialog_end()
	}
	y++

	base.drawMenuDialogs(ui)

	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	//show apps
	for i, app := range base.settings.Apps {

		nm := app
		if len(nm) > 3 {
			nm = nm[:3]
		}

		//drag & drop(under button)
		ui.Div_start(0, y, 1, 1)
		{
			dst := i
			ui.Div_drag("app", dst)
			src, pos, done := ui.Div_drop("app", true, false, false)
			if done {
				Div_DropMoveElement(&base.settings.Apps, &base.settings.Apps, src, dst, pos)
			}
		}
		ui.Div_end()

		click := ui.Comp_buttonText(0, y, 1, 1, nm, "", "", true, base.settings.Selected == i)
		if click == 1 {
			base.settings.Selected = i
		}
		appUid := fmt.Sprintf("app_context_%d", i)
		if click == 2 {
			ui.Dialog_open(appUid, 1)
		}
		if ui.Dialog_start(appUid) {
			ui.Div_colMax(0, 5)
			ui.Div_row(1, 0.1)

			if ui.Comp_buttonMenu(0, 0, 1, 1, base.trns.RENAME, "", true, false) > 0 {
				ui.Dialog_open(appUid+"_rename", 1)
				//dialog(new_name + button) ...
			}

			ui.Div_SpacerRow(0, 1, 1, 1)

			if ui.Comp_buttonMenu(0, 2, 1, 1, base.trns.REMOVE, "", true, false) > 0 {
				ui.Dialog_open(appUid+"_delete", 1)
				//dialog confirm ...
			}

			ui.Dialog_end()
		}

		y++
	}

	//+
	if ui.Comp_buttonText(0, y, 1, 1, "+", "", base.trns.CREATE_APP, true, false) > 0 {
		ui.Dialog_open("new_app", 1)
	}
	if ui.Dialog_start("new_app") {

		ui.Div_colMax(0, 10)

		ui.Comp_editbox(0, 0, 1, 1, &base.settings.NewAppName, 0, "", "", false, false)

		if ui.Comp_button(0, 1, 1, 1, base.trns.CREATE_APP, "", true) > 0 {
			OsFolderCreate("apps/" + base.settings.NewAppName)
			base.settings.Refresh()
			ui.Dialog_close()
		}

		ui.Dialog_end()
	}
}

func (base *SABase) drawApp(ui *Ui) {
	//...
}

func (base *SABase) drawParameters(ui *Ui) {
	//...
}

func (base *SABase) drawNetwork(ui *Ui) {

	pl := ui.buff.win.io.GetPalette()

	//fade "press tab" centered in background
	lv := ui.GetCall()
	ui._compDrawText(lv.call.canvas, "press tab", "", pl.GetGrey(0.7), false, false, 1, 1, true)

	//...
}

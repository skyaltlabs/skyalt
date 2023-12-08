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
	"strconv"
)

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
		base.Save()
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

		ui.Comp_editbox_desc(base.trns.DPI, 0, 4, 1, y, 1, 2, &ini.Dpi, 0, "", "", false, false, true)
		y += 2

		ui.Comp_editbox_desc(base.trns.THREADS, 0, 4, 1, y, 1, 2, &ini.Threads, 0, "", "", false, false, true)
		y += 2

		ui.Comp_switch(1, y, 1, 1, &ini.Stats, false, base.trns.SHOW_STATS, "", true)
		y++

		ui.Dialog_end()
	}

}

func (base *SABase) drawIcons(app *SAApp, ui *Ui, icon_rad float64) {

	ui.Paint_rect(0, 0, 1, 1, 0, ui.win.io.GetPalette().GetGrey(0.7), 0)

	ui.DivInfo_set(SA_DIV_SET_scrollVnarrow, 1, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

	ui.Div_colMax(0, 100)

	//ui.Div_row(0, 1)
	ui.Div_row(1, 0.1) //spacer
	ui.Div_rowMax(2, 100)
	ui.Div_row(4, 0.1) //spacer

	//Menu
	{
		if ui.Comp_buttonIcon(0, 0, 1, 1, "file:apps/base/resources/logo_small.png", 0, "", true) > 0 {
			ui.Dialog_open("menu", 1)
		}
		if ui.Dialog_start("menu") {
			base.drawMenu(ui)
			ui.Dialog_end()
		}

		base.drawMenuDialogs(ui)
	}

	ui.Div_SpacerRow(0, 1, 1, 1)

	//Apps
	{
		ui.Div_start(0, 2, 1, 1)

		ui.Div_colMax(0, 100)
		for i := 0; i < len(base.Apps); i++ {
			ui.Div_row(i, icon_rad)
		}

		y := 0
		for i, app := range base.Apps {

			nm := app.Name
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
					Div_DropMoveElement(&base.Apps, &base.Apps, src, dst, pos)
				}
			}
			ui.Div_end()

			click := ui.Comp_buttonText(0, y, 1, 1, nm, "", "", true, base.Selected == i)
			if click == 1 {
				base.Selected = i
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
		ui.Div_end()
	}

	//+
	{
		if ui.Comp_buttonText(0, 3, 1, 1, "+", "", base.trns.CREATE_APP, true, false) > 0 {
			ui.Dialog_open("new_app", 1)
		}
		if ui.Dialog_start("new_app") {
			ui.Div_colMax(0, 10)

			ui.Comp_editbox(0, 0, 1, 1, &base.NewAppName, 0, "", "", false, false, true)

			if ui.Comp_button(0, 1, 1, 1, base.trns.CREATE_APP, "", true) > 0 {
				OsFolderCreate("apps/" + base.NewAppName)
				base.Refresh()
				ui.Dialog_close()
			}

			ui.Dialog_end()
		}
	}

	ui.Div_SpacerRow(0, 4, 1, 1)

	//IDE
	{
		if ui.Comp_buttonText(0, 5, 1, 1, "IDE", "", "", true, app.IDE) > 0 {
			app.IDE = !app.IDE
		}
		//shortcut? ...
	}

}
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
	"strings"
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
	if ui.Comp_buttonMenuIcon(0, y, 1, 1, ui.trns.SAVE, InitWinMedia_url("file:apps/base/resources/save.png"), iconMargin, false, Comp_buttonProp()) > 0 {
		base.Save()
		ui.Dialog_close()
	}
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	//settings
	if ui.Comp_buttonMenuIcon(0, y, 1, 1, ui.trns.SETTINGS, InitWinMedia_url("file:apps/base/resources/settings.png"), iconMargin, false, Comp_buttonProp()) > 0 {
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

		ui.Comp_textIcon(0, 0, 1, 1, ui.trns.ZOOM, InitWinMedia_url("file:apps/base/resources/zoom.png"), iconMargin)

		if ui.Comp_buttonOutlined(1, 0, 1, 1, "+", Comp_buttonProp()) > 0 {
			ini.Dpi += 3
		}

		dpiV := int(float64(ini.Dpi) / float64(ini.Dpi_default) * 100)
		ui.Comp_text(2, 0, 1, 1, strconv.Itoa(dpiV)+"%", 1)

		if ui.Comp_buttonOutlined(3, 0, 1, 1, "-", Comp_buttonProp()) > 0 {
			ini.Dpi -= 3
		}
	}
	ui.Div_end()
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	//window/fullscreen switch
	{
		ff := ui.trns.WINDOW_MODE
		icon := "file:apps/base/resources/window_mode.png"
		if !ini.Fullscreen {
			ff = ui.trns.FULLSCREEN_MODE
			icon = "file:apps/base/resources/fullscreen_mode.png"
		}
		if ui.Comp_buttonMenuIcon(0, y, 1, 1, ff, InitWinMedia_url(icon), iconMargin, false, Comp_buttonProp()) > 0 {
			ini.Fullscreen = !ini.Fullscreen
		}
	}
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	if ui.Comp_buttonMenuIcon(0, y, 1, 1, ui.trns.ABOUT, InitWinMedia_url("file:apps/base/resources/about.png"), iconMargin, false, Comp_buttonProp()) > 0 {
		ui.Dialog_close()
		ui.Dialog_open("about", 0)
	}
	y++
	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	if ui.Comp_buttonMenuIcon(0, y, 1, 1, ui.trns.QUIT, InitWinMedia_url("file:apps/base/resources/quit.png"), iconMargin, false, Comp_buttonProp()) > 0 {
		base.exit = true
		ui.Dialog_close()
	}
	y++
}

// https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes
var g_lang_names = []string{"English", "Chinese(中文)", "Hindi(हिंदी)", "Spanish(Español)", "Russian(Руштина)", "Czech(Česky)"}
var g_lang_codes = []string{"en", "zh", "hi", "es", "ru", "cs"}

func (base *SABase) drawMenuDialogs(ui *Ui) {
	if ui.Dialog_start("about") {
		ui.Div_colMax(0, 15)
		ui.Div_row(1, 3)

		ui.Comp_text(0, 0, 1, 1, ui.trns.ABOUT, 1)

		ui.Comp_image(0, 1, 1, 1, InitWinMedia_url("file:apps/base/resources/logo.png"), OsCd{A: 255}, 0, OsV2{1, 1}, false)

		ui.Comp_text(0, 2, 1, 1, "v0.1", 1)

		ui.Comp_buttonText(0, 3, 1, 1, "www.skyalt.com", Comp_buttonProp().Url("https://www.skyalt.com"))
		ui.Comp_buttonText(0, 4, 1, 1, "github.com/skyaltlabs/skyalt/", Comp_buttonProp().Url("https://github.com/skyaltlabs/skyalt/"))

		ui.Comp_text(0, 5, 1, 1, ui.trns.COPYRIGHT, 1)
		ui.Comp_text(0, 6, 1, 1, ui.trns.WARRANTY, 1)
		ui.Dialog_end()
	}

	if ui.Dialog_start("settings") {
		ui.Div_colMax(1, 15)
		ui.Div_colMax(2, 1)

		y := 0

		ui.Comp_text(1, 0, 1, 1, ui.trns.SETTINGS, 1)
		y++

		ini := &ui.win.io.ini

		//languages
		{
			ui.Comp_text(1, y, 1, 1, ui.trns.LANGUAGES, 0)
			y++
			for i := range ini.Languages {
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
							ui.reloadTranslations()
						}
						ui.Comp_image(0, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/reorder.png"), OsCd{A: 255}, 0.15, OsV2{1, 1}, false)
					}
					ui.Div_end()

					if ui.Comp_combo(2, 0, 1, 1, &ini.Languages[i], g_lang_names, g_lang_codes, "", true, true) {
						ui.reloadTranslations()
					}

					if ui.Comp_buttonLight(3, 0, 1, 1, "X", Comp_buttonProp().Enable(len(ini.Languages) > 1 || i > 0)) > 0 {
						ini.Languages = append(ini.Languages[:i], ini.Languages[i+1:]...)
						ui.reloadTranslations()
					}
				}
				ui.Div_end()
				i++
				y++
			}

			ui.Div_start(1, y, 1, 1)
			if ui.Comp_buttonLight(0, 0, 1, 1, "+", Comp_buttonProp()) > 0 {
				ini.Languages = append(ini.Languages, "en")
				ui.reloadTranslations()
			}
			y++
			ui.Div_end()

			y++ //space
		}

		//format
		{
			format_names := []string{ui.trns.DATE_FORMAT_EU, ui.trns.DATE_FORMAT_US, ui.trns.DATE_FORMAT_ISO, ui.trns.DATE_FORMAT_TEXT}
			format_values := []string{"eu", "us", "iso", "text"} //"2base"
			ui.Comp_combo_desc(ui.trns.DATE_FORMAT, 0, 4, 1, y, 1, 2, &ini.DateFormat, format_names, format_values, "", true, true)
			y += 2
		}

		//theme
		{
			format_names := []string{ui.trns.LIGHT, ui.trns.DARK, ui.trns.CUSTOM}
			format_values := []string{"light", "dark", "custom"}
			ui.Comp_combo_desc(ui.trns.THEME, 0, 4, 1, y, 1, 2, &ini.Theme, format_names, format_values, "", true, true)
			y++

			//custom palette
			if ui.win.io.ini.Theme == "custom" {
				pl := &ui.win.io.ini.CustomPalette
				ui.Div_start(1, y, 1, 2)
				{
					ui.Div_col(0, 4)
					ui.Div_colMax(1, 100)
					ui.Div_colMax(2, 100)
					ui.Div_colMax(3, 100)
					ui.Div_colMax(4, 100)
					ui.Div_colMax(5, 100)

					if ui.Comp_buttonLight(0, 0, 1, 1, "Reset", Comp_buttonProp()) > 0 {
						*pl = ui.win.io.palettes[0] //light
					}

					ui.Comp_text(1, 0, 1, 1, "Primary", 1)
					ui.comp_colorPicker(1, 1, 1, 1, &pl.P, "p", "", true)

					ui.Comp_text(2, 0, 1, 1, "Secondary", 1)
					ui.comp_colorPicker(2, 1, 1, 1, &pl.S, "s", "", true)

					ui.Comp_text(3, 0, 1, 1, "Tertiary", 1)
					ui.comp_colorPicker(3, 1, 1, 1, &pl.T, "t", "", true)

					ui.Comp_text(4, 0, 1, 1, "Background", 1)
					ui.comp_colorPicker(4, 1, 1, 1, &pl.B, "b", "", true)

					ui.Comp_text(5, 0, 1, 1, "Error", 1)
					ui.comp_colorPicker(5, 1, 1, 1, &pl.E, "e", "", true)

				}
				ui.Div_end()
				y += 2

				y++ //space
			}
		}

		ui.Div_start(1, y, 1, 1)
		{
			ui.Div_colMax(0, 4) //empty space
			ui.Div_colMax(1, 100)
			ui.Div_colMax(2, 2.1)
			ui.Div_colMax(3, 2)
			ui.Div_colMax(4, 1)
			ui.Comp_checkbox(1, 0, 1, 1, &ini.UseDarkTheme, false, ui.trns.USE_DARK_THEME, "", true)
			ui.Comp_editbox_desc(ui.trns.FROM, 2, 1.1, 2, 0, 1, 1, &ini.UseDarkThemeStart, Comp_editboxProp().Precision(0).Enable(ini.UseDarkTheme))
			ui.Comp_editbox_desc(ui.trns.TO, 2, 1, 3, 0, 1, 1, &ini.UseDarkThemeEnd, Comp_editboxProp().Precision(0).Enable(ini.UseDarkTheme))
			ui.Comp_text(4, 0, 1, 1, ui.trns.HOUR, 0)
		}
		ui.Div_end()
		y += 2 //space

		ui.Comp_editbox_desc(ui.trns.DPI, 0, 4, 1, y, 1, 2, &ini.Dpi, Comp_editboxProp().Precision(0))
		y += 2

		//ui.Comp_editbox_desc(ui.trns.THREADS, 0, 4, 1, y, 1, 2, &ini.Threads, Comp_editboxProp().Precision(0))
		//y += 2

		ui.Comp_switch(1, y, 1, 1, &ini.Stats, false, ui.trns.SHOW_STATS, "", true)
		y++

		ui.Comp_switch(1, y, 1, 1, &ini.Grid, false, ui.trns.SHOW_GRID, "", true)
		y++

		ui.Comp_switch(1, y, 1, 1, &ini.Offline, true, ui.trns.ONLINE, "", true) //true = reverseValue
		y++

		ui.Comp_switch(1, y, 1, 1, &ini.MicOff, true, ui.trns.MICROPHONE, "", true) //true = reverseValue
		y++

		y++ //space

		//OpenAI key
		{
			key := ""
			//encode
			if len(ini.OpenAI_key) > 6 {
				key = ini.OpenAI_key[:3] //first 3
				for i := 3; i < len(ini.OpenAI_key)-3; i++ {
					key += "*"
				}
				key += ini.OpenAI_key[len(ini.OpenAI_key)-3:] //last 3
			}
			_, _, _, fnshd, _ := ui.Comp_editbox_desc("OpenAI API key", 0, 4, 1, y, 1, 2, &key, Comp_editboxProp().Formating(false))
			if fnshd {
				if strings.IndexByte(key, '*') < 0 {
					//decode
					ini.OpenAI_key = key
				}
			}
			y++
		}

		y++ //space

		//delete Temp
		if ui.Comp_buttonLight(1, y, 1, 1, "Delete Cache", Comp_buttonProp().SetError(true).Confirmation("Are you sure?", "confirm_delete_cache")) > 0 {
			OsFolderRemove("temp")
		}

		ui.Dialog_end()
	}

}

func (base *SABase) drawLauncher(app *SAApp, icon_rad float64) {

	ui := app.base.ui

	ui.Paint_rect(0, 0, 1, 1, 0, ui.win.io.GetPalette().GetGrey(0.7), 0)

	ui.DivInfo_set(SA_DIV_SET_scrollVnarrow, 1, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

	ui.Div_colMax(0, 100)

	//ui.Div_row(0, 1)
	ui.Div_row(1, 0.1) //spacer
	ui.Div_rowMax(2, 100)

	//Menu
	{
		if ui.Comp_buttonIcon(0, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/logo_small.png"), 0.1, "", Comp_buttonProp().Cd(CdPalette_B)) > 0 {
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

		ui.DivInfo_set(SA_DIV_SET_scrollVnarrow, 1, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

		ui.Div_colMax(0, 100)
		for i := 0; i < len(base.Apps)+1; i++ { //+1 = "+"
			ui.Div_row(i, icon_rad)
		}

		if base.Selected >= 0 {
			ui.Div_row(base.Selected, icon_rad+1)
		}

		y := 0
		for i, app := range base.Apps {

			var click int

			//drag & drop(under button)
			ui.Div_start(0, y, 1, 1)
			{
				if base.Selected == i {
					ui.Div_colMax(0, 100)
					ui.Div_rowMax(0, 100)
					ui.Div_row(1, 0.7) //IDE
					ui.Div_row(2, 0.3) //bottom space
				} else {
					ui.Div_colMax(0, 100)
					ui.Div_rowMax(0, 100)
				}

				if base.Selected == i {
					pl := ui.win.io.GetPalette()
					cd := pl.P
					cd.A = 100
					ui.Paint_rect(0, 0, 1, 1, 0.06, cd, 0)
				}

				dst := i
				ui.Div_drag("app", dst)
				src, pos, done := ui.Div_drop("app", true, false, false)
				if done {
					Div_DropMoveElement(&base.Apps, &base.Apps, src, dst, pos)
				}

				if app.iconPath != "" {
					click = ui.Comp_buttonIcon(0, 0, 1, 1, InitWinMedia_url(app.iconPath), 0.4, app.Name, Comp_buttonProp().Cd(CdPalette_B))
				} else {
					nm := app.Name
					if len(nm) > 3 {
						nm = nm[:3]
					}
					click = ui.Comp_buttonText(0, 0, 1, 1, nm, Comp_buttonProp())
				}
				if base.Selected == i {
					//IDE
					if ui.Comp_button(0, 1, 1, 1, OsTrnString(app.IDE, "**IDE**", "IDE"), Comp_buttonProp().Cd(CdPalette_B).DrawBack(false)) > 0 {
						app.IDE = !app.IDE
					}
				}

				_, progress := base.jobs.FindAppProgress(app)
				if progress >= 0 {
					ui.Div_start(0, 0, 1, 1)
					pl := ui.win.io.GetPalette()
					ui.Paint_rect(0, 0.47, 1, 0.06, 0.2, pl.OnP, 0)

					ui.Paint_rect(0, 0.47, progress, 0.06, 0.2, pl.P, 0)
					ui.Div_end()
				}

			}
			ui.Div_end()

			if click == 1 {
				if base.Selected == i {
					app.IDE = !app.IDE //already selected => switch IDE
				}

				base.Selected = i
			}
			appUid := fmt.Sprintf("app_context_%d", i)
			if click == 2 {
				ui.Dialog_open(appUid, 1)
			}

			renameDialog := appUid + "_rename"
			if ui.Dialog_start(appUid) {
				ui.Div_colMax(0, 5)
				ui.Div_row(1, 0.1)

				if ui.Comp_buttonMenu(0, 0, 1, 1, ui.trns.RENAME, false, Comp_buttonProp()) > 0 {
					ui.Dialog_close()
					base.NewAppName = app.Name
					ui.Dialog_open(renameDialog, 1)
				}

				ui.Div_SpacerRow(0, 1, 1, 1)

				if ui.Comp_buttonMenu(0, 2, 1, 1, ui.trns.REMOVE, false, Comp_buttonProp().SetError(true).Confirmation(fmt.Sprintf("Do you really wanna delete '%s' app?", app.Name), "confirm_delete_app_"+app.Name)) > 0 {
					if OsFolderRemove(app.GetFolderPath()) == nil {
						app.Destroy()
						base.Apps = append(base.Apps[:i], base.Apps[i+1:]...) //remove
					}
					ui.Dialog_close()
				}

				ui.Dialog_end()
			}
			y++

			if ui.Dialog_start(renameDialog) {
				ui.Div_colMax(0, 5)

				ui.Comp_editbox(0, 0, 1, 1, &base.NewAppName, Comp_editboxProp())
				if ui.Comp_button(0, 1, 1, 1, ui.trns.RENAME, Comp_buttonProp().Enable(base.NewAppName != "")) > 0 {
					if OsFileRename(app.GetFolderPath(), SAApp_GetNewFolderPath(base.NewAppName)) == nil {
						app.Name = base.NewAppName
						ui.Dialog_close()
						base.NewAppName = "" //reset
					}
				}
				ui.Dialog_end()
			}
		}

		//+
		{
			if ui.Comp_buttonLight(0, y, 1, 1, "+", Comp_buttonProp().Tooltip(ui.trns.CREATE_APP)) > 0 {
				ui.Dialog_open("new_app", 1)
			}
			if ui.Dialog_start("new_app") {
				ui.Div_colMax(0, 10)

				ui.Comp_editbox(0, 0, 1, 1, &base.NewAppName, Comp_editboxProp())

				if ui.Comp_button(0, 1, 1, 1, ui.trns.CREATE_APP, Comp_buttonProp()) > 0 {
					OsFolderCreate("apps/" + base.NewAppName)
					base.Refresh()
					ui.Dialog_close()
					base.NewAppName = "" //reset
				}

				ui.Dialog_end()
			}
		}

		ui.Div_end()
	}
}

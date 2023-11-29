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

type SAApp struct {
	Name     string
	Cam      OsV2f
	Cam_zoom float32

	nodes              *Nodes
	cam_move           bool
	node_move          bool
	node_select        bool
	cam_start          OsV2f
	touch_start        OsV2
	node_move_selected *Node

	node_connect     *Node
	node_connect_in  int
	node_connect_out int

	//selection_start OsV2

	saveIt bool

	tab_touchPos OsV2
}

func (app *SAApp) GetPath() string {
	return "apps/" + app.Name + "/app.json"
}

type SABaseSettings struct {
	Apps     []SAApp
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
		if a.Name == name {
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
			sts.Apps = append(sts.Apps, SAApp{Name: f.Name, Cam_zoom: 1})
		}
	}

	//remove(deleted from disk)
	for i := len(sts.Apps) - 1; i >= 0; i-- {
		if files.FindInSubs(sts.Apps[i].Name, true) < 0 {
			sts.Apps = append(sts.Apps[:i], sts.Apps[i+1:]...)
		}
	}

	//remove duplicity
	for i := len(sts.Apps) - 1; i >= 0; i-- {

		if sts.findApp(sts.Apps[i].Name) != i {
			sts.Apps = append(sts.Apps[:i], sts.Apps[i+1:]...)
		}
	}
}

type SABase struct {
	ui *Ui

	settings SABaseSettings

	exit bool

	trns SABase_translations
}

func NewSABase(ui *Ui) (*SABase, error) {
	var base SABase

	base.ui = ui

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

func (base *SABase) Save() {
	//apps
	for _, a := range base.settings.Apps {
		if a.nodes != nil {
			if a.saveIt {
				a.nodes.Save(a.GetPath())
			}
		}
	}

	//layouts
	base.ui.Save(SABase_GetPathLayout())

	//Base
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

}

func (base *SABase) Destroy() {
	base.Save()

	//close & save apps
	for _, a := range base.settings.Apps {
		if a.nodes != nil {
			a.nodes.Destroy()
		}
	}
}

func (base *SABase) getPath() string {
	return "apps/base/app.json"
}

func SABase_GetPathLayout() string {
	dev, _ := os.Hostname()
	return "apps/layout_" + dev + ".json"
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

	app := &base.settings.Apps[base.settings.Selected]
	if app.nodes == nil {
		app.nodes, _ = NewNodes("apps/" + app.Name + "/app.json") //err ...
	}
	app.saveIt = true

	if base.settings.HasApp() {
		ui.Div_startName(1, 0, 1, 1, base.settings.Apps[base.settings.Selected].Name)
		base.drawApp(app, ui)
		ui.Div_end()
	}

	ui.Div_start(2, 0, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_rowResize(0, "parameters", 10)
		ui.Div_rowMax(1, 100)

		ui.Div_start(0, 0, 1, 1)
		base.drawParameters(app, ui)
		ui.Div_end()

		ui.Div_start(0, 1, 1, 1)
		base.drawNetwork(app, ui)
		ui.Div_end()

	}
	ui.Div_end()
}

func (base *SABase) drawParameters(app *SAApp, ui *Ui) {
	var node *Node
	for _, n := range app.nodes.nodes {
		if n.Selected {
			node = n
			break
		}
	}

	if node == nil {
		pl := ui.buff.win.io.GetPalette()
		lv := ui.GetCall()
		ui._compDrawText(lv.call.canvas, "No node selected", "", pl.GetGrey(0.7), SKYALT_FONT_HEIGHT, false, false, 1, 1, true)

		return
	}

	ui.Div_colMax(0, 100)
	ui.Div_row(1, 0.1)
	ui.Div_rowMax(2, 100)

	ui.Div_start(0, 0, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Comp_editbox_desc("Name", 0, 2, 0, 0, 1, 1, &node.Name, 0, "", "", false, false) //rename
		ui.Comp_switch(1, 0, 1, 1, &node.Bypass, true, "", "Bypass", true)
	}
	ui.Div_end()

	ui.Div_SpacerRow(0, 1, 1, 1)

	fn := app.nodes.FindFn(node.FnName)
	if fn != nil && fn.parameters != nil {
		ui.Div_start(0, 2, 1, 1)
		fn.parameters(node, ui)
		ui.Div_end()
	}
}

func (base *SABase) drawApp(app *SAApp, ui *Ui) {
	for _, n := range app.nodes.nodes {
		if n.Bypass {
			continue
		}

		fn := app.nodes.FindFn(n.FnName)
		if fn != nil && fn.parameters != nil {
			if fn.render != nil {
				ui.Div_start(n.GridCoord.Start.X, n.GridCoord.Start.Y, n.GridCoord.Size.X, n.GridCoord.Size.Y)

				fn.render(n, ui)

				if n.Selected {
					ui.Paint_rect(0, 0, 1, 1, 0, base.getYellow(), 0.03)
				}

				//alt+click => select node and zoom_in network ...

				ui.Div_end()
			}

		}
	}
}

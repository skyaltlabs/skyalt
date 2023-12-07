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

func (base *SABase) HasApp() bool {
	if base.Selected >= len(base.Apps) {
		base.Selected = len(base.Apps) - 1
	}
	return base.Selected >= 0
}

func (base *SABase) findApp(name string) int {
	for i, a := range base.Apps {
		if a.Name == name {
			return i
		}
	}
	return -1
}

func (base *SABase) Refresh() {
	files := OsFileListBuild("apps", "", true)

	//add(new on disk)
	for _, f := range files.Subs {
		if !f.IsDir || f.Name == "base" {
			continue

		}
		if base.findApp(f.Name) < 0 {
			base.Apps = append(base.Apps, NewSAApp(f.Name, base))
		}
	}

	//remove(deleted from disk)
	for i := len(base.Apps) - 1; i >= 0; i-- {
		if files.FindInSubs(base.Apps[i].Name, true) < 0 {
			base.Apps = append(base.Apps[:i], base.Apps[i+1:]...)
		}
	}

	//remove duplicity
	for i := len(base.Apps) - 1; i >= 0; i-- {

		if base.findApp(base.Apps[i].Name) != i {
			base.Apps = append(base.Apps[:i], base.Apps[i+1:]...)
		}
	}
}

type SABase struct {
	ui *Ui

	Apps       []*SAApp
	Selected   int
	NewAppName string

	exit bool

	trns SATranslations
}

func NewSABase(ui *Ui) (*SABase, error) {
	base := &SABase{}

	base.ui = ui

	//open
	{
		OsFolderCreate("apps")
		OsFolderCreate("apps/base")

		js, err := os.ReadFile(base.getPath())
		if err == nil {
			err := json.Unmarshal([]byte(js), base)
			if err != nil {
				fmt.Printf("warnning: Unmarshal() failed: %v\n", err)
			}
		}
		for _, a := range base.Apps {
			a.parent = base
		}
	}

	//translations
	err := base.reloadTranslations(ui)
	if err != nil {
		return nil, fmt.Errorf("reloadTranslations() failed: %w", err)
	}

	base.Refresh()

	return base, nil
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
	for _, a := range base.Apps {
		if a.view != nil {
			if a.saveIt {
				a.view.Save(a.GetPath())
			}
		}
	}

	//layouts
	base.ui.Save(SABase_GetPathLayout())

	//Base
	{
		js, err := json.MarshalIndent(base, "", "")
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
	for _, a := range base.Apps {
		if a.view != nil {
			a.view.Destroy()
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
	base.HasApp() //fix range

	ui.renderStart(0, 0, 1, 1, true)

	//hard/soft render ...
	base.drawFrame(ui)

	ui.renderEnd(true)

	return !base.exit
}

func (base *SABase) drawFrame(ui *Ui) {

	app := base.Apps[base.Selected]
	if app.view == nil {
		app.view, _ = NewNodeView("apps/" + app.Name + "/app.json") //err ...
	}
	app.saveIt = true

	icon_rad := 1.7

	ui.Div_rowMax(0, 100)
	ui.Div_col(0, icon_rad)
	ui.Div_colMax(1, 100)
	if app.IDE {
		ui.Div_colResize(2, "settings", 10, false)
	}

	ui.Div_start(0, 0, 1, 1)
	base.drawIcons(app, ui, icon_rad)
	ui.Div_end()

	if base.HasApp() {

		ui.Div_startName(1, 0, 1, 1, base.Apps[base.Selected].Name)
		{
			if app.IDE {
				app.view.renderAppIDE(ui)
			} else {
				app.view.renderApp(ui, true)
			}
		}
		ui.Div_end()
	}

	if app.IDE {
		ui.Div_start(2, 0, 1, 1)

		ui.Div_colMax(0, 100)
		ui.Div_rowMax(1, 100)

		ui.Div_start(0, 0, 1, 1)
		app.drawGraphHeader(ui)
		ui.Div_end()

		ui.Div_start(0, 1, 1, 1)
		app.drawGraph(ui)
		ui.Div_end()

		ui.Div_end()
	}
}

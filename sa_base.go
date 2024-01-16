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

type SABase struct {
	ui *Ui

	Apps       []*SAApp
	Selected   int
	NewAppName string

	exit bool

	server *SANodeServer
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
			a.init(base)
		}
	}

	var err error
	base.server, err = NewSANodeServer("nodes", 4567)
	if err != nil {
		return nil, fmt.Errorf("NewNodeServer() failed: %w", err)
	}

	base.Refresh()

	return base, nil
}

func (base *SABase) Destroy() {
	base.Save()

	//close & save apps
	for _, a := range base.Apps {
		a.Destroy()
	}

	base.server.Destroy()
}

func (base *SABase) Save() {
	//apps
	for _, a := range base.Apps {
		if a.root != nil && a.saveIt {
			a.root.Save(a.GetJsonPath())
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

func (base *SABase) getPath() string {
	return "apps/base/app.json"
}

func SABase_GetPathLayout() string {
	dev, _ := os.Hostname()
	return "apps/layout_" + dev + ".json"
}

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
			continue //ignore

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

	//refresh server nodes list ...
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
	if app.root == nil {
		app.root, _ = NewSANodeRoot("apps/"+app.Name+"/app.json", app) //err ...
	}
	if app.act == nil {
		app.act = app.root
	}

	icon_rad := 1.7

	ui.Div_rowMax(0, 100)
	ui.Div_col(0, icon_rad)
	ui.Div_colMax(1, 100)
	if app.IDE {
		ui.Div_colResize(2, "parameters", 8, false)
	}

	ui.Div_start(0, 0, 1, 1)
	base.drawLauncher(app, ui, icon_rad)
	ui.Div_end()

	if base.HasApp() {
		ui.Div_startName(1, 0, 1, 1, base.Apps[base.Selected].Name)
		{
			app.Execute(ui.win.io.ini.Threads)

			if app.IDE {
				ui.Div_colMax(0, 100)
				ui.Div_rowMax(1, 100)

				ui.Div_start(0, 0, 1, 1)
				app.RenderHeader(ui)
				ui.Div_end()

				ui.Div_start(0, 1, 1, 1)
				app.renderIDE(ui)
				ui.Div_end()
			} else {
				app.RenderApp(false)
			}

		}
		ui.Div_end()
	}

	if app.IDE {
		ui.Div_start(2, 0, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_rowResize(0, "node", 5, false)
			ui.Div_rowMax(1, 100)

			sel := app.act.FindSelected()
			if sel != nil {
				ui.Div_start(0, 0, 1, 1)
				sel.RenderAttrs()
				ui.Div_end()
			}

			ui.Div_start(0, 1, 1, 1)
			app.graph.drawGraph(ui)
			ui.Div_end()
		}
		ui.Div_end()
	}

	app.History(ui)
}

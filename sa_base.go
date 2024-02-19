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

	mic                   *WinMic
	mic_actived_last_tick int64

	node_groups SAGroups

	copiedNodes []*SANode

	services *SAServices

	last_tick int64
}

func NewSABase(ui *Ui) (*SABase, error) {
	base := &SABase{}
	base.ui = ui

	base.node_groups = InitSAGroups()

	base.services = NewSAServices()

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

	base.Refresh()

	return base, nil
}

func (base *SABase) Destroy() {

	base.services.Destroy()

	base.Save()

	if base.mic != nil {
		base.mic.Destroy()
	}

	//close & save apps
	for _, a := range base.Apps {
		a.Destroy()
	}
}

func (base *SABase) Save() {
	//apps
	for _, a := range base.Apps {
		if a.root != nil {
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

func (base *SABase) tickMick() {

	if base.ui.win.io.ini.MicOff {
		if base.mic != nil {
			base.mic.Destroy()
			base.mic = nil
		}
		return
	}

	mic_active := false
	for _, app := range base.Apps {
		if len(app.mic_nodes) > 0 {
			mic_active = true
			break
		}
	}
	if mic_active {
		if base.mic == nil {
			//create
			var err error
			base.mic, err = NewWinMic()
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		base.mic_actived_last_tick = OsTicks()
	} else {
		if !OsIsTicksIn(base.mic_actived_last_tick, 2000) && base.mic != nil {
			if base.mic != nil {
				base.mic.Destroy()
				base.mic = nil
			}
		}
		return
	}

	//on/off
	base.mic.SetEnable(mic_active) //turn it ON

	if base.mic.IsPlaying() {
		mic_data := base.mic.Get()
		for _, app := range base.Apps {
			app.AddMic(mic_data)
		}
	}
}

func (base *SABase) Tick() {
	base.services.Tick()
	base.tickMick()

	base.HasApp() //fix range

	//check if some db changed
	if (!OsIsTicksIn(base.last_tick, 5000) && base.ui.win.disk.HasDbFileChanged()) ||
		base.ui.win.disk.HasDbBeenWritten() {

		for _, a := range base.Apps {
			a.SetExecute()
		}
		base.last_tick = OsTicks()
	}

	app := base.GetApp()
	app.Tick()
}

func (base *SABase) Render() bool {
	base.HasApp() //fix range
	app := base.GetApp()

	base.ui.renderStart(0, 0, 1, 1, true)

	ui := base.ui
	icon_rad := 1.5

	ui.Div_rowMax(0, 100)
	ui.Div_col(0, icon_rad)
	ui.Div_colMax(1, 100)
	if app.IDE {
		ui.Div_colResize(2, "parameters", 8, false)
	}

	ui.Div_start(0, 0, 1, 1)
	base.drawLauncher(app, icon_rad)
	ui.Div_end()

	if base.HasApp() {
		ui.Div_startName(1, 0, 1, 1, base.Apps[base.Selected].Name)
		{
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

			//attributes layout
			sel := app.root.FindSelected()
			if sel != nil {
				ui.Div_start(0, 0, 1, 1)
				sel.RenderAttrs()
				ui.Div_end()
			}

			//graph layout
			if app.Cam_z <= 0 {
				app.Cam_z = 1
			}

			ui.Div_start(0, 1, 1, 1)
			{
				ui.Div_colMax(0, 100)
				ui.Div_rowMax(0, 100)
				pn_x := 1
				if app.graph.showNodeList {
					ui.Div_col(1, 3) //min
					ui.Div_colResize(1, "node_list", 5, false)
					pn_x = 2
				}

				//graph
				ui.Div_start(0, 0, 1, 1)
				graphCanvas, keyAllow := app.graph.drawGraph(app.root, ui)
				ui.Div_end()

				//node list
				if app.graph.showNodeList {
					ui.Div_start(1, 0, 1, 1)
					app.graph.drawNodeList(graphCanvas, app.root, ui)
					ui.Div_end()
				}

				//panel
				ui.Div_start(pn_x, 0, 1, 1)
				app.graph.drawPanel(graphCanvas, keyAllow, ui)
				ui.Div_end()
			}
			ui.Div_end()

		}
		ui.Div_end()
	}

	app.History(ui)

	base.ui.renderEnd(true)

	return !base.exit
}

func (base *SABase) GetApp() *SAApp {
	app := base.Apps[base.Selected]
	if app.root == nil {
		app.root, _ = NewSANodeRoot(app.GetJsonPath(), app) //err ...

		//testing import/export
		/*if app.Name == "LLM_chat" {
			code := app.ExportCode()
			b := NewSAApp("dd", base)
			b.ImportCode(code)
		}*/
	}
	return app
}

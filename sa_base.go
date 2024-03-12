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
}

func NewSABase(ui *Ui) (*SABase, error) {
	base := &SABase{}
	base.ui = ui

	base.node_groups = InitSAGroups()

	base.services = NewSAServices(ui)

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

	app := base.GetApp()
	app.Tick()
}

func (base *SABase) Render() bool {
	base.HasApp() //fix range
	app := base.GetApp()

	base.ui.renderStart(0, 0, 1, 1)

	ui := base.ui
	icon_rad := 1.5

	ui.Div_rowMax(0, 100)
	ui.Div_col(0, icon_rad)
	ui.Div_colMax(1, 100)
	if app.IDE {
		ui.Div_colResize(2, "graph", 8, false)
	}

	ui.Div_start(0, 0, 1, 1)
	base.drawLauncher(app, icon_rad)
	ui.Div_end()

	//canvas
	if base.HasApp() {
		app.rebuildLists() //!!!

		ui.Div_startName(1, 0, 1, 1, base.Apps[base.Selected].Name)
		{
			if app.IDE {
				app.renderIDE()
			} else {
				app.RenderApp(false)
			}
		}
		ui.Div_end()
	}

	if app.IDE {
		//graph
		ui.Div_start(2, 0, 1, 1)
		{
			var graphCanvas OsV4

			ui.Div_col(0, 4)
			ui.Div_colMax(0, 100)
			ui.Div_rowMax(0, 100)

			sel_node := app.root.FindSelected()
			if app.graph.showNodeList || (sel_node != nil && sel_node.ShowCodeChat) {
				ui.Div_col(1, 3) //min
				ui.Div_colResize(1, "right_panel", 7, false)
			}

			ui.Div_start(0, 0, 1, 1)
			{
				ui.Div_colMax(0, 100)

				ui.Div_row(0, 3)
				ui.Div_rowResize(0, "attributes", 4, false)
				ui.Div_rowMax(1, 100)

				//attributes
				ui.Div_start(0, 0, 1, 1)
				{
					selNode := app.root.FindSelected()
					if selNode != nil {
						selNode.RenderAttrs()
					} else {
						ui.Div_colMax(0, 100)
						ui.Div_rowMax(0, 100)
						ui.Comp_text(0, 0, 1, 1, "No node selected", 1)
					}

				}
				ui.Div_end()

				//graph layout
				ui.Div_start(0, 1, 1, 1)
				{
					ui.Div_colMax(0, 100)
					ui.Div_rowMax(1, 100)

					if app.Cam_z <= 0 {
						app.Cam_z = 1
					}

					//graph
					ui.Div_start(0, 1, 1, 1)
					var keyAllow bool
					graphCanvas, keyAllow = app.graph.drawGraph(app.root)
					ui.Div_end()

					//panel
					ui.Div_start(0, 0, 1, 1)
					app.graph.drawPanel(graphCanvas, keyAllow)
					ui.Div_end()
				}
				ui.Div_end()
			}
			ui.Div_end()

			//node list
			if app.graph.showNodeList {
				ui.Div_start(1, 0, 1, 1)
				app.graph.drawNodeList(graphCanvas)
				ui.Div_end()
			} else if sel_node != nil && sel_node.ShowCodeChat {
				ui.Div_start(1, 0, 1, 1)
				UiCodeGo_AttrChat(sel_node)
				ui.Div_end()
			}

		}
		ui.Div_end()

		app.graph.History()
	}

	//app.flamingo.Tick()

	base.ui.renderEnd(true)

	return !base.exit
}

func (base *SABase) GetApp() *SAApp {
	app := base.Apps[base.Selected]
	if app.root == nil {
		app.root, _ = NewSANodeRoot(app.GetJsonPath(), app) //err ...
	}
	return app
}

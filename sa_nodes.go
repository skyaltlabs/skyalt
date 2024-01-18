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

import "strings"

type SANodeGroup struct {
	name  string
	icon  WinMedia
	nodes []string //add funcs ...........
}

func (gr *SANodeGroup) Find(name string) bool {
	for _, fn := range gr.nodes {
		if strings.EqualFold(fn, name) {
			return true
		}
	}
	return false
}

type SANodeGroups struct {
	groups []*SANodeGroup
	ui     *SANodeGroup
}

func InitSANodeGroups() SANodeGroups {
	var grs SANodeGroups

	path := "file:apps/base/resources/"

	grs.ui = &SANodeGroup{name: "UI", icon: InitWinMedia_url(path + "node_ui.png"), nodes: []string{"layout", "dialog", "button", "text", "checkbox", "switch", "editbox", "divider", "combo", "color_palette", "color", "calendar", "date", "map", "image", "file_drop"}}
	grs.groups = append(grs.groups, grs.ui)

	grs.groups = append(grs.groups, &SANodeGroup{name: "Variables", icon: InitWinMedia_url(path + "node_vars.png"), nodes: []string{"medium"}})
	grs.groups = append(grs.groups, &SANodeGroup{name: "File", icon: InitWinMedia_url(path + "node_file.png"), nodes: []string{"read_file", "write_file"}})
	grs.groups = append(grs.groups, &SANodeGroup{name: "Convert", icon: InitWinMedia_url(path + "node_convert.png"), nodes: []string{"csv_to_json", "gpx_to_json"}})
	grs.groups = append(grs.groups, &SANodeGroup{name: "SQLite", icon: InitWinMedia_url(path + "node_db.png"), nodes: []string{"sqlite_select", "sqlite_insert", "sqlite_update", "sqlite_delete", "sqlite_execute"}})

	//add custom ...
	//fns = append(fns, app.base.server.nodes...) //from /nodes dir

	return grs
}
func SANodeGroups_HasNodeSub(node string) bool {
	return strings.EqualFold(node, "layout") || strings.EqualFold(node, "dialog")
}

func (grs *SANodeGroups) IsUI(node string) bool {
	return grs.ui.Find(node)
}

func (grs *SANodeGroups) FindNode(node string) *SANodeGroup {

	for _, gr := range grs.groups {
		if gr.Find(node) {
			return gr
		}
	}

	return nil
}
func (grs *SANodeGroups) FindNodeIcon(node string) WinMedia {

	gr := grs.FindNode(node)
	if gr != nil {
		return gr.icon
	}
	return WinMedia{}
}

func (grs *SANodeGroups) getList() []string {

	var fns []string

	for _, gr := range grs.groups {
		fns = append(fns, gr.nodes...)
	}

	return fns
}

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

type SAGroupNode struct {
	name   string
	fn     func(node *SANode) bool
	render func(node *SANode, renderIt bool)
}

type SAGroup struct {
	name  string
	icon  WinMedia
	nodes []*SAGroupNode
}

func (gr *SAGroup) Find(name string) *SAGroupNode {
	for _, nd := range gr.nodes {
		if strings.EqualFold(nd.name, name) {
			return nd
		}
	}
	return nil
}

type SAGroups struct {
	groups []*SAGroup
	ui     *SAGroup
}

func InitSAGroups() SAGroups {
	var grs SAGroups

	path := "file:apps/base/resources/"

	grs.ui = &SAGroup{name: "UI", icon: InitWinMedia_url(path + "node_ui.png"), nodes: []*SAGroupNode{
		{name: "layout", render: SAExe_Render_Layout},
		{name: "dialog", render: SAExe_Render_Dialog},
		{name: "button", render: SAExe_Render_Button},
		{name: "text", render: SAExe_Render_Text},
		{name: "checkbox", render: SAExe_Render_Checkbox},
		{name: "switch", render: SAExe_Render_Switch},
		{name: "editbox", render: SAExe_Render_Editbox},
		{name: "divider", render: SAExe_Render_Divider},
		{name: "combo", render: SAExe_Render_Combo},
		{name: "color_palette", render: SAExe_Render_ColorPalette},
		{name: "color", render: SAExe_Render_ColorPicker},
		{name: "file_drop", render: SAExe_Render_FileDrop},
		{name: "file_picker", render: SAExe_Render_FilePicker},
		{name: "folder_picker", render: SAExe_Render_FolderPicker},
		{name: "calendar", render: SAExe_Render_Calendar},
		{name: "date", render: SAExe_Render_Date},
		{name: "map", render: SAExe_Render_Map},
		{name: "image", render: SAExe_Render_Image},
		{name: "list", render: SAExe_Render_List},
		{name: "table", render: SAExe_Render_Table},
		{name: "microphone", render: SAExe_Render_Microphone},
	}}
	grs.groups = append(grs.groups, grs.ui)

	grs.groups = append(grs.groups, &SAGroup{name: "Variables", icon: InitWinMedia_url(path + "node_vars.png"), nodes: []*SAGroupNode{
		{name: "vars", fn: SAExe_Vars},
		{name: "for", fn: SAExe_For},
		{name: "SetAttribute", fn: SAExe_SetAttribute},
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "File", icon: InitWinMedia_url(path + "node_file.png"), nodes: []*SAGroupNode{
		{name: "read_dir", fn: SAExe_File_dir},
		{name: "read_file", fn: SAExe_File_read},
		{name: "write_file", fn: SAExe_File_write},
	}})
	grs.groups = append(grs.groups, &SAGroup{name: "Convert", icon: InitWinMedia_url(path + "node_convert.png"), nodes: []*SAGroupNode{
		{name: "csv_to_json", fn: SAExe_Convert_CsvToJson},
		{name: "gpx_to_json", fn: SAExe_Convert_GpxToJson},
	}})
	grs.groups = append(grs.groups, &SAGroup{name: "SQLite", icon: InitWinMedia_url(path + "node_db.png"), nodes: []*SAGroupNode{
		{name: "sqlite_select", fn: SAExe_Sqlite_select},
		{name: "sqlite_insert", fn: SAExe_Sqlite_insert},
		//"sqlite_update", "sqlite_delete", "sqlite_execute" ........
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Neural networks", icon: InitWinMedia_url(path + "node_nn.png"), nodes: []*SAGroupNode{
		{name: "nn_whisper_cpp", fn: SAExe_NN_whisper_cpp},
		{name: "nn_llama_cpp", fn: SAExe_NN_llama_cpp},
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Coding", icon: InitWinMedia_url(path + "node_code.png"), nodes: []*SAGroupNode{
		{name: "code_python", fn: SAExe_Code_python},
	}})

	return grs
}
func SAGroups_HasNodeSub(node string) bool {
	return strings.EqualFold(node, "layout") || strings.EqualFold(node, "dialog") || strings.EqualFold(node, "for")
}
func SAGroups_IsNodeFor(node string) bool {
	return strings.EqualFold(node, "for")
}

func (grs *SAGroups) IsUI(node string) bool {
	return grs.ui.Find(node) != nil
}

func (grs *SAGroups) FindNode(node string) *SAGroupNode {
	for _, gr := range grs.groups {
		nd := gr.Find(node)
		if nd != nil {
			return nd
		}
	}
	return nil
}

func (grs *SAGroups) FindNodeGroup(node string) *SAGroup {

	for _, gr := range grs.groups {
		if gr.Find(node) != nil {
			return gr
		}
	}

	return nil
}
func (grs *SAGroups) FindNodeGroupIcon(node string) WinMedia {

	gr := grs.FindNodeGroup(node)
	if gr != nil {
		return gr.icon
	}
	return WinMedia{}
}

func (grs *SAGroups) getList() []string {

	var fns []string

	for _, gr := range grs.groups {
		for _, nd := range gr.nodes {
			fns = append(fns, nd.name)
		}
	}

	return fns
}

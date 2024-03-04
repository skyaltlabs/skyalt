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
	"strings"
)

type SAGroupNode struct {
	name   string
	render func(node *SANode)
	attrs  func(node *SANode)
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
}

func InitSAGroups() SAGroups {
	var grs SAGroups

	path := "file:apps/base/resources/"

	grs.groups = append(grs.groups, &SAGroup{name: "UI", icon: InitWinMedia_url(path + "node_ui.png"), nodes: []*SAGroupNode{
		{name: "button", render: UiButton_render, attrs: UiButton_Attrs},
		{name: "text", render: UiText_render, attrs: UiText_Attrs},
		{name: "editbox", render: UiEditbox_render, attrs: UiEditbox_Attrs},
		{name: "checkbox", render: UiCheckbox_render, attrs: UiCheckbox_Attrs},
		{name: "switch", render: UiSwitch_render, attrs: UiSwitch_Attrs},
		/*
			{name: "slider", render: SAExe_Render_Slider},
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
			{name: "layout", render: SAExe_Render_Layout},
			{name: "dialog", render: SAExe_Render_Dialog},*/
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Disk access", icon: InitWinMedia_url(path + "node_file.png"), nodes: []*SAGroupNode{
		{name: "disk_dir", render: UiDiskDir_render, attrs: UiDiskDir_Attrs},
		{name: "disk_file", render: UiDiskFile_render, attrs: UiDiskFile_Attrs},
		/*{name: "sqlite", fn: SAExe_File_sqlite},
		{name: "read_file", fn: SAExe_File_read},
		{name: "write_file", fn: SAExe_File_write},*/
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Neural networks", icon: InitWinMedia_url(path + "node_nn.png"), nodes: []*SAGroupNode{
		{name: "nn_whisper_cpp", attrs: UiWhisperCpp_Attrs},
		{name: "nn_llama_cpp", attrs: UiLLamaCpp_Attrs},
		{name: "nn_g4f", attrs: UiG4F_Attrs},
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Coding", icon: InitWinMedia_url(path + "node_code.png"), nodes: []*SAGroupNode{
		{name: "code_go", attrs: UiCodeGo_Attrs},
		//{name: "code_python", fn: SAExe_Code_python},
	}})

	return grs
}

func SAGroups_HasNodeSub(node string) bool {
	return strings.EqualFold(node, "layout") || strings.EqualFold(node, "dialog") // || strings.EqualFold(node, "for") || strings.EqualFold(node, "if")
}

func (grs *SAGroups) IsUI(node string) bool {
	gr := grs.FindNode(node)
	return gr != nil && gr.render != nil
}

func (grs *SAGroups) IsTrigger(node string) bool {
	return strings.EqualFold(node, "button") || strings.EqualFold(node, "editbox")
}

func (grs *SAGroups) IsCode(node string) bool {
	return strings.EqualFold(node, "code_go") || strings.EqualFold(node, "code_python")
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

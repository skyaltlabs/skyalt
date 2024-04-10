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
		{name: "slider", render: UiSlider_render, attrs: UiSlider_Attrs},
		{name: "color", render: UiColor_render, attrs: UiColor_Attrs},
		{name: "combo", render: UiCombo_render, attrs: UiCombo_Attrs},
		{name: "divider", render: UiDivider_render, attrs: UiDivider_Attrs},
		{name: "timer", render: UiTimer_render, attrs: UiTimer_Attrs},
		{name: "date", render: UiDate_render, attrs: UiDate_Attrs},
		{name: "microphone", render: UiMicrophone_render, attrs: UiMicrophone_Attrs},
		{name: "map", render: UiMap_render, attrs: UiMap_Attrs},
		{name: "layout", render: UiLayout_render, attrs: UiLayout_Attrs},
		{name: "list", render: UiList_render, attrs: UiList_Attrs},
		{name: "chart", render: UiChart_render, attrs: UiChart_Attrs},

		/*	{name: "image", render: SAExe_Render_Image},
			{name: "dialog", render: SAExe_Render_Dialog},*/
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Access", icon: InitWinMedia_url(path + "node_file.png"), nodes: []*SAGroupNode{
		{name: "disk_dir", render: UiDiskDir_render, attrs: UiDiskDir_Attrs},
		{name: "disk_file", render: UiDiskFile_render, attrs: UiDiskFile_Attrs},
		{name: "db_file", render: UiSQLite_render, attrs: UiSQLite_Attrs},
		{name: "net", attrs: UiNet_Attrs},
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Neural networks", icon: InitWinMedia_url(path + "node_nn.png"), nodes: []*SAGroupNode{
		{name: "whispercpp", attrs: UiWhisperCpp_Attrs},
		{name: "llamacpp", attrs: UiLLamaCpp_Attrs},
		{name: "openai", attrs: UiOpenAI_Attrs},
	}})

	grs.groups = append(grs.groups, &SAGroup{name: "Functions", icon: InitWinMedia_url(path + "node_code.png"), nodes: []*SAGroupNode{
		{name: "code", attrs: UiCode_Attrs},
	}})

	return grs
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

func (grs *SAGroups) getList() []string {

	var fns []string

	for _, gr := range grs.groups {
		for _, nd := range gr.nodes {
			fns = append(fns, nd.name)
		}
	}

	return fns
}

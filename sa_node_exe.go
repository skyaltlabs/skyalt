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
	"path/filepath"
	"strings"
)

func UiButton_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "grid_show", true)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "clicked", false)
}

func UiButton_render(node *SANode) {
	grid := node.GetGrid()
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//clicked := node.GetAttrBool("clicked", false)

	if node.app.base.ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, tooltip, enable) > 0 {
		node.Attrs["clicked"] = true
	}
}

func UiText_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "grid_show", true)
	node.ShowAttrString(&grid, "label", "", node.GetAttrBool("multi_line", false))
	node.ShowAttrIntCombo(&grid, "align_h", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrIntCombo(&grid, "align_v", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrBool(&grid, "multi_line", false)
	node.ShowAttrBool(&grid, "selection", true)
	node.ShowAttrBool(&grid, "show_border", false)
}

func UiText_render(node *SANode) {
	grid := node.GetGrid()
	label := node.GetAttrString("label", "")
	align_v := node.GetAttrInt("align_v", 0)
	align_h := node.GetAttrInt("align_h", 0)
	selection := node.GetAttrBool("selection", true)
	show_border := node.GetAttrBool("show_border", false)

	if node.GetAttrBool("multi_line", false) {
		node.app.base.ui.Comp_textSelectMulti(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, show_border)
	} else {
		node.app.base.ui.Comp_textSelect(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, label, OsV2{align_h, align_v}, selection, show_border)
	}
}

func UiEditbox_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "grid_show", true)
	node.ShowAttrString(&grid, "value", "", node.GetAttrBool("multi_line", false))
	node.ShowAttrString(&grid, "ghost", "", false)
	node.ShowAttrIntCombo(&grid, "align_h", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrIntCombo(&grid, "align_v", 0, []string{"Left", "Center", "Right"}, []string{"0", "1", "2"})
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "multi_line", false)
	node.ShowAttrBool(&grid, "multi_line_enter_finish", false)
	node.ShowAttrBool(&grid, "finished", false)
}

func UiEditbox_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrString("value", "")
	ghost := node.GetAttrString("ghost", "")
	align_v := node.GetAttrInt("align_v", 0)
	align_h := node.GetAttrInt("align_h", 0)
	enable := node.GetAttrBool("enable", true)
	multi_line := node.GetAttrBool("multi_line", false)
	multi_line_enter_finish := node.GetAttrBool("multi_line_enter_finish", false)
	//finished := node.GetAttrBool("finished", false)

	_, _, _, fnshd, _ := node.app.base.ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, Comp_editboxProp().Ghost(ghost).MultiLine(multi_line).MultiLineEnterFinish(multi_line_enter_finish).Enable(enable).Align(align_h, align_v))
	if fnshd {
		node.Attrs["value"] = value
		node.Attrs["finished"] = true
	}
}

func UiCheckbox_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "grid_show", true)
	node.ShowAttrBool(&grid, "value", false)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiCheckbox_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrBool("value", false)
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, tooltip, enable) {
		node.Attrs["value"] = value
		node.Attrs["changed"] = true
	}
}

func UiSwitch_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 4)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrV4(&grid, "grid", InitOsV4(0, 0, 1, 1))
	node.ShowAttrBool(&grid, "grid_show", true)
	node.ShowAttrBool(&grid, "value", false)
	node.ShowAttrString(&grid, "label", "", false)
	node.ShowAttrString(&grid, "tooltip", "", false)
	node.ShowAttrBool(&grid, "enable", true)
	node.ShowAttrBool(&grid, "changed", false)
}

func UiSwitch_render(node *SANode) {
	grid := node.GetGrid()
	value := node.GetAttrBool("value", false)
	label := node.GetAttrString("label", "")
	tooltip := node.GetAttrString("tooltip", "")
	enable := node.GetAttrBool("enable", true)
	//changed := node.GetAttrBool("changed", false)

	if node.app.base.ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, label, tooltip, enable) {
		node.Attrs["value"] = value
		node.Attrs["changed"] = true
	}
}

func UiCodeGo_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)
	ui.Div_colMax(2, 100)

	ui.Div_row(0, 1)
	ui.Div_row(2, 3)
	ui.Div_rowMax(0, 2)
	ui.Div_rowMax(2, 6)

	ui.Comp_text(0, 0, 1, 1, "Request", 0)
	ui.Comp_text(0, 2, 1, 1, "Answer", 0)

	ui.Comp_editbox(1, 0, 2, 1, &node.Code.TempCommand, Comp_editboxProp().Align(0, 0).MultiLine(true).TempToValue(true))

	if node.Code.TempCommand != node.Code.Command {
		ui.Comp_textCd(1, 1, 1, 1, "Warning: Re-generage answer", 0, CdPalette_E)
	}

	//generate button
	if ui.Comp_button(2, 1, 1, 1, "Generate", "", node.Code.TempCommand != node.Code.Command) > 0 {
		err := node.Code.GetAnswer()
		if err != nil {
			node.SetError(err)
		}
	}

	//answer
	ui.Comp_editbox(1, 2, 2, 1, &node.Code.Answer, Comp_editboxProp().Align(0, 0).MultiLine(true))

	//run button
	if ui.Comp_button(2, 3, 1, 1, "Run", "", true) > 0 {
		err := node.Code.Execute()
		if err != nil {
			node.SetError(err)
		}
	}

	//triggers
	ui.Comp_text(0, 4, 1, 1, "Triggers", 0)
	ui.Div_start(1, 4, 2, len(node.Code.Triggers))
	{
		ui.Div_colMax(1, 100)
		for i, tr := range node.Code.Triggers {
			if ui.Comp_button(0, i, 1, 1, "X", "Un-link", true) > 0 {
				node.Code.Triggers = append(node.Code.Triggers[:i], node.Code.Triggers[i+1:]...) //remove
			}
			ui.Comp_text(1, i, 1, 1, tr, 0)
		}
	}
	ui.Div_end()
}

func UiDiskDir_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	ui.Comp_text(0, 0, 1, 1, "path", 0)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrFilePicker(&grid, "path", "", false)
	node.ShowAttrBool(&grid, "write", false)
}

var g_whisper_formats = []string{"verbose_json", "json", "text", "srt", "vtt"}
var g_whisper_modelList = []string{"ggml-tiny.en", "ggml-tiny", "ggml-base.en", "ggml-base", "ggml-small.en", "ggml-small", "ggml-medium.en", "ggml-medium", "ggml-large-v1", "ggml-large-v2", "ggml-large-v3"}
var g_whisper_modelsFolder = "services/whisper.cpp/models/"

func UiWhisperCpp_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	//build model list
	var models []string
	for _, m := range g_whisper_modelList {
		if m != "" { //1st is empty
			if OsFileExists(filepath.Join(g_whisper_modelsFolder, m+".bin")) {
				models = append(models, m)
			}
		}
	}

	node.ShowAttrStringCombo(&grid, "model", OsTrnString(len(models) > 0, models[0], ""), models, models)

	node.ShowAttrInt(&grid, "offset_t", 0)
	node.ShowAttrInt(&grid, "offset_n", 0)
	node.ShowAttrInt(&grid, "duration", 0)
	node.ShowAttrInt(&grid, "max_context", -1)
	node.ShowAttrInt(&grid, "max_len", 0)
	node.ShowAttrInt(&grid, "best_of", 2)
	node.ShowAttrInt(&grid, "beam_size", -1)

	node.ShowAttrFloat(&grid, "word_thold", 0.01, 3)
	node.ShowAttrFloat(&grid, "entropy_thold", 2.4, 3)
	node.ShowAttrFloat(&grid, "logprob_thold", -1, 3)

	node.ShowAttrBool(&grid, "translate", false)
	node.ShowAttrBool(&grid, "diarize", false)
	node.ShowAttrBool(&grid, "tinydiarize", false)
	node.ShowAttrBool(&grid, "split_on_word", false)
	node.ShowAttrBool(&grid, "no_timestamps", false)

	node.ShowAttrString(&grid, "language", "", false)
	node.ShowAttrBool(&grid, "detect_language", false)

	node.ShowAttrFloat(&grid, "temperature", 0, 3)
	node.ShowAttrFloat(&grid, "temperature_inc", 0.2, 3)

	node.ShowAttrStringCombo(&grid, "response_format", "verbose_json", g_whisper_formats, g_whisper_formats)

	// downloader ...
}

var g_llama_modelsFolder = "services/llama.cpp/models/"

func UiLLamaCpp_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	//build model list
	var models []string
	modelFiles := OsFileListBuild(g_llama_modelsFolder, "", true)
	for _, m := range modelFiles.Subs {
		if !m.IsDir && !strings.HasPrefix(m.Name, "ggml-vocab") {
			models = append(models, m.Name)
		}
	}

	node.ShowAttrStringCombo(&grid, "model", OsTrnString(len(models) > 0, models[0], ""), models, models)

	//...
	/*stopAttr := node.GetAttr("stop", []byte(`["</s>", "Llama:", "User:"]`))
	err := json.Unmarshal(stopAttr.GetBlob().data, &props.Stop)
	if err != nil {
		stopAttr.SetError(err)
	}*/

	node.ShowAttrInt(&grid, "seed", -1)
	node.ShowAttrInt(&grid, "n_predict", 400)

	node.ShowAttrFloat(&grid, "temperature", 0.8, 3)
	node.ShowAttrFloat(&grid, "dynatemp_range", 0.0, 3)
	node.ShowAttrFloat(&grid, "dynatemp_exponent", 1.0, 3)
	node.ShowAttrInt(&grid, "repeat_last_n", 256)
	node.ShowAttrFloat(&grid, "repeat_penalty", 1.18, 3)

	node.ShowAttrInt(&grid, "top_k", 40)
	node.ShowAttrFloat(&grid, "top_p", 0.5, 3)
	node.ShowAttrFloat(&grid, "min_p", 0.05, 3)
	node.ShowAttrFloat(&grid, "tfs_z", 1.0, 3)
	node.ShowAttrFloat(&grid, "typical_p", 1.0, 3)
	node.ShowAttrFloat(&grid, "presence_penalty", 0.0, 3)
	node.ShowAttrFloat(&grid, "frequency_penalty", 0.0, 3)
	node.ShowAttrInt(&grid, "mirostat", 0)
	node.ShowAttrFloat(&grid, "mirostat_tau", 5, 3)
	node.ShowAttrFloat(&grid, "mirostat_eta", 0.1, 3)
	//Grammar
	node.ShowAttrInt(&grid, "n_probs", 0)
	//Image_data
	node.ShowAttrBool(&grid, "cache_prompt", false)
	node.ShowAttrInt(&grid, "slot_id", -1)

	//downloader ...
}

var g_g4f_modelList = []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"}

func UiG4F_Attrs(node *SANode) {
	ui := node.app.base.ui
	ui.Div_colMax(0, 3)
	ui.Div_colMax(1, 100)

	grid := InitOsV4(0, 0, 1, 1)

	node.ShowAttrStringCombo(&grid, "model", g_g4f_modelList[0], g_g4f_modelList, g_g4f_modelList)
}

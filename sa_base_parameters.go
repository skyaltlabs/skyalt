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

	ui.Comp_editbox_desc("Name", 0, 2, 0, 0, 1, 1, &node.Name, 0, "", "", false, false) //rename

	ui.Div_SpacerRow(0, 1, 1, 1)

	fn := app.nodes.FindFn(node.FnName)
	if fn != nil && fn.parameters != nil {
		ui.Div_start(0, 2, 1, 1)
		fn.parameters(node, ui)
		ui.Div_end()
	}
}

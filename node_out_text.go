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

func NodeOutText_init() *NodeFnDef {
	fn := NewNodeFnDef("out_text", NodeOutText_exe, NodeOutText_parameters, NodeOutText_render)
	return fn
}

func NodeOut_renderGrid(x, y, w, h int, node *Node, ui *Ui) {
	ui.Div_start(x, y, w, h)

	ui.Div_colMax(0, 100)
	ui.Div_colMax(1, 100)
	ui.Div_colMax(2, 100)
	ui.Div_colMax(3, 100)
	ui.Div_colMax(4, 100)

	ui.Comp_text(0, 0, 1, 1, "Grid(x, y, w, h)", 0)
	ui.Comp_editbox(1, 0, 1, 1, &node.GridCoord.Start.X, 0, "", "", false, false)
	ui.Comp_editbox(2, 0, 1, 1, &node.GridCoord.Start.Y, 0, "", "", false, false)
	ui.Comp_editbox(3, 0, 1, 1, &node.GridCoord.Size.X, 0, "", "", false, false)
	ui.Comp_editbox(4, 0, 1, 1, &node.GridCoord.Size.Y, 0, "", "", false, false)

	ui.Div_end()
}

func NodeOutText_parameters(node *Node, ui *Ui) {

	ui.Div_colMax(0, 100)

	NodeOut_renderGrid(0, 0, 1, 1, node, ui)

	ui.Comp_editbox_desc("Text", 0, 2, 0, 1, 1, 1, node.GetParam("text"), 0, "", "", false, false)

	ui.Comp_combo_desc("Align", 0, 2, 0, 2, 1, 1, node.GetParam("align"), "Left|Center|Right", "", true, false)
	//alignV ...
	//textH ...
}

func NodeOutText_exe(inputs []NodeData, node *Node, nodes *Nodes) ([]NodeData, error) {

	return nil, nil
}

func NodeOutText_render(node *Node, ui *Ui) {
	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)
	ui.Comp_text(0, 0, 1, 1, node.GetParamString("text"), node.GetParamInt("align"))
}

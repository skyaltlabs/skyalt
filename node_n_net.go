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
	"fmt"
	"strconv"
)

func NodeNet_build() *NodeFnDef {
	fn := NewNodeFnDef("net", NodeSub_init, NodeSub_parameters, NodeSub_exe, NodeSub_render)

	return fn
}

func NodeSub_init(node *Node) {

	node.Cam_z = 1
	node.Grid_w = 1
	node.Grid_h = 1

	for i := 0; i < 5; i++ {
		node.Cols = append(node.Cols, NodeColRow{Min: 1, Max: 1, Resize: 1})
		node.Rows = append(node.Rows, NodeColRow{Min: 1, Max: 1, Resize: 1})
	}

}

func NodeOut_renderGrid(x, y, w, h int, node *Node, ui *Ui) {
	ui.Div_start(x, y, w, h)

	ui.Div_colMax(0, 100)
	ui.Div_colMax(1, 100)
	ui.Div_colMax(2, 100)
	ui.Div_colMax(3, 100)
	ui.Div_colMax(4, 100)

	ui.Comp_text(0, 0, 1, 1, "Grid(x, y, w, h)", 0)
	ui.Comp_editbox(1, 0, 1, 1, &node.Grid_x, 0, "", "", false, false, true)
	ui.Comp_editbox(2, 0, 1, 1, &node.Grid_y, 0, "", "", false, false, true)
	ui.Comp_editbox(3, 0, 1, 1, &node.Grid_w, 0, "", "", false, false, true)
	ui.Comp_editbox(4, 0, 1, 1, &node.Grid_h, 0, "", "", false, false, true)

	ui.Div_end()
}

func NodeSub_parameters(node *Node, ui *Ui) {

	ui.Div_colMax(0, 100)

	NodeOut_renderGrid(0, 0, 1, 1, node, ui)

	ui.Comp_editbox_desc("Cam_x", 0, 2, 0, 1, 1, 1, &node.Cam_x, 0, "", "", false, false, true)
	ui.Comp_editbox_desc("Cam_y", 0, 2, 0, 2, 1, 1, &node.Cam_y, 0, "", "", false, false, true)
	ui.Comp_editbox_desc("Cam_z", 0, 2, 0, 3, 1, 1, &node.Cam_z, 0, "", "", false, false, true)

	//ui.Comp_switch(0, 3, 1, 1, node.GetParam("show_grid"), false, "Show grid", "", true)
}

func NodeSub_exe(inputs []NodeData, node *Node) ([]NodeData, error) {

	//node.ExecuteSubs()	//...

	return nil, nil
}

func NodeSub_render(node *Node, ui *Ui) {

	//if node.app.showColRow ...

	//node.drawApp(ui)
	node.drawAppDevMode(ui)

}

func (node *Node) drawApp(ui *Ui) {

	for i, c := range node.Cols {
		ui.Div_col(i, c.Min)
		ui.Div_colMax(i, c.Max)
		if c.ResizeName != "" {
			active, v := ui.Div_colResize(i, c.ResizeName, c.Resize, true)
			if active {
				node.Cols[i].Resize = v
			}
		}
	}

	for i, r := range node.Rows {
		ui.Div_row(i, r.Min)
		ui.Div_rowMax(i, r.Max)
		if r.ResizeName != "" {
			active, v := ui.Div_rowResize(i, r.ResizeName, r.Resize, true)
			if active {
				node.Rows[i].Resize = v
			}
		}
	}

	for _, n := range node.Subs {
		if n.Bypass {
			continue
		}

		fn := node.FindFn(n.FnName)
		if fn != nil && fn.render != nil {
			ui.Div_start(n.Grid_x, n.Grid_y, n.Grid_w, n.Grid_h)
			{
				fn.render(n, ui)

				if n.Selected {
					ui.Paint_rect(0, 0, 1, 1, 0, Node_getYellow(), 0.03)
				}

				//alt+click => select node and zoom_in network ...
			}
			ui.Div_end()
		}
	}

}

func (node *Node) drawColsRowsDialog(name string, items *[]NodeColRow, i int, ui *Ui) {

	if ui.Dialog_start(name) {

		ui.Div_col(0, 10)

		//add left/right
		ui.Div_start(0, 0, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)
			ui.Div_colMax(2, 100)

			if ui.Comp_buttonLight(0, 0, 1, 1, "Add before", "", i > 0) > 0 {
				*items = append(*items, NodeColRow{})
				copy((*items)[i+1:], (*items)[i:])
				(*items)[i] = NodeColRow{Min: 1, Max: 1, Resize: 1}
				ui.Dialog_close()
			}

			ui.Comp_text(1, 0, 1, 1, strconv.Itoa(i), 1) //description

			if ui.Comp_buttonLight(2, 0, 1, 1, "Add after", "", true) > 0 {
				*items = append(*items, NodeColRow{})
				copy((*items)[i+2:], (*items)[i+1:])
				(*items)[i+1] = NodeColRow{Min: 1, Max: 1, Resize: 1}
				ui.Dialog_close()
			}
		}
		ui.Div_end()

		ui.Comp_editbox_desc("Min", 0, 2, 0, 1, 1, 1, &(*items)[i].Min, 2, "", "", false, false, true)
		ui.Comp_editbox_desc("Max", 0, 2, 0, 2, 1, 1, &(*items)[i].Max, 2, "", "", false, false, true)

		ui.Div_start(0, 3, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)

			ui.Comp_editbox_desc("Resize", 0, 2, 0, 0, 1, 1, &(*items)[i].ResizeName, 2, "", "Name", false, true, true)
			ui.Comp_text(1, 0, 1, 1, strconv.FormatFloat((*items)[i].Resize, 'f', 2, 64), 0)
		}
		ui.Div_end()

		//remove
		if ui.Comp_button(0, 5, 1, 1, "Remove", "", len(*items) > 1) > 0 {
			*items = append((*items)[:i], (*items)[i+1:]...)
			ui.Dialog_close()
		}

		ui.Dialog_end()
	}
}

func (node *Node) drawAppDevMode(ui *Ui) {

	ui.Div_colMax(1, 100)
	ui.Div_rowMax(1, 100)

	var colDiv *UiLayoutDiv
	var rowDiv *UiLayoutDiv

	//cols header
	ui.Div_start(1, 0, 1, 1)
	{
		colDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		for i, c := range node.Cols {
			ui.Div_col(i, c.Min)
			ui.Div_colMax(i, c.Max)
			if c.ResizeName != "" {
				active, v := ui.Div_colResize(i, c.ResizeName, c.Resize, true)
				if active {
					node.Cols[i].Resize = v
				}
			}
		}

		for i := range node.Cols {

			nm := fmt.Sprintf("col_details_%d", i)

			//drag & drop
			ui.Div_start(i, 0, 1, 1)
			{
				ui.Div_drag("cols", i)
				src, pos, done := ui.Div_drop("cols", false, true, false)
				if done {
					Div_DropMoveElement(&node.Cols, &node.Cols, src, i, pos)
				}
			}
			ui.Div_end()

			if ui.Comp_button(i, 0, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
				ui.Dialog_open(nm, 1)
			}

			node.drawColsRowsDialog(nm, &node.Cols, i, ui)
		}

		//"+" to append new ...
	}
	ui.Div_end()

	//rows header
	ui.Div_start(0, 1, 1, 1)
	{
		rowDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		for i, r := range node.Rows {
			ui.Div_row(i, r.Min)
			ui.Div_rowMax(i, r.Max)

			if r.ResizeName != "" {
				active, v := ui.Div_rowResize(i, r.ResizeName, r.Resize, true)
				if active {
					node.Rows[i].Resize = v
				}
			}
		}

		for i := range node.Rows {

			nm := fmt.Sprintf("row_details_%d", i)

			if ui.Comp_button(0, i, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
				ui.Dialog_open(nm, 1)
			}
			node.drawColsRowsDialog(nm, &node.Rows, i, ui)
		}
	}
	ui.Div_end()

	//app
	ui.Div_start(1, 1, 1, 1)

	ui.GetCall().call.data.scrollH.attach = &colDiv.data.scrollH
	ui.GetCall().call.data.scrollV.attach = &rowDiv.data.scrollV

	node.drawApp(ui)
	ui.Div_end()
}

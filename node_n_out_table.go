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

import "fmt"

func NodeOutTable_build() *NodeFnDef {
	fn := NewNodeFnDef("out_table", NodeOutTable_init, NodeOutTable_parameters, NodeOutTable_exe, NodeOutTable_render)
	fn.AddInput("db", false)
	return fn
}

func NodeOutTable_init(node *Node) {
	node.Grid_w = 1
	node.Grid_h = 1
}

func NodeOutTable_parameters(node *Node, ui *Ui) {
	ui.Div_colMax(0, 100)
	NodeOut_renderGrid(0, 0, 1, 1, node, ui)

}

func NodeOutTable_exe(inputs []NodeData, node *Node) ([]NodeData, error) {

	if len(inputs) == 0 {
		return nil, fmt.Errorf("0 inputs")
	}

	if len(inputs[0].dbs) == 0 {
		return nil, fmt.Errorf("0 databases in input")
	}

	var err error
	node.db, err = NewDiskDb(inputs[0].dbs[0].path, false, node.app.disk)

	return nil, err
}

func NodeOutTable_render(node *Node, ui *Ui) {

	if node.db == nil {
		return
	}
	//musíme volat nejdříve Execute a potom až render() ... .............................

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	tables, err := node.db.GetTableInfo()
	if err != nil {
		return
	}
	table := tables[0]

	for i := range table.Columns {
		ui.Div_colMax(i, 5)
		//resize ...
	}

	//head
	for i, col := range table.Columns {
		ui.Comp_text(0, i, 1, 1, col.Name+"("+col.Type+")", 1)
	}

	//rows
	count := 0
	row := node.db.db.QueryRow("SELECT COUNT(*) FROM " + table.Name)
	if row != nil {
		row.Scan(&count)
	}
	rowSize := 1
	st, en := ui.DivRange_Ver(float64(rowSize))
	{
		//before
		y := 0
		ui.Div_row(y, float64(st*rowSize))
		y++

		//visible
		for i := 0; i < (en - st); i++ {
			ui.Div_row(y, float64(rowSize))
			y++
		}

		//after
		if en < count {
			ui.Div_row(y, float64((count-en)*rowSize))
			y++
		}
	}

	rows, err := node.db.db.Query("SELECT * FROM " + table.Name)
	if err == nil {
		for rows.Next() {
			//row.Scan(&count)

			//...

		}

	}
}

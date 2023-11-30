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

func NodeSelect_build() *NodeFnDef {
	fn := NewNodeFnDef("select", nil, NodeSelect_parameters, NodeSelect_exe, nil)
	fn.AddInput("db", false)
	fn.AddOutput("db", false)
	return fn
}

func NodeSelect_parameters(node *Node, ui *Ui) {

	ui.Div_colMax(0, 100)

	ui.Comp_editbox_desc("Query", 0, 2, 0, 0, 1, 1, node.GetParam("query"), 0, "", "", false, false, true)
}

func NodeSelect_exe(inputs []NodeData, node *Node) ([]NodeData, error) {

	var outs []NodeData
	outs = append(outs, NodeData{})
	outs[0].dbs = append(outs[0].dbs, NodeDataDb{path: node.Name, alias: node.Name, inMemory: true})

	//create :memory db
	var err error
	node.db, err = NewDiskDb(node.Name, true, node.app.disk)
	if err != nil {
		return nil, fmt.Errorf("NewDiskDb() failed: %w", err)
	}

	//attach inputs
	for _, in := range inputs {
		for _, d := range in.dbs {
			err = node.db.Attach(d.path, d.alias, d.inMemory)
			if err != nil {
				return nil, fmt.Errorf("Attach() failed: %w", err)
			}
		}
	}

	//run query
	_, err = node.db.Write("CREATE TABLE main.result AS " + node.GetParamString("query"))
	if err != nil {
		return nil, fmt.Errorf("Write() failed: %w", err)
	}

	node.db.Commit()

	//maybe DETOUCH
	for _, in := range inputs {
		for _, d := range in.dbs {
			err = node.db.Detach(d.alias)
			if err != nil {
				return nil, fmt.Errorf("Attach() failed: %w", err)
			}
		}
	}

	return outs, nil
}

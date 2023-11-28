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

func NodeFile_init() *NodeFnDef {
	fn := NewNodeFnDef("file", NodeFile_exe, NodeFile_parameterss)
	//fn.AddParam("path", "", NodeFn_EDITBOX)
	//fn.AddParam("alias", "", NodeFn_EDITBOX)
	fn.AddOutput("db", false)
	return fn
}

func NodeFile_parameterss(node *Node, ui *Ui) {

	ui.Div_colMax(0, 100)

	ui.Comp_editbox_desc("Path", 0, 2, 0, 0, 1, 1, node.Parameters["path"], 0, "", "", false, false)
	ui.Comp_editbox_desc("Alias", 0, 2, 0, 1, 1, 1, node.Parameters["alias"], 0, "", "", false, false)
}

func NodeFile_exe(inputs []NodeData, node *Node, nodes *Nodes) ([]NodeData, error) {

	path := nodes.disk.folder + "/" + node.Parameters["path"]
	alias := node.Parameters["alias"]

	if alias == "" {
		alias = node.Name
	}

	if path == "" {
		return nil, fmt.Errorf("path is empty")
	}

	var outs []NodeData
	outs = append(outs, NodeData{})
	outs[0].dbs = append(outs[0].dbs, NodeDataDb{path: path, alias: alias, inMemory: false})

	return outs, nil
}

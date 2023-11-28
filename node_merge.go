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

func NodeMerge_init() *NodeFnDef {
	fn := NewNodeFnDef("merge", NodeFile_exe, nil)
	fn.SetInfiniteInputs(true)
	fn.AddOutput("db", false)
	return fn
}

func NodeMerge_exe(inputs []NodeData, node *Node, nodes *Nodes) ([]NodeData, error) {

	var outs []NodeData
	outs = append(outs, NodeData{})

	for _, in := range inputs {
		outs[0].dbs = append(outs[0].dbs, in.dbs...)
	}

	return outs, nil
}

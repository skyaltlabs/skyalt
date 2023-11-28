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
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type NodeIn struct {
	Node string
	Pos  int
}

type NodeDataDb struct {
	path     string
	alias    string
	inMemory bool
}

type NodeData struct {
	dbs []NodeDataDb
	//json string
}

const (
	NodeFn_EDITBOX  = 0
	NodeFn_SLIDER   = 1
	NodeFn_SWITCH   = 2
	NodeFn_CHECKBOX = 3
)

type NodeFnDefIO struct {
	name string
	json bool
}
type NodeFnDef struct {
	name       string
	fn         func(inputs []NodeData, node *Node, nodes *Nodes) ([]NodeData, error)
	parameters func(node *Node, ui *Ui)

	//params       []NodeFnDefParam
	ins          []NodeFnDefIO
	outs         []NodeFnDefIO
	infinite_ins bool
	//infinite_outs bool
}

func NewNodeFnDef(name string, fnPtr func(inputs []NodeData, node *Node, nodes *Nodes) ([]NodeData, error), parametersPtr func(node *Node, ui *Ui)) *NodeFnDef {
	var fn NodeFnDef
	fn.name = name
	fn.fn = fnPtr
	fn.parameters = parametersPtr
	return &fn
}

func (fn *NodeFnDef) AddInput(name string, isJson bool) {
	fn.ins = append(fn.ins, NodeFnDefIO{name: name, json: isJson})
}
func (fn *NodeFnDef) AddOutput(name string, isJson bool) {
	fn.outs = append(fn.outs, NodeFnDefIO{name: name, json: isJson})
}
func (fn *NodeFnDef) SetInfiniteInputs(enable bool) {
	fn.infinite_ins = enable
}

type Node struct {
	Name string

	Pos            OsV2f
	pos_start      OsV2f
	move_active    bool
	Selected       bool
	selected_cover bool

	FnName     string
	Parameters map[string]string
	Inputs     []NodeIn

	Bypass bool

	outputs []NodeData

	changed bool
	running bool
	done    bool

	err error

	//info_action   string
	//info_progress float64 //0-1

	db *DiskDb
}

func NewNode(name string, fnName string) *Node {
	var node Node
	node.Name = name
	node.Parameters = make(map[string]string)
	node.FnName = fnName
	node.changed = true
	return &node
}

func (node *Node) Destroy() {
	if node.db != nil {
		node.db.Destroy()
	}
	//for _, out := range node.outputs {
	//	out.Destroy()
	//}
}

func (node *Node) KeyProgessSelection(keys *WinKeys) bool {

	if keys.shift {
		if node.selected_cover {
			return true
		}
		return node.Selected
	} else if keys.ctrl {
		if node.selected_cover {
			return false
		}
		return node.Selected
	}

	return node.selected_cover
}

func (node *Node) SetParam(key, value string) {
	node.Parameters[key] = value
	node.changed = true
}

func (node *Node) SetInput(out_pos int, src *Node, src_pos int) {

	for i := len(node.Inputs); i <= out_pos; i++ {
		node.Inputs = append(node.Inputs, NodeIn{})
	}

	node.Inputs[out_pos] = NodeIn{Node: src.Name, Pos: src_pos}
	node.changed = true
}

func (node *Node) RenameInputs(src, dst string) {
	for _, in := range node.Inputs {
		if in.Node == src {
			in.Node = dst
		}
	}
}
func (node *Node) RemoveInputs(name string) {
	for _, in := range node.Inputs {
		if in.Node == name {
			in.Node = ""
		}
	}
}

func (node *Node) Execute(nodes *Nodes) {
	//function
	fn := nodes.FindFn(node.FnName)
	if fn == nil {
		node.err = fmt.Errorf("Function(%s) not found", node.FnName)
		return
	}

	//free previous db
	if node.db != nil {
		node.db.Destroy()
		node.db = nil
	}

	//inputs
	var ins []NodeData
	for i, in := range node.Inputs {
		n := nodes.FindNode(in.Node)
		if n == nil {
			node.err = fmt.Errorf("Node(%s) for %d input not found", in.Node, i+1)
			return
		}

		//resize
		if in.Pos >= len(n.outputs) {
			t := n.outputs
			n.outputs = make([]NodeData, in.Pos+1)
			copy(n.outputs, t)
		}

		//set
		ins = append(ins, n.outputs[in.Pos])
	}

	//call
	if node.Bypass {
		//copy inputs to outputs
		node.outputs = make([]NodeData, len(ins))
		copy(node.outputs, ins)
	} else {
		node.outputs, node.err = fn.fn(ins, node, nodes)
		if node.err != nil {
			fmt.Printf("Node(%s) has error(%v)\n", node.Name, node.err)
		}
	}

	if !nodes.IsRunning() {
		node.err = fmt.Errorf("Interrupted")
		return
	}

	if node.db != nil {
		node.db.Commit()
	}

	node.changed = true //child can update
	node.done = true
	node.running = false
}

func (node *Node) areInputsErrorFree(nodes *Nodes) bool {

	for i, in := range node.Inputs {
		n := nodes.FindNode(in.Node)
		if n == nil {
			node.err = fmt.Errorf("Node(%s) for %d input not found", in.Node, i+1)
			return false
		}

		if n.err != nil {
			n.err = fmt.Errorf("incomming error from %d input", i+1)
			return false
		}
	}

	return true
}

func (node *Node) areInputsReadyToRun(nodes *Nodes) bool {

	for _, in := range node.Inputs {
		n := nodes.FindNode(in.Node)
		if n == nil || n.running {
			return false
		}
	}

	return true
}

func (node *Node) areInputsDone(nodes *Nodes) bool {
	for _, in := range node.Inputs {
		n := nodes.FindNode(in.Node)
		if n == nil || !n.done {
			return false
		}
	}
	return true
}

func (node *Node) isInputsChanged(nodes *Nodes) bool {
	for _, in := range node.Inputs {
		n := nodes.FindNode(in.Node)
		if n == nil || n.changed {
			return true
		}
	}
	return false
}

type Nodes struct {
	nodes []*Node
	fns   []*NodeFnDef

	disk *Disk
	dbs  []*DiskDb

	interrupt atomic.Bool
}

func NewNodes(path string) (*Nodes, error) {
	var nodes Nodes

	var err error
	nodes.disk, err = NewDisk("databases")
	if err != nil {
		return nil, fmt.Errorf("NewDisk() failed: %w", err)
	}

	js, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal([]byte(js), &nodes.nodes)
		if err != nil {
			fmt.Printf("Unmarshal(%s) failed: %v\n", path, err)
		}
	}

	//check ins/outs ranges & all inputs are set
	//...

	//add basic nodes
	nodes.AddFunc(NodeFile_init())
	nodes.AddFunc(NodeMerge_init())
	nodes.AddFunc(NodeSelect_init())

	return &nodes, nil
}

func (nodes *Nodes) Destroy() {

	for _, db := range nodes.dbs {
		db.Destroy()
	}

	for _, n := range nodes.nodes {
		n.Destroy()
	}
}

func (nodes *Nodes) Save(path string) error {
	if path == "" {
		return nil
	}

	js, err := json.MarshalIndent(&nodes.nodes, "", "")
	if err != nil {
		return fmt.Errorf("MarshalIndent() failed: %w", err)
	}

	err = os.WriteFile(path, js, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile() failed: %w", err)
	}

	return nil
}

func (nodes *Nodes) GetNodesRangeCoord() OsV4 {
	//...
	return OsV4{}
}

func (nodes *Nodes) getUniqueName(name string) string {

	if nodes.FindNode(name) == nil {
		return name
	}

	i := 1
	for {
		nm := fmt.Sprintf("%s_%d", name, i)
		if nodes.FindNode(nm) == nil {
			return nm
		}
		i++
	}
}

func (nodes *Nodes) AddNode(name string, fnDef *NodeFnDef) *Node {

	if name == "" {
		name = nodes.getUniqueName(fnDef.name)
	}

	n := nodes.FindNode(name)
	if n != nil {
		return nil //already exist
	}

	n = NewNode(name, fnDef.name)
	nodes.nodes = append(nodes.nodes, n)
	return n
}

func (nodes *Nodes) RemoveNode(name string) bool {

	found := false
	for i, n := range nodes.nodes {
		if n.Name == name {
			nodes.nodes = append(nodes.nodes[:i], nodes.nodes[i+1:]...)
			found = true
			break
		}
	}

	for _, n := range nodes.nodes {
		n.RemoveInputs(name)
	}

	return found
}

func (nodes *Nodes) RenameNode(src, dst string) bool {

	n := nodes.FindNode(dst)
	if n != nil {
		return false //newName already exist
	}

	found := false
	for _, n := range nodes.nodes {
		if n.Name == src {
			n.Name = dst
		}
		n.RenameInputs(src, dst)
	}

	return found
}

func (nodes *Nodes) AddFunc(fn *NodeFnDef) *NodeFnDef {
	nodes.fns = append(nodes.fns, fn)
	return fn
}

func (nodes *Nodes) FindFn(name string) *NodeFnDef {
	for _, fn := range nodes.fns {
		if fn.name == name {
			return fn
		}
	}
	return nil
}
func (nodes *Nodes) FindNode(name string) *Node {
	for _, n := range nodes.nodes {
		if n.Name == name {
			return n
		}
	}
	return nil
}

func (nodes *Nodes) IsRunning() bool {
	return !nodes.interrupt.Load()
}

func (nodes *Nodes) Maintenance() {

}

func (nodes *Nodes) Execute() {

	//prepare
	for _, n := range nodes.nodes {
		n.done = false
		n.running = false
	}

	//multi-thread executing
	var numActiveThreads atomic.Int64
	maxActiveThreads := int64(runtime.NumCPU())
	var wg sync.WaitGroup
	run := true
	for run && nodes.IsRunning() {
		run = false
		for _, n := range nodes.nodes {

			if !n.done {
				run = true

				if !n.running {
					if n.areInputsErrorFree(nodes) {
						if n.areInputsReadyToRun(nodes) {
							if n.areInputsDone(nodes) {

								if n.changed || n.isInputsChanged(nodes) || n.err != nil {

									//maximum concurent threads
									if numActiveThreads.Load() >= maxActiveThreads {
										time.Sleep(10 * time.Millisecond)
									}

									//run it
									n.running = true
									wg.Add(1)
									go func(nn *Node) {
										numActiveThreads.Add(1)
										nn.Execute(nodes)
										wg.Done()
										numActiveThreads.Add(-1)
									}(n)
								} else {
									n.done = true
								}
							}
						}
					} else {
						n.done = true
					}
				}
			}
		}
	}

	wg.Wait()

	if nodes.IsRunning() {
		for _, n := range nodes.nodes {
			n.changed = false
		}

		nodes.Maintenance()
	}
}

/*func NodesTest() {

	nds, err := NewNodes("")
	if err != nil {
		return
	}
	defer nds.Destroy("")

	nA := nds.AddNode("a", fOpen)
	nA.SetParam("path", "7gui.sqlite")
	nB := nds.AddNode("b", fSelect)
	nB.SetParam("query", "SELECT label FROM a.__skyalt__")
	nB.AddInput(nA, 0)
	nC := nds.AddNode("c", fSelect)
	nC.SetParam("query", "SELECT COUNT(*) FROM result")
	nC.AddInput(nB, 0)

	nds.Execute()
	nB.db.Print()
	nC.db.Print()

	nB.SetParam("query", "SELECT label FROM a.__skyalt__")

	nds.Execute()
}*/

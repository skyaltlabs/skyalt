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
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	parent *Node

	FnName   string
	Id       int
	Bypass   bool
	Pos      OsV2f
	Selected bool

	Subs    []*Node
	Attrs   []*NodeParamOut
	Inputs  []*NodeParamIn
	outputs []*NodeParamOut

	pos_start      OsV2f
	selected_cover bool

	changed bool
	running bool
	done    bool

	err error
}

func NewNode(parent *Node, fnName string) *Node {
	var node Node
	node.parent = parent
	node.FnName = fnName

	if parent != nil {
		node.Id = parent.getUniqueId()
	} else {
		node.Id = 1
	}
	node.changed = true

	return &node
}

func (node *Node) Destroy() {
	for _, n := range node.Subs {
		n.Destroy()
	}
}

func (node *Node) getPath() string {

	var path string

	if node.parent != nil {
		path += node.parent.getPath()
	}

	path += node.GetAttr("name").Value + "/"

	return path
}

func (node *Node) IsGuiSub() bool {
	return node.FnName == "gui_sub"
}

func (a *Node) FindMirror(b *Node, b_act *Node) *Node {

	if b == b_act {
		return a
	}
	for i, na := range a.Subs {
		ret := na.FindMirror(b.Subs[i], b_act)
		if ret != nil {
			return ret
		}
	}
	return nil
}

func (src *Node) Copy() (*Node, error) {

	js, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	dst := &Node{}
	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, err
	}

	dst.UpdateParents(nil)

	return dst, nil
}

func (a *Node) Cmp(b *Node) bool {

	if a.Id != b.Id || a.Bypass != b.Bypass || a.Pos.X != b.Pos.X || a.Pos.Y != b.Pos.Y || a.Selected != b.Selected || a.FnName != b.FnName {
		return false
	}

	if len(a.Subs) != len(b.Subs) || len(a.Attrs) != len(b.Attrs) || len(a.Inputs) != len(b.Inputs) {
		return false
	}

	for i, itA := range a.Attrs {
		itB := b.Attrs[i]
		if itA.Name != itB.Name || itA.Value != itB.Value || itA.Gui_type != itB.Gui_type || itA.Gui_options != itB.Gui_options {
			return false
		}
	}

	for i, itA := range a.Inputs {
		itB := b.Inputs[i]

		if itA.Name != itB.Name || itA.Value != itB.Value || itA.Gui_type != itB.Gui_type || itA.Gui_options != itB.Gui_options {
			return false
		}
		if itA.Wire_id != itB.Wire_id || itA.Wire_param != itB.Wire_param {
			return false
		}

	}

	for i, itA := range a.Subs {
		if !itA.Cmp(b.Subs[i]) {
			return false
		}
	}

	return true
}

func (node *Node) UpdateParents(parent *Node) {
	node.parent = parent
	for _, n := range node.Subs {
		n.UpdateParents(node)
	}
}

func (node *Node) getUniqueId() int {

	max := 0
	for _, n := range node.Subs {
		max = OsMax(max, n.Id)
	}
	return max + 1
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

func (node *Node) FindAttr(name string) *NodeParamOut {
	for _, attr := range node.Attrs {
		if attr.Name == name {
			return attr
		}
	}
	return nil
}
func (node *Node) FindInput(name string) *NodeParamIn {
	for _, in := range node.Inputs {
		if in.Name == name {
			return in
		}
	}
	return nil
}
func (node *Node) FindOutput(name string) *NodeParamOut {
	for _, out := range node.outputs {
		if out.Name == name {
			return out
		}
	}
	return nil
}

func (node *Node) GetAttr(name string) *NodeParamOut {
	attr := node.FindAttr(name)
	if attr == nil {
		//add
		attr = &NodeParamOut{Name: name}
		node.Attrs = append(node.Attrs, attr)
	}
	return attr
}

func (node *Node) GetInput(name string) *NodeParamIn {
	in := node.FindInput(name)
	if in == nil {
		//add
		in = &NodeParamIn{Name: name}
		node.Inputs = append(node.Inputs, in)
	}
	return in
}

func (node *Node) GetOutput(name string) *NodeParamOut {
	out := node.FindOutput(name)
	if out == nil {
		//add
		out = &NodeParamOut{Name: name}
		node.outputs = append(node.outputs, out)
	}
	return out
}

func (node *Node) GetInputString(name string) string {
	in := node.GetInput(name)

	_, out := in.FindWireOut(node)
	if out != nil {
		return out.Value
	}

	return in.Value
}
func (node *Node) GetInputInt(name string) int {
	v, _ := strconv.Atoi(node.GetInputString(name))
	return v
}

func (node *Node) FindNode(id int) *Node {
	for _, n := range node.Subs {
		if n.Id == id {
			return n
		}
	}
	return nil
}

func (node *Node) AddNode(fnName string) *Node {
	n := NewNode(node, fnName)
	node.Subs = append(node.Subs, n)
	return n
}

func (node *Node) RemoveNode(id int) bool {

	if id == 1 {
		return false //can't delete main
	}

	found := false
	for i, n := range node.Subs {
		if n.Id == id {
			node.Subs = append(node.Subs[:i], node.Subs[i+1:]...)
			found = true
			break
		}
	}

	for _, n := range node.Subs {
		n.RemoveInputs(id)
	}

	return found
}

func (node *Node) RemoveInputs(id int) {
	for _, in := range node.Inputs {
		if in.Wire_id == id {
			in.Wire_id = 0
		}
	}
}

func (node *Node) Execute(app *NodeView) {
	//function
	/*fn := node.GetFn()
	if fn == nil {
		node.err = fmt.Errorf("Function(%s) not found", node.FnName)
		return
	}*/

	//free previous db
	/*if node.db != nil {
		node.db.Destroy()
		node.db = nil
	}*/

	//update inputs
	/*for _, in := range node.Inputs {
		_, node.err = in.UpdateInput(node)
		if node.err != nil {
			return
		}
	}*/

	//call
	if node.Bypass {
		//copy inputs to outputs
		//node.outputs = make([]NodeData, len(ins))
		//copy(node.outputs, ins)
	} else {

		if node.FnName != "main" {
			nc := app.server.Start(node.FnName)
			if nc != nil {

				//copy inputs ...
				nc.Start()
				//copy outputs ...

				if nc.progress.Error != "" {
					node.err = errors.New(nc.progress.Error)
				}
			} else {
				node.err = fmt.Errorf("can't find node exe(%s)", node.FnName)
			}
		}

		/*node.err = fn.exe(node)
		if node.err != nil {
			fmt.Printf("Node(%d) has error(%v)\n", node.Id, node.err)
		}*/
	}

	if !app.IsRunning() {
		node.err = fmt.Errorf("Interrupted")
		return
	}

	/*if node.db != nil {
		node.db.Commit()
	}*/

	node.changed = true //child can update
	node.done = true
	node.running = false
}

func (node *Node) ExecuteSubs(app *NodeView) {

	//prepare
	for _, n := range node.Subs {
		n.done = false
		n.running = false
	}

	//multi-thread executing
	var numActiveThreads atomic.Int64
	maxActiveThreads := int64(runtime.NumCPU())
	var wg sync.WaitGroup
	run := true
	for run && app.IsRunning() {
		run = false
		for _, n := range node.Subs {

			if !n.done {
				run = true

				if !n.running {
					if n.areInputsErrorFree() {
						if n.areInputsReadyToRun() {
							if n.areInputsDone() {

								if n.changed || n.isInputsChanged() || n.err != nil {

									//maximum concurent threads
									if numActiveThreads.Load() >= maxActiveThreads {
										time.Sleep(10 * time.Millisecond)
									}

									//run it
									n.running = true
									wg.Add(1)
									go func(nn *Node) {
										numActiveThreads.Add(1)
										nn.Execute(app)
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

	if app.IsRunning() {
		for _, n := range node.Subs {
			n.changed = false
		}

		//app.Maintenance()
	}
}

func (node *Node) areInputsErrorFree() bool {

	for _, in := range node.Inputs {
		n := in.FindWireNode(node.parent)
		if n != nil {
			if n.err != nil {
				n.err = fmt.Errorf("incomming error from input(%s)", in.Name)
				return false
			}
		}
	}

	return true
}

func (node *Node) areInputsReadyToRun() bool {

	for _, in := range node.Inputs {
		n := in.FindWireNode(node.parent)
		if n != nil && n.running {
			return false //still running
		}
	}

	return true
}

func (node *Node) areInputsDone() bool {

	for _, in := range node.Inputs {
		n := in.FindWireNode(node.parent)
		if n != nil && !n.done {
			return false //not finished
		}
	}

	return true
}

func (node *Node) isInputsChanged() bool {

	for _, in := range node.Inputs {
		n := in.FindWireNode(node.parent)
		if n != nil && n.changed {
			return true //changed
		}
	}

	return false
}

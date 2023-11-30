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

type NodeFnDefIO struct {
	name string
	json bool
}
type NodeFnDef struct {
	name       string
	init       func(node *Node)
	parameters func(node *Node, ui *Ui)
	exe        func(inputs []NodeData, node *Node) ([]NodeData, error)
	render     func(node *Node, ui *Ui)

	ins          []NodeFnDefIO
	outs         []NodeFnDefIO
	infinite_ins bool
	//infinite_outs bool
}

func NewNodeFnDef(name string,
	initPtr func(node *Node),
	parametersPtr func(node *Node, ui *Ui),
	exePtr func(inputs []NodeData, node *Node) ([]NodeData, error),
	renderPtr func(node *Node, ui *Ui)) *NodeFnDef {
	var fn NodeFnDef
	fn.name = name
	fn.init = initPtr
	fn.parameters = parametersPtr
	fn.exe = exePtr
	fn.render = renderPtr
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

type NodeApp struct {
	root Node

	fns []*NodeFnDef

	disk *Disk
	dbs  []*DiskDb

	interrupt atomic.Bool
}

func NewNodeApp(path string) (*NodeApp, error) {
	var app NodeApp

	var err error
	app.disk, err = NewDisk("databases")
	if err != nil {
		return nil, fmt.Errorf("NewDisk() failed: %w", err)
	}

	app.root = *NewNode(nil, &app, "root", NodeNet_build().name)
	NodeSub_init(&app.root)

	js, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal([]byte(js), &app.root)
		if err != nil {
			fmt.Printf("Unmarshal(%s) failed: %v\n", path, err)
		}
	}

	app.root.SetParentAndApp(nil, &app)

	//check ins/outs ranges & all inputs are set
	//...

	//add basic nodes
	app.AddFunc(NodeFile_build())
	app.AddFunc(NodeMerge_build())
	app.AddFunc(NodeSelect_build())
	app.AddFunc(NodeNet_build())
	app.AddFunc(NodeOutText_build())

	return &app, nil
}

func (app *NodeApp) Destroy() {

	for _, db := range app.dbs {
		db.Destroy()
	}

	app.root.Destroy()
}

func (app *NodeApp) Save(path string) error {
	if path == "" {
		return nil
	}

	js, err := json.MarshalIndent(&app.root, "", "")
	if err != nil {
		return fmt.Errorf("MarshalIndent() failed: %w", err)
	}

	err = os.WriteFile(path, js, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile() failed: %w", err)
	}

	return nil
}

/*func (app *NodeApp) GetNodesRangeCoord() OsV4 {
	//...
	return OsV4{}
}*/

func (app *NodeApp) AddFunc(fn *NodeFnDef) *NodeFnDef {

	fnn := app.FindFn(fn.name)
	if fnn != nil {
		fmt.Printf("Error: Func(%s) already added\n", fn.name)
		//panic? ...
	}

	app.fns = append(app.fns, fn)
	return fn
}

func (app *NodeApp) FindFn(name string) *NodeFnDef {
	for _, fn := range app.fns {
		if fn.name == name {
			return fn
		}
	}
	return nil
}

func (app *NodeApp) IsRunning() bool {
	return !app.interrupt.Load()
}

/*func (node *Node) prepareToExecute() {

	node.done = false
	node.running = false

	for _, n := range node.Subs {
		n.prepareToExecute()
	}
}*/

func (node *Node) ExecuteSubs() {

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
	for run && node.app.IsRunning() {
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
										nn.Execute()
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

	if node.app.IsRunning() {
		for _, n := range node.Subs {
			n.changed = false
		}

		//app.Maintenance()
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

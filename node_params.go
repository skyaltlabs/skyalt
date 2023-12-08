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
)

type NodeParamOut struct {
	node *Node

	Name        string
	Value       string //interface{}
	Gui_type    string `json:",omitempty"`
	Gui_options string `json:",omitempty"`

	coordDot   OsV4 //px on screen
	coordLabel OsV4 //px on screen
}

func (out *NodeParamOut) Marshal(st any) error {
	js, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("Output(%s) Marshal() failed: %w", out.Name, err)
	}
	out.Value = string(js)
	return nil
}

type NodeParamIn struct {
	node *Node

	Name        string
	Value       string
	Gui_type    string `json:",omitempty"`
	Gui_options string `json:",omitempty"`

	Wire_id    int    `json:",omitempty"`
	Wire_param string `json:",omitempty"`

	coordDot   OsV4 //px on screen
	coordLabel OsV4 //px on screen
}

func (in *NodeParamIn) Unmarshal(st any) error {
	err := json.Unmarshal([]byte(in.Value), st)
	if err != nil {
		return fmt.Errorf("Input(%s) Unmarshal() failed: %w", in.Name, err)
	}
	return nil
}

func (in *NodeParamIn) SetWire(paramOut *NodeParamOut) {

	if paramOut == nil {
		in.Wire_id = 0
		in.Wire_param = ""
	} else {
		in.Wire_id = paramOut.node.Id
		in.Wire_param = paramOut.Name
	}

	in.node.SetChanged()

}

func (in *NodeParamIn) String() (string, error) {
	type Value struct {
		Value string
	}
	var v Value
	err := in.Unmarshal(&v)
	return v.Value, err
}
func (in *NodeParamIn) Int() (int, error) {
	type Value struct {
		Value int
	}
	var v Value
	err := in.Unmarshal(&v)
	return v.Value, err
}

/*func (in *NodeParamIn) FindWireNode(node *Node) *Node {

	if node == nil {
		return nil
	}

	if in.Wire_id <= 0 {
		return nil
	}
	n := node.parent.FindNode(in.Wire_id)
	if n == nil {
		in.SetWire(nil, nil, nil) //reset
	}
	return n
}*/

func (in *NodeParamIn) FindWireOut() *NodeParamOut {

	outNode := in.node.parent.FindNode(in.Wire_id)
	if outNode == nil {
		return nil
	}

	out := outNode.FindAttr(in.Wire_param) //attr
	if out == nil {
		out = outNode.FindOutput(in.Wire_param) //out
	}

	if out == nil {
		in.SetWire(nil) //reset
	}

	return out
}
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

func SAExe_Vars(node *SANode) bool {
	//nothing here
	return true
}

// TODO: render sub nodes in same level as this node ...
// - the node can be large(area) so reorder() must count with that
func SAExe_For(node *SANode) bool {
	inputAttr := node.GetAttr("input", []byte("[]"))
	isInputNumber := inputAttr.GetResult().IsNumber()

	_keyAttr := node.GetAttr("_key", "")
	_valueAttr := node.GetAttr("_value", "")

	var list []*SANode
	node.buildSubList(&list)
	node.markUnusedAttrs()

	if isInputNumber {
		n := inputAttr.GetInt()
		for i := 0; i < n; i++ {
			_keyAttr.GetResult().SetInt(i)
			_valueAttr.GetResult().SetInt(i)
			node.app.ExecuteList(list)
		}
	} else {
		nArr := inputAttr.NumArrayItems()
		nMap := inputAttr.NumMapItems()

		if nArr > 0 {
			for i := 0; i < nArr; i++ {
				_keyAttr.GetResult().SetInt(i)
				_valueAttr.GetResult().value = inputAttr.GetArrayItem(i)

				node.app.ExecuteList(list)
			}
		}
		if nMap > 0 {
			for i := 0; i < nArr; i++ {
				key, val := inputAttr.GetMapItem(i)
				_keyAttr.GetResult().SetString(key)
				_valueAttr.GetResult().value = val

				node.app.ExecuteList(list)
			}
		}
	}

	return true
}

func SAExe_Setter_destNode(node *SANode) *SANode {
	nodeAttr := node.GetAttr("node", "")
	attrAttr := node.GetAttr("attr", "")
	nd := node.parent.FindNode(nodeAttr.GetString())
	if nd == nil {
		return nil
	}
	attr := nd.findAttr(attrAttr.GetString())
	if attr == nil {
		return nil //both(node &  attr) must be valid
	}
	return nd
}

func SAExe_Setter(node *SANode) bool {
	triggerAttr := node.GetAttrUi("trigger", 0, SAAttrUi_SWITCH)

	nodeAttr := node.GetAttr("node", "")
	attrAttr := node.GetAttr("attr", "")
	value := node.GetAttr("value", "").GetString()

	nd := node.parent.FindNode(nodeAttr.GetString())
	if nd == nil {
		nodeAttr.SetErrorStr("Not exist")
		return false
	}

	attr := nd.findAttr(attrAttr.GetString())
	if attr == nil {
		attrAttr.SetErrorStr("Not exist")
		return false
	}

	if triggerAttr.GetBool() {
		attr.AddSetAttr(value)
		triggerAttr.AddSetAttr("0")
	}

	return true
}

func SAExe_Setter_renameNode(node *SANode, oldName string, newName string) {
	nodeAttr := node.GetAttr("node", "")
	if nodeAttr.GetString() == oldName {
		nodeAttr.SetExpString(newName, false)
	}
}

func SAExe_Setter_renameAttr(node *SANode, nodeName string, oldAttrName string, newAttrName string) {
	nodeAttr := node.GetAttr("node", "")
	attrAttr := node.GetAttr("attr", "")
	if nodeAttr.GetString() == nodeName && attrAttr.GetString() == oldAttrName {
		attrAttr.SetExpString(newAttrName, false)
	}
}

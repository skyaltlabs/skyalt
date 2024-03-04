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
	"sort"
)

type SANodeColRow struct {
	Pos              int
	Min, Max, Resize float64 `json:",omitempty"`
	ResizeName       string  `json:",omitempty"`
}

func (a *SANodeColRow) Cmp(b *SANodeColRow) bool {
	return a.Min == b.Min && a.Max == b.Max && a.Resize == b.Resize && a.ResizeName == b.ResizeName
}

func SANodeColRow_Check(items *[]*SANodeColRow) {
	//ranges
	for _, c := range *items {
		c.Pos = OsMax(c.Pos, 0)
		c.Min = OsMaxFloat(c.Min, 0)
		c.Max = OsMaxFloat(c.Max, 0)
		c.Resize = OsMaxFloat(c.Resize, 0)
	}

	//sort
	sort.Slice(*items, func(i, j int) bool {
		return (*items)[i].Pos < (*items)[j].Pos
	})

	//remove same poses
	for i := len(*items) - 1; i > 0; i-- {
		if (*items)[i].Pos == (*items)[i-1].Pos {
			*items = append((*items)[:i], (*items)[i+1:]...)
		}
	}
}

func SANodeColRow_Insert(items *[]*SANodeColRow, src *SANodeColRow, pos int, shift bool) {
	if src == nil {
		src = &SANodeColRow{Pos: pos, Min: 1, Max: 1, Resize: 1}
	}
	src.Pos = pos

	//move items afer pos
	if shift {
		for _, c := range *items {
			if c.Pos >= pos {
				c.Pos++
			}
		}
	}

	*items = append(*items, src)
}
func SANodeColRow_Remove(items *[]*SANodeColRow, pos int) {

	remove_i := -1

	//move items afer pos
	for i, c := range *items {
		if c.Pos == pos {
			remove_i = i
		}
		if c.Pos >= pos {
			c.Pos--
		}
	}

	//remove
	if remove_i >= 0 {
		*items = append((*items)[:remove_i], (*items)[remove_i+1:]...)
	}
}

func SANodeColRow_Find(items *[]*SANodeColRow, pos int) *SANodeColRow {
	for _, c := range *items {
		if c.Pos == pos {
			return c
		}
	}
	return nil
}
func SANodeColRow_GetMaxPos(items *[]*SANodeColRow) int {
	mx := 0
	for _, c := range *items {
		mx = OsMax(mx, c.Pos)
	}
	return mx
}

type SANodePath struct {
	names []string
}

func (path *SANodePath) _getPath(w *SANode) {
	if w.parent != nil {
		path._getPath(w.parent)

		//inside If, because don't add root name
		path.names = append(path.names, w.Name)
	}
}
func NewSANodePath(w *SANode) SANodePath {
	var path SANodePath
	if w != nil {
		path._getPath(w)
	}
	return path
}
func (path *SANodePath) Is() bool {
	return len(path.names) > 0
}
func (a SANodePath) Cmp(b SANodePath) bool {
	if len(a.names) != len(b.names) {
		return false
	}
	for i, nmA := range a.names {
		if nmA != b.names[i] {
			return false
		}
	}

	return true
}
func (path *SANodePath) FindPath(root *SANode) *SANode {
	node := root
	for _, nm := range path.names {
		node = node.FindNode(nm)
		if node == nil {
			return nil
		}
	}
	return node
}

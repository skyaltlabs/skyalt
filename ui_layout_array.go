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

type UiLayoutArrayItem struct {
	min    float32
	max    float32
	resize *UiLayoutArrayRes
}

type UiLayoutArrayRes struct {
	value float32
	name  string
}

type UiLayoutArray struct {
	resizes []*UiLayoutArrayRes //backups

	inputs  []UiLayoutArrayItem
	outputs []int32
}

func (dst *UiLayoutArray) CopySub(src *UiLayoutArray, src_start int, src_end int, cell int) {
	dst.Clear()

	rs := float32(LayoutArray_resizerSize(cell)) / 2 / float32(cell)

	last_resize := false
	src_end = OsMin(src_end, len(src.outputs))
	for i := src_start; i < src_end; i++ {

		dst.findOrAdd(i).min = float32(src.outputs[i]) / float32(cell)

		isResize := (src.inputs[i].resize != nil)
		if isResize && !last_resize {
			dst.findOrAdd(i).min -= rs
		}
		if !isResize && last_resize {
			dst.findOrAdd(i).min += rs
		}

		last_resize = isResize
	}
}

func (arr *UiLayoutArray) Clear() {
	arr.inputs = arr.inputs[:0]
}

func (arr *UiLayoutArray) NumInputs() int {
	return len(arr.inputs)
}

func (arr *UiLayoutArray) Resize(num int) {
	//huge amount of RAM if I set few items on position 1000 - all before are alocated and set & reset after frame is rendered ...
	for i := arr.NumInputs(); i < num; i++ {
		arr.inputs = append(arr.inputs, UiLayoutArrayItem{min: 1, max: 0, resize: nil})
	}
}

func LayoutArray_resizerSize(cell int) int {
	v := cell / 4
	if v < 9 {
		return 9
	}
	return v
}

func (arr *UiLayoutArray) IsLastResizeValid() bool {
	n := arr.NumInputs()
	return n >= 2 && arr.inputs[n-2].resize == nil && arr.inputs[n-1].resize != nil
}

func (arr *UiLayoutArray) GetResizeIndex(i int) int {

	if arr.IsLastResizeValid() {
		if i+2 == arr.NumInputs() {
			if arr.inputs[i+1].resize != nil {
				return i + 1 // show resizer before column/row
			}
			return -1
		}
		if i+1 == arr.NumInputs() {
			return -1 // last was return as Previous, so no last
		}
	}

	if i < arr.NumInputs() {
		if arr.inputs[i].resize != nil {
			return i
		}
	}
	return -1
}

func (arr *UiLayoutArray) Convert(cell int, start int, end int) OsV2 {

	var ret OsV2

	for i := 0; i < end; i++ {
		ok := (i < len(arr.outputs))

		if i < start {
			if ok {
				ret.X += int(arr.outputs[i])
			} else {
				ret.X += cell
			}
		} else {
			if ok {
				ret.Y += int(arr.outputs[i])
			} else {
				ret.Y += cell
			}
		}
	}

	if end > 0 && (end-1 < arr.NumInputs()) && arr.GetResizeIndex(int(end)-1) >= 0 {
		ret.Y -= LayoutArray_resizerSize(cell)
	}

	return ret
}

func (arr *UiLayoutArray) ConvertMax(cell int, start int, end int) OsV2 {
	var ret OsV2

	for i := 0; i < end; i++ {
		ok := (i < arr.NumInputs())

		if i < start {
			if ok {
				ret.X += int(arr.inputs[i].max * float32(cell))
			} else {
				ret.X += cell
			}
		} else {
			if ok {
				ret.Y += int(arr.inputs[i].max * float32(cell))
			} else {
				ret.Y += cell
			}
		}
	}

	return ret
}

func (arr *UiLayoutArray) GetCloseCell(pos int) int {
	if pos < 0 {
		return -1
	}
	allPixels := 0
	allPixelsLast := 0
	for i := 0; i < len(arr.outputs); i++ {
		allPixels += int(arr.outputs[i])

		if pos >= allPixelsLast && pos < allPixels {
			return i //found
		}

		allPixelsLast = allPixels
	}

	return len(arr.outputs)
}

func (arr *UiLayoutArray) GetResizerPos(i int, cell int) int {
	if i >= len(arr.outputs) {
		return 0
	}

	allPixels := 0
	for ii := 0; ii <= i; ii++ {
		allPixels += int(arr.outputs[ii])
	}

	return allPixels - LayoutArray_resizerSize(cell)
}

func (arr *UiLayoutArray) IsResizerTouch(touchPos int, cell int) int {
	space := LayoutArray_resizerSize(cell)

	for i := 0; i < arr.NumInputs(); i++ {
		if arr.GetResizeIndex(i) >= 0 {
			p := arr.GetResizerPos(i, cell)
			if touchPos > p && touchPos < p+space {
				return i
			}
		}
	}
	return -1
}

func (arr *UiLayoutArray) OutputAll() int {
	sum := 0
	for i := 0; i < len(arr.outputs); i++ {
		sum += int(arr.outputs[i])
	}
	return sum
}

func (arr *UiLayoutArray) Update(cell int, window int) {

	arr.outputs = make([]int32, arr.NumInputs())

	//project in -> out
	for i := 0; i < len(arr.inputs); i++ {
		//min
		minV := float64(arr.inputs[i].min)
		minV = OsClampFloat(minV, 0.001, 100000000)

		if arr.inputs[i].resize == nil {
			// max
			maxV := minV
			if arr.inputs[i].max > 0 {
				maxV = float64(arr.inputs[i].max)
				maxV = OsMaxFloat(minV, maxV)
			}

			arr.inputs[i].min = float32(minV)
			arr.inputs[i].max = float32(maxV)
		} else {

			resV := float64(arr.inputs[i].resize.value)
			resV = OsMaxFloat(resV, minV)

			if arr.inputs[i].max > 0 {
				maxV := float64(arr.inputs[i].max)
				maxV = OsMaxFloat(minV, maxV)

				resV = OsClampFloat(resV, minV, maxV)
			}

			arr.inputs[i].min = float32(resV)
			arr.inputs[i].max = float32(resV)
			arr.inputs[i].resize.value = float32(resV)
		}
	}

	//sum
	allPixels := 0
	for i := 0; i < len(arr.outputs); i++ {
		arr.outputs[i] = int32(arr.inputs[i].min * float32(cell))
		allPixels += int(arr.outputs[i])
	}

	// make it larger(if maxes allow)
	hasSpace := (len(arr.outputs) > 0)
	for allPixels < window && hasSpace {
		rest := window - allPixels
		tryAdd := OsMax(1, rest/int(len(arr.outputs)))

		hasSpace = false
		for i := 0; i < len(arr.outputs) && allPixels < window; i++ {

			maxAdd := int(arr.inputs[i].max*float32(cell)) - int(arr.outputs[i])
			add := OsClamp(tryAdd, 0, maxAdd)

			arr.outputs[i] += int32(add)
			allPixels += add

			if maxAdd > tryAdd {
				hasSpace = true
			}
		}
	}
}

func (arr *UiLayoutArray) HasResize() bool {
	for _, c := range arr.inputs {
		if c.resize != nil {
			return true
		}
	}
	return false
}

func (arr *UiLayoutArray) FindOrAddResize(name string) (*UiLayoutArrayRes, bool) {

	//find
	for _, it := range arr.resizes {
		if it.name == name {
			return it, true
		}
	}

	//add
	it := &UiLayoutArrayRes{name: name, value: 1}
	arr.resizes = append(arr.resizes, it)
	return it, false
}

func (arr *UiLayoutArray) GetOutput(i int) int {

	if i < len(arr.outputs) {
		return int(arr.outputs[i])
	}
	return -1
}

func (arr *UiLayoutArray) findOrAdd(pos int) *UiLayoutArrayItem {

	if pos >= len(arr.inputs) {
		arr.Resize(pos + 1)
	}

	return &arr.inputs[pos]
}

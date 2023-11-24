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

type UiLayout struct {
	cols UiLayoutArray
	rows UiLayoutArray

	scrollV        UiLayoutScroll
	scrollH        UiLayoutScroll
	scrollOnScreen bool //show scroll all the time

	//maybe have them one time in Root like a 'over* Layout', 'overScroll* Layout', etc. - in LayoutTouch ...
	over          bool
	overScroll    bool
	touch_inside  bool
	touch_active  bool
	touch_end     bool
	touch_enabled bool

	app  *UiLayoutApp
	hash uint64
}

func (lay *UiLayout) Init(hash uint64, app *UiLayoutApp) {

	lay.app = app
	lay.hash = hash
	lay.touch_enabled = true

	lay.scrollV.Init()
	lay.scrollH.Init()

	it := lay.app.FindGlobalScrollHash(hash)
	if it != nil {

		lay.scrollV.wheel = it.ScrollVpos
		lay.scrollH.wheel = it.ScrollHpos

		for _, rs := range it.Cols_resize {
			res, _ := lay.cols.FindOrAddResize(rs.Name)
			res.value = float32(rs.Value)
		}

		for _, rs := range it.Rows_resize {
			res, _ := lay.rows.FindOrAddResize(rs.Name)
			res.value = float32(rs.Value)
		}
	}
}

func (lay *UiLayout) Save() {

	hasColResize := lay.cols.HasResize()
	hasRowResize := lay.rows.HasResize()

	// save scroll into Rec
	if lay.scrollV.wheel != 0 || lay.scrollH.wheel != 0 || hasColResize || hasRowResize {
		it := lay.app.AddGlobalScrollHash(lay.hash)

		it.ScrollVpos = 0
		it.ScrollHpos = 0
		it.Cols_resize = nil
		it.Rows_resize = nil

		if lay.scrollV.wheel != 0 {
			it.ScrollVpos = lay.scrollV.wheel
		}
		if lay.scrollH.wheel != 0 {
			it.ScrollHpos = lay.scrollH.wheel
		}

		if hasColResize {
			for _, c := range lay.cols.resizes {
				it.Cols_resize = append(it.Cols_resize, UiLayoutAppItemResize{Name: c.name, Value: c.value})
			}
		}

		if hasRowResize {
			for _, r := range lay.rows.resizes {
				it.Rows_resize = append(it.Rows_resize, UiLayoutAppItemResize{Name: r.name, Value: r.value})
			}
		}

	} else {
		sc := lay.app.FindGlobalScrollHash(lay.hash)
		if sc != nil {
			*sc = UiLayoutAppItem{}
		}
	}
}

func (lay *UiLayout) Close() {
	lay.Save()
}

func (lay *UiLayout) Reset() {
	lay.cols.Clear()
	lay.rows.Clear()
}

func (lay *UiLayout) UpdateArray(cell int, window OsV2, endGrid OsV2) {

	if endGrid.X > lay.cols.NumInputs() {
		lay.cols.Resize(int(endGrid.X))
	}
	if endGrid.Y > lay.rows.NumInputs() {
		lay.rows.Resize(int(endGrid.Y))
	}
	lay.cols.Update(cell, window.X)
	lay.rows.Update(cell, window.Y)
}

func (lay *UiLayout) Convert(cell int, in OsV4) OsV4 {

	c := lay.cols.Convert(cell, in.Start.X, in.Start.X+in.Size.X)
	r := lay.rows.Convert(cell, in.Start.Y, in.Start.Y+in.Size.Y)

	return OsV4{OsV2{c.X, r.X}, OsV2{c.Y, r.Y}}
}

func (lay *UiLayout) ConvertMax(cell int, in OsV4) OsV4 {
	c := lay.cols.ConvertMax(cell, in.Start.X, in.Start.X+in.Size.X)
	r := lay.rows.ConvertMax(cell, in.Start.Y, in.Start.Y+in.Size.Y)

	return OsV4{OsV2{c.X, r.X}, OsV2{c.Y, r.Y}}
}

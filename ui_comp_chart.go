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
	"fmt"
	"strconv"
)

type UiLayoutChartItem struct {
	X, Y  float64
	Label string
}

func UiLayoutChart_getBound(items []UiLayoutChartItem) (UiLayoutChartItem, UiLayoutChartItem) {

	var min, max UiLayoutChartItem
	//	if len(items) > 0 {
	//		min = items[0]
	//		max = items[0]
	//	}
	for _, it := range items {
		min.X = OsMinFloat(min.X, it.X)
		min.Y = OsMinFloat(min.Y, it.Y)
		max.X = OsMaxFloat(max.X, it.X)
		max.Y = OsMaxFloat(max.Y, it.Y)
	}
	return min, max
}

func cellMargin(cell_left, cell_right, cell_top, cell_bottom float64, x, y, w, h float64, ui *Ui) (float64, float64, float64, float64) {

	canvas := ui.GetCall().call.canvas

	l := float64(ui.CellWidth(cell_left)) / float64(canvas.Size.X)
	r := float64(ui.CellWidth(cell_right)) / float64(canvas.Size.X)
	t := float64(ui.CellWidth(cell_top)) / float64(canvas.Size.Y)
	b := float64(ui.CellWidth(cell_bottom)) / float64(canvas.Size.Y)

	x = l + ((1 - l - r) * x)
	y = t + ((1 - t - b) * y)
	w = ((1 - l - r) * w)
	h = ((1 - t - b) * h)

	return x, y, w, h
}

func _UiLayoutChart_drawAxisX(min, max UiLayoutChartItem, left_margin, right_margin, top_margin, bottom_margin float64, cdAxis, cdAxisGrey OsCd, ui *Ui, drawAxis, drawValues bool) {
	vx := max.X - min.X
	vy := max.Y - min.Y

	if drawAxis {
		//0
		y := (0 - min.Y) / vy
		x, y, w, h := cellMargin(left_margin, right_margin, 0, bottom_margin, 0, 1-y, 1, 0, ui)
		ui.Paint_line(0, 0, 1, 1, 0, x, y, x+w, y+h, cdAxis, 0.03)
	}

	if drawValues {
		top_margin2 := float64(ui.GetCall().call.canvas.Size.Y)/float64(ui.win.Cell()) - bottom_margin
		cx := float64(ui.CellWidth(2)) / float64(ui.GetCall().call.canvas.Size.X)

		x, y, w, h := cellMargin(left_margin, right_margin, top_margin, bottom_margin, 0, 0, 0, 1, ui)
		ui.Paint_line(0, 0, 1, 1, 0, x, y, x+w, y+h, cdAxisGrey, 0.03)

		p := 0.0
		for {
			x, y, w, h := cellMargin(left_margin, right_margin, top_margin, bottom_margin, p, 0, 0, 1, ui)
			ui.Paint_line(0, 0, 1, 1, 0, x, y, x+w, y+h, cdAxisGrey, 0.03)

			x, y, w, h = cellMargin(left_margin, right_margin, top_margin2, 0, p-cx/2, 0, cx, 1, ui)
			ui.Div_startEx(0, 0, 1, 1, x, y, w, h, fmt.Sprintf("axis_x%f", p))
			ui.Div_colMax(0, 100)
			ui.Div_row(0, bottom_margin)
			ui.GetCall().call.data.scrollV.show = false
			ui.Comp_text(0, 0, 1, 1, strconv.FormatFloat(min.X+(vx*p), 'f', 2, 64), 1)
			//ui.Paint_rect(0, 0, 1, 1, 0, InitOsCdBlack(), 0.03)
			ui.Div_end()

			if p == 1 {
				break
			}

			p += cx
			if p > 1 {
				p = 1
			}
		}
	}
}

func _UiLayoutChart_drawAxisY(min, max UiLayoutChartItem, left_margin, right_margin, top_margin, bottom_margin float64, cdAxis, cdAxisGrey OsCd, ui *Ui, drawAxis, drawValues bool) {
	vx := max.X - min.X
	vy := max.Y - min.Y

	if drawAxis {
		//0
		x := (0 - min.X) / vx
		x, y, w, h := cellMargin(left_margin, 0, 0, bottom_margin, x, 0, 0, 1, ui)
		ui.Paint_line(0, 0, 1, 1, 0, x, y, x+w, y+h, cdAxis, 0.03)

	}

	if drawValues {
		right_margin2 := float64(ui.GetCall().call.canvas.Size.X)/float64(ui.win.Cell()) - left_margin
		cy := float64(ui.CellWidth(2)) / float64(ui.GetCall().call.canvas.Size.Y)

		p := 0.0
		for {
			x, y, w, h := cellMargin(left_margin, right_margin, top_margin, bottom_margin, 0, p, 1, 0, ui)
			ui.Paint_line(0, 0, 1, 1, 0, x, y, x+w, y+h, cdAxisGrey, 0.03)

			x, y, w, h = cellMargin(0, right_margin2, top_margin, bottom_margin, 0, p-cy/2, 1, cy, ui)
			ui.Div_startEx(0, 0, 1, 1, x, y, w, h, fmt.Sprintf("axis_y%f", p))
			ui.Div_colMax(0, 100)
			ui.Div_rowMax(0, 100)
			ui.GetCall().call.data.scrollV.show = false
			ui.Comp_text(0, 0, 1, 1, strconv.FormatFloat(max.Y-(vy*p), 'f', 2, 64), 1)
			//ui.Paint_rect(0, 0, 1, 1, 0, InitOsCdBlack(), 0.03)
			ui.Div_end()

			if p == 1 {
				break
			}

			p += cy
			if p > 1 {
				p = 1
			}
		}
	}
}

func _UiLayoutChart_drawLines(items []UiLayoutChartItem, min, max UiLayoutChartItem, left_margin, right_margin, top_margin, bottom_margin float64, cd OsCd, rad float64, lineThick float64, ui *Ui) {
	vx := max.X - min.X
	vy := max.Y - min.Y

	//lines
	last_x := float64(0)
	last_y := float64(0)
	for i, it := range items {
		x := (it.X - min.X) / vx
		y := (it.Y - min.Y) / vy

		x, y, _, _ = cellMargin(left_margin, right_margin, top_margin, bottom_margin, x, 1-y, 0, 0, ui)

		if rad > 0 {
			ui.Paint_circle(0, 0, 1, 1, 0, x, y, rad, cd, 0)
		}
		//ui.Paint_tooltip()	//...
		if i > 0 {
			ui.Paint_line(0, 0, 1, 1, 0, last_x, last_y, x, y, cd, lineThick)
		}

		last_x = x
		last_y = y
	}
}

func _UiLayoutChart_drawColumns(items []UiLayoutChartItem, min, max UiLayoutChartItem, left_margin, right_margin, top_margin, bottom_margin float64, column_margin float64, cd OsCd, ui *Ui) {
	//vx := max.X - min.X
	vy := max.Y - min.Y

	cmarx := float64(ui.CellWidth(column_margin)) / float64(ui.GetCall().call.canvas.Size.X)
	_, _, cmar, _ := cellMargin(left_margin, right_margin, top_margin, bottom_margin, 0, 0, cmarx, 0, ui)

	//values
	y0 := (0 - min.Y) / vy
	jump_x := 1 / float64(len(items))
	for i, it := range items {
		x := float64(i) * jump_x
		y := (it.Y - 0) / vy

		x, y, w, h := cellMargin(left_margin, right_margin, top_margin, bottom_margin, x, 1-y0, jump_x, -y, ui)

		ui.Paint_rect(x+cmar, y, w-2*cmar, h, 0, cd, 0)
	}
}

func _UiLayoutChart_drawAxisXlabels(items []UiLayoutChartItem, left_margin, right_margin, top_margin2, bottom_margin float64, ui *Ui) {

	top_margin2 = float64(ui.GetCall().call.canvas.Size.Y)/float64(ui.win.Cell()) - bottom_margin

	jump_x := 1 / float64(len(items))
	for i, it := range items {
		x := float64(i) * jump_x

		x, y, w, h := cellMargin(left_margin, right_margin, top_margin2, 0, x, 0, jump_x, 1, ui)

		ui.Div_startEx(0, 0, 1, 1, x, y, w, h, fmt.Sprintf("label%d", i))
		ui.Div_colMax(0, 100)
		ui.Div_row(0, bottom_margin)
		ui.GetCall().call.data.scrollV.show = false
		ui.Comp_text(0, 0, 1, 1, it.Label, 1)
		//ui.Paint_rect(0, 0, 1, 1, 0, InitOsCdBlack(), 0.03)
		ui.Div_end()

	}
}

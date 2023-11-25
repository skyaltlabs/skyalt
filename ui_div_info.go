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

import "math"

const (
	SA_DIV_GET_uid             = 0
	SA_DIV_GET_cell            = 1
	SA_DIV_GET_layoutWidth     = 2
	SA_DIV_GET_layoutHeight    = 3
	SA_DIV_GET_screenWidth     = 4
	SA_DIV_GET_screenHeight    = 5
	SA_DIV_GET_layoutStartX    = 6
	SA_DIV_GET_layoutStartY    = 7
	SA_DIV_GET_touch           = 8
	SA_DIV_GET_touchX          = 9
	SA_DIV_GET_touchY          = 10
	SA_DIV_GET_touchOver       = 11
	SA_DIV_GET_touchOverScroll = 12
	SA_DIV_GET_touchInside     = 13
	SA_DIV_GET_touchStart      = 14
	SA_DIV_GET_touchWheel      = 15
	SA_DIV_GET_touchClicks     = 16
	SA_DIV_GET_touchForce      = 17
	SA_DIV_GET_touchActive     = 18
	SA_DIV_GET_touchEnd        = 19
	SA_DIV_GET_touchCol        = 20
	SA_DIV_GET_touchRow        = 21
	SA_DIV_GET_startCol        = 22
	SA_DIV_GET_startRow        = 23
	SA_DIV_GET_endCol          = 24
	SA_DIV_GET_endRow          = 25

	SA_DIV_GET_scrollVpos    = 26
	SA_DIV_GET_scrollHpos    = 27
	SA_DIV_GET_scrollVshow   = 28
	SA_DIV_GET_scrollHshow   = 29
	SA_DIV_GET_scrollVnarrow = 30
	SA_DIV_GET_scrollHnarrow = 31
)

const (
	SA_DIV_SET_touch_enable   = 0
	SA_DIV_SET_scrollOnScreen = 1
	SA_DIV_SET_copyCols       = 2
	SA_DIV_SET_copyRows       = 3
	SA_DIV_SET_attachScrollH  = 4
	SA_DIV_SET_attachScrollV  = 5

	SA_DIV_SET_scrollVpos    = 26
	SA_DIV_SET_scrollHpos    = 27
	SA_DIV_SET_scrollVshow   = 28
	SA_DIV_SET_scrollHshow   = 29
	SA_DIV_SET_scrollVnarrow = 30
	SA_DIV_SET_scrollHnarrow = 31
)

func (levels *Ui) DivInfo_get(cmd uint8, uid float64) float64 {

	lv := levels.GetCall()

	div := lv.call.FindUid(uid)
	if div == nil {
		return -1
	}

	switch cmd {
	case SA_DIV_GET_uid:
		return math.Float64frombits(div.data.hash)

	case SA_DIV_GET_cell:
		return float64(levels.win.Cell())

	case SA_DIV_GET_layoutWidth:
		return float64(div.canvas.Size.X) / float64(levels.win.Cell())
	case SA_DIV_GET_layoutHeight:
		return float64(div.canvas.Size.Y) / float64(levels.win.Cell())

	case SA_DIV_GET_screenWidth:
		return float64(div.crop.Size.X) / float64(levels.win.Cell())
	case SA_DIV_GET_screenHeight:
		return float64(div.crop.Size.Y) / float64(levels.win.Cell())

	case SA_DIV_GET_layoutStartX:
		return float64(div.data.scrollH.GetWheel()) / float64(levels.win.Cell())
		//return float64(div.crop.Start.X-div.canvas.Start.X) / float64(levels.win.Cell())
	case SA_DIV_GET_layoutStartY:
		return float64(div.data.scrollV.GetWheel()) / float64(levels.win.Cell())
		//return float64(div.crop.Start.Y-div.canvas.Start.Y) / float64(levels.win.Cell())

	case SA_DIV_GET_touch:
		return OsTrnFloat(div.enableInput, 1, 0)

	case SA_DIV_GET_touchX:
		rpos := div.GetRelativePos(levels.win.io.touch.pos)
		return float64(rpos.X) / float64(div.canvas.Size.X)
	case SA_DIV_GET_touchY:
		rpos := div.GetRelativePos(levels.win.io.touch.pos)
		return float64(rpos.Y) / float64(div.canvas.Size.Y)

	case SA_DIV_GET_touchOver:
		return OsTrnFloat(div.data.over, 1, 0)

	case SA_DIV_GET_touchOverScroll:
		return OsTrnFloat(div.data.overScroll, 1, 0)

	case SA_DIV_GET_touchInside:
		return OsTrnFloat(div.data.touch_inside, 1, 0)

	case SA_DIV_GET_touchStart:
		if div.enableInput {
			return OsTrnFloat(levels.win.io.touch.start, 1, 0)
		} else {
			return 0
		}
	case SA_DIV_GET_touchWheel:
		if div.enableInput {
			return float64(levels.win.io.touch.wheel)
		} else {
			return 0
		}
	case SA_DIV_GET_touchClicks:
		if div.enableInput {
			return float64(levels.win.io.touch.numClicks)
		} else {
			return 0
		}
	case SA_DIV_GET_touchForce:
		if div.enableInput {
			return OsTrnFloat(levels.win.io.touch.rm, 1, 0)
		} else {
			return 0
		}

	case SA_DIV_GET_touchActive:
		return OsTrnFloat(div.data.touch_active, 1, 0)
	case SA_DIV_GET_touchEnd:
		return OsTrnFloat(div.data.touch_end, 1, 0)

	case SA_DIV_GET_touchCol:
		return float64(div.data.cols.GetCloseCell(div.GetRelativePos(levels.win.io.touch.pos).X))
	case SA_DIV_GET_touchRow:
		return float64(div.data.rows.GetCloseCell(div.GetRelativePos(levels.win.io.touch.pos).Y))

	case SA_DIV_GET_startCol:
		return float64(div.data.cols.GetCloseCell(div.GetRelativePos(div.crop.Start).X))
	case SA_DIV_GET_startRow:
		return float64(div.data.rows.GetCloseCell(div.GetRelativePos(div.crop.Start).Y))

	case SA_DIV_GET_endCol:
		return float64(div.data.cols.GetCloseCell(div.GetRelativePos(div.crop.End()).X))
	case SA_DIV_GET_endRow:
		return float64(div.data.rows.GetCloseCell(div.GetRelativePos(div.crop.End()).Y))

	case SA_DIV_GET_scrollVpos:
		return float64(div.data.scrollV.GetWheel()) / float64(levels.win.Cell())
	case SA_DIV_GET_scrollHpos:
		return float64(div.data.scrollH.GetWheel()) / float64(levels.win.Cell())

	case SA_DIV_GET_scrollVshow:
		return OsTrnFloat(div.data.scrollV.show, 1, 0)
	case SA_DIV_GET_scrollHshow:
		return OsTrnFloat(div.data.scrollH.show, 1, 0)

	case SA_DIV_GET_scrollVnarrow:
		return OsTrnFloat(div.data.scrollV.narrow, 1, 0)
	case SA_DIV_GET_scrollHnarrow:
		return OsTrnFloat(div.data.scrollH.narrow, 1, 0)
	}

	return -1
}

func (levels *Ui) DivInfo_set(cmd uint8, val float64, uid float64) float64 {

	lv := levels.GetCall()

	div := lv.call.FindUid(uid)
	if div == nil {
		return -1
	}

	switch cmd {
	case SA_DIV_SET_touch_enable:
		bck := div.data.touch_enabled
		div.data.touch_enabled = OsTrnBool(val > 0, true, false)
		return OsTrnFloat(bck, 1, 0)

	case SA_DIV_SET_scrollVpos:
		bck := float64(div.data.scrollV.GetWheel()) / float64(levels.win.Cell())
		div.data.scrollV.wheel = int(val * float64(levels.win.Cell()))
		return bck

	case SA_DIV_SET_scrollHpos:
		bck := float64(div.data.scrollH.GetWheel()) / float64(levels.win.Cell())
		div.data.scrollH.wheel = int(val * float64(levels.win.Cell()))
		return bck

	case SA_DIV_SET_scrollVshow:
		bck := div.data.scrollV.show
		div.data.scrollV.show = OsTrnBool(val > 0, true, false)
		return OsTrnFloat(bck, 1, 0)

	case SA_DIV_SET_scrollHshow:
		bck := div.data.scrollH.show
		div.data.scrollH.show = OsTrnBool(val > 0, true, false)
		return OsTrnFloat(bck, 1, 0)

	case SA_DIV_SET_scrollVnarrow:
		bck := div.data.scrollV.narrow
		div.data.scrollV.narrow = OsTrnBool(val > 0, true, false)
		return OsTrnFloat(bck, 1, 0)

	case SA_DIV_SET_scrollHnarrow:
		bck := div.data.scrollH.narrow
		div.data.scrollH.narrow = OsTrnBool(val > 0, true, false)
		return OsTrnFloat(bck, 1, 0)

	case SA_DIV_SET_scrollOnScreen:
		bck := div.data.scrollOnScreen
		div.data.scrollOnScreen = OsTrnBool(val > 0, true, false)
		return OsTrnFloat(bck, 1, 0)

	case SA_DIV_SET_copyCols:
		src := div.FindUid(val)
		if src != nil {
			div.data.cols.CopySub(&src.data.cols, 0, len(src.data.cols.outputs), levels.win.Cell())
			return 1
		}
		return -1

	case SA_DIV_SET_copyRows:
		src := div.FindUid(val)
		if src != nil {
			div.data.rows.CopySub(&src.data.rows, 0, len(src.data.rows.outputs), levels.win.Cell())
			return 1
		}
		return -1

	case SA_DIV_SET_attachScrollH:
		src := div.FindUid(val)
		if src != nil {
			div.data.scrollH.attach = &src.data.scrollH
			src.data.scrollH.attach = &div.data.scrollH
			return 1
		}
		return -1

	case SA_DIV_SET_attachScrollV:
		src := div.FindUid(val)
		if src != nil {
			div.data.scrollV.attach = &src.data.scrollV
			src.data.scrollV.attach = &div.data.scrollV
			return 1
		}
		return -1

	}

	return -1
}

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
)

type WinPaintBuff struct {
	win     *Win
	winRect OsV2

	crop OsV4

	skipDraw bool //not needed? ...

	depth       int
	dialogs_max int

	hosts_iter      int
	lastReset_ticks int64
}

const WinPaintBuff_MAX_ITER = 2

func NewWinPaintBuff(win *Win) *WinPaintBuff {
	var b WinPaintBuff
	b.win = win
	return &b
}

func (b *WinPaintBuff) Destroy() {
}

func (b *WinPaintBuff) ResetHost() { // rename to redraw ...
	b.hosts_iter = 0
	b.lastReset_ticks = OsTicks()
}

func (b *WinPaintBuff) IsHostHard() bool {
	return b.hosts_iter < WinPaintBuff_MAX_ITER
}

func (b *WinPaintBuff) IncHost() bool {
	old := b.hosts_iter
	b.hosts_iter++
	return old < WinPaintBuff_MAX_ITER+1 //one more than IsHostHard(), because 2x HARD(2nd draw into buffer) + 1x SOFT(render on screen)
}

func (b *WinPaintBuff) IsHostRender() bool {
	return b.hosts_iter == WinPaintBuff_MAX_ITER-1
}

func (b *WinPaintBuff) Prepare(crop OsV4, drawBack bool) {
	b.skipDraw = false

	b.AddCrop(crop)

	if drawBack {
		b.AddRect(crop, b.win.io.GetPalette().B, 0) //depth=100
	}
	b.depth += 10 //items or are depth=110
}

func (b *WinPaintBuff) DialogStart(crop OsV4) error {
	b.crop = crop

	if !b.skipDraw {
		b.depth = (b.depth + 100) - ((b.depth + 100) % 100)

		b.dialogs_max++

		//dialog's background
		b.AddCrop(crop)
		b.AddRect(crop, b.win.io.GetPalette().B, 0)
	}

	return nil
}

func (b *WinPaintBuff) DialogEnd() error {
	if !b.skipDraw {
		if b.depth > 0 {
			b.depth = (b.depth - 100) - ((b.depth - 100) % 100)

			b.depth += 10 //items or are depth=110
		}
	}

	return nil
}

func (b *WinPaintBuff) FinalDraw() error {

	win, _ := b.win.GetScreenCoord()
	b.win.SetClipRect(win)

	//grey
	for i := 0; i < b.dialogs_max; i++ {
		b.win.DrawRect(win.Start, win.End(), i*100+50, OsCd{0, 0, 0, 80})
	}

	b.depth = 0
	b.dialogs_max = 0

	return nil
}

func (b *WinPaintBuff) AddCrop(crop OsV4) OsV4 {

	if !b.skipDraw {
		b.win.SetClipRect(crop)
	}

	ret := b.crop
	b.crop = crop
	return ret
}

func (b *WinPaintBuff) getDepth() int {
	return b.depth
}

func (b *WinPaintBuff) AddRect(coord OsV4, cd OsCd, thick int) {
	if !b.skipDraw {
		start := coord.Start
		end := coord.End()
		if thick == 0 {
			b.win.DrawRect(start, end, b.getDepth(), cd)
		} else {
			b.win.DrawRect_border(start, end, b.getDepth(), cd, thick)
		}
	}
}
func (b *WinPaintBuff) AddRectRound(coord OsV4, rad int, cd OsCd, thick int) {
	if !b.skipDraw {
		b.win.DrawRectRound(coord, rad, b.getDepth(), cd, thick)
	}
}

func (b *WinPaintBuff) AddLine(start OsV2, end OsV2, cd OsCd, thick int) {
	if !b.skipDraw {
		v := end.Sub(start)
		if !v.IsZero() {
			b.win.DrawLine(start, end, b.getDepth(), thick, cd)
		}
	}
}

func (buf *WinPaintBuff) AddBezier(a OsV2, b OsV2, c OsV2, d OsV2, cd OsCd, thick int, dash bool) {
	if !buf.skipDraw {
		buf.win.DrawBezier(a, b, c, d, buf.getDepth(), thick, cd, dash)
	}
}

func (buf *WinPaintBuff) AddTringle(a OsV2, b OsV2, c OsV2, cd OsCd) {
	if !buf.skipDraw {
		buf.win.DrawTriangle(a, b, c, buf.getDepth(), cd)
	}
}

func (b *WinPaintBuff) AddCircle(coord OsV4, cd OsCd, width int) {
	if !b.skipDraw {
		p := coord.Middle()
		b.win.DrawCicle(p, OsV2{coord.Size.X / 2, coord.Size.Y / 2}, b.getDepth(), cd, width)
	}
}

func (b *WinPaintBuff) AddImage(path WinMedia, coord OsV4, cd OsCd, alignV int, alignH int, fill bool, background bool) {

	if !b.skipDraw {
		img, err := b.win.AddImage(path) //2nd thread => black
		if err != nil {
			b.AddText(path.GetString()+" has error", 0, 0, coord, b.win.io.GetPalette().E, OsV2{1, 1}, true)
			return
		}

		if img == nil {
			return //image is empty
		}

		origSize := img.origSize

		//position
		var q OsV4
		{
			if !fill {
				rect_size := OsV2_InRatio(coord.Size, origSize)
				q = OsV4_center(coord, rect_size)
			} else {
				q.Start = coord.Start
				q.Size = OsV2_OutRatio(coord.Size, origSize)
			}

			if alignH == 0 {
				q.Start.X = coord.Start.X
			} else if alignH == 1 {
				q.Start.X = OsV4_centerFull(coord, q.Size).Start.X
			} else if alignH == 2 {
				q.Start.X = coord.End().X - q.Size.X
			}

			if alignV == 0 {
				q.Start.Y = coord.Start.Y
			} else if alignV == 1 {
				q.Start.Y = OsV4_centerFull(coord, q.Size).Start.Y
			} else if alignV == 2 {
				q.Start.Y = coord.End().Y - q.Size.Y
			}
		}

		//draw image
		imgRectBackup := b.AddCrop(b.crop.GetIntersect(coord))
		err = img.Draw(q, b.getDepth()-OsTrn(background, 1, 0), cd)
		if err != nil {
			fmt.Printf("Draw() failed: %v\n", err)
		}
		b.AddCrop(imgRectBackup)
	}
}

func (b *WinPaintBuff) AddText(text string, textH float64, lineH float64, coord OsV4, frontCd OsCd, align OsV2, enableFormating bool) {
	if !b.skipDraw {
		b.win.DrawText(text, textH, lineH, coord, b.getDepth(), align, frontCd, enableFormating)
	}
}

func (b *WinPaintBuff) AddTextBack(rangee OsV2, text string, textH float64, lineH float64, coord OsV4, cd OsCd, align OsV2, enableFormating bool, underline bool) {

	if rangee.X == rangee.Y {
		return
	}

	start := b.win.GetTextStart(text, textH, lineH, coord, align, enableFormating)

	var rng OsV2
	rng.X = b.win.GetTextSize(rangee.X, text, textH, lineH, enableFormating).X
	rng.Y = b.win.GetTextSize(rangee.Y, text, textH, lineH, enableFormating).X

	rng.Sort()

	if rng.X != rng.Y {
		if underline {
			Y := coord.Start.Y + coord.Size.Y
			b.AddRect(OsV4{Start: OsV2{start.X + rng.X, Y - 2}, Size: OsV2{rng.Y, 2}}, cd, 0)
		} else {
			hPx, _ := b.win.getTextAndLineHight(textH, lineH)

			c := InitOsV4(start.X+rng.X, coord.Start.Y, rng.Y-rng.X, coord.Size.Y)
			c = c.AddSpaceY((coord.Size.Y-hPx)/2 - (hPx / 2)) //smaller height

			b.AddRect(c, cd, 0)
		}
	}
}

func (b *WinPaintBuff) AddTextCursor(text string, textH float64, lineH float64, coord OsV4, cd OsCd, align OsV2, enableFormating bool, cursorPos int) OsV4 {
	b.win.cursorEdit = true
	cd.A = b.win.cursorCdA

	start := b.win.GetTextStart(text, textH, lineH, coord, align, enableFormating)

	rngX := b.win.GetTextSize(cursorPos, text, textH, lineH, enableFormating).X
	hPx, _ := b.win.getTextAndLineHight(textH, lineH)

	c := InitOsV4(start.X+rngX, coord.Start.Y, OsMax(1, b.win.Cell()/15), coord.Size.Y)
	c = c.AddSpaceY((coord.Size.Y-hPx)/2 - (hPx / 2)) //smaller height

	b.AddRect(c, cd, 0)

	return c
}

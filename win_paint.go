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

	dialogs_num int
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
		b.AddRect(crop, b.win.io.GetPalette().B, 0)
	}
}

func (b *WinPaintBuff) DialogStart(crop OsV4) error {
	b.crop = crop

	if !b.skipDraw {
		b.dialogs_num++
		b.dialogs_max++

		//dialog's background
		b.AddCrop(crop)
		b.AddRect(crop, b.win.io.GetPalette().B, 0)
	}

	return nil
}

func (b *WinPaintBuff) DialogEnd() error {
	if !b.skipDraw {
		if b.dialogs_num > 0 {
			b.dialogs_num--
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

	b.dialogs_num = 0
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
	return b.dialogs_num * 100
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

func (b *WinPaintBuff) AddCircle(coord OsV4, cd OsCd, thick int) {
	if !b.skipDraw {
		p := coord.Middle()
		b.win.DrawCicle(p, OsV2{coord.Size.X / 2, coord.Size.Y / 2}, b.getDepth(), cd, thick)
	}
}

func (b *WinPaintBuff) AddImage(path WinMediaPath, coord OsV4, cd OsCd, alignV int, alignH int, fill bool) {

	if !b.skipDraw {
		img, err := b.win.AddImage(path) //2nd thread => black
		if err != nil {
			b.AddText(path.GetString()+" has error", coord, b.win.fonts.Get(SKYALT_FONT_PATH), b.win.io.GetPalette().E, b.win.io.GetDPI()/8, OsV2{1, 1}, nil, true)
			return
		}

		if img == nil {
			return //image is empty
		}

		origSize := img.origSize

		var q OsV4
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

		imgRectBackup := b.AddCrop(b.crop.GetIntersect(coord))
		err = img.Draw(q, b.getDepth(), cd)
		if err != nil {
			fmt.Printf("Draw() failed: %v\n", err)
		}
		b.AddCrop(imgRectBackup)
	}
}

func (b *WinPaintBuff) AddText(text string, coord OsV4, font *WinFont, cd OsCd, h int, align OsV2, cds []OsCd, enableFormating bool) {
	if !b.skipDraw {
		font.Print(text, g_WinFont_DEFAULT_Weight, h, coord, b.getDepth(), align, cd, cds, true, enableFormating, b.win)
	}
}

func (b *WinPaintBuff) AddTextBack(rangee OsV2, text string, coord OsV4, font *WinFont, cd OsCd, h int, align OsV2, underline bool, enableFormating bool) error {

	if rangee.X == rangee.Y {
		return nil
	}

	start, err := font.Start(text, g_WinFont_DEFAULT_Weight, h, coord, align, enableFormating, nil)
	if err != nil {
		return fmt.Errorf("Start() failed: %w", err)
	}

	var rng OsV2
	rng.X, err = font.GetPxPos(text, g_WinFont_DEFAULT_Weight, h, rangee.X, enableFormating)
	if err != nil {
		return fmt.Errorf("GetPxPos(1) failed: %w", err)
	}
	rng.Y, err = font.GetPxPos(text, g_WinFont_DEFAULT_Weight, h, rangee.Y, enableFormating)
	if err != nil {
		return fmt.Errorf("GetPxPos(2) failed: %w", err)
	}
	rng.Sort()

	if rng.X != rng.Y {
		if underline {
			Y := coord.Start.Y + coord.Size.Y
			b.AddRect(OsV4{Start: OsV2{start.X + rng.X, Y - 2}, Size: OsV2{rng.Y, 2}}, cd, 0)
		} else {
			c := InitOsV4(start.X+rng.X, coord.Start.Y, rng.Y-rng.X, coord.Size.Y)
			c = c.AddSpaceY((coord.Size.Y-h)/2 - (h / 2)) //smaller height

			b.AddRect(c, cd, 0)
		}
	}
	return nil
}

func (b *WinPaintBuff) AddTextCursor(text string, coord OsV4, font *WinFont, cd OsCd, h int, align OsV2, cursorPos int, cell int, enableFormating bool) (OsV4, error) {
	b.win.cursorEdit = true
	cd.A = b.win.cursorCdA

	start, err := font.Start(text, g_WinFont_DEFAULT_Weight, h, coord, align, enableFormating, nil)
	if err != nil {
		return OsV4{}, fmt.Errorf("TextCursor().Start() failed: %w", err)
	}

	ex, err := font.GetPxPos(text, g_WinFont_DEFAULT_Weight, h, cursorPos, enableFormating)
	if err != nil {
		return OsV4{}, fmt.Errorf("TextCursor().GetPxPos() failed: %w", err)
	}

	cursorQuad := InitOsV4(start.X+ex, coord.Start.Y, OsMax(1, cell/15), coord.Size.Y)
	cursorQuad = cursorQuad.AddSpaceY((coord.Size.Y-h)/2 - (h / 2)) //smaller height

	b.AddRect(cursorQuad, cd, 0)

	return cursorQuad, nil
}

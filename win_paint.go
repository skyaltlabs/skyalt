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
	win   *Win
	crop  OsV4
	depth int

	dialog_depth_backups []int
}

const WinPaintBuff_MAX_ITER = 2

func NewWinPaintBuff(win *Win) *WinPaintBuff {
	var b WinPaintBuff
	b.win = win
	return &b
}

func (b *WinPaintBuff) Destroy() {
}

func (b *WinPaintBuff) Prepare(crop OsV4, drawBack bool) {
	b.AddCrop(crop)

	if drawBack {
		b.AddRect(crop, b.win.io.GetPalette().B, 0) //depth=100
	}
	b.depth += 10 //items or are depth=110
}

func (b *WinPaintBuff) DialogStart(crop OsV4, drawBack bool, index int) error {
	b.crop = crop

	b.dialog_depth_backups = append(b.dialog_depth_backups, b.depth)

	b.depth = index * 100 //(b.depth + 100) - ((b.depth + 100) % 100)

	//dialog's background
	b.AddCrop(crop)
	if drawBack {
		b.AddRect(crop, b.win.io.GetPalette().B, 0)
	}

	return nil
}

func (b *WinPaintBuff) DialogEnd() error {
	if b.depth > 0 {

		b.depth = b.dialog_depth_backups[len(b.dialog_depth_backups)-1]
		b.dialog_depth_backups = b.dialog_depth_backups[:len(b.dialog_depth_backups)-1]

		//b.depth = (b.depth - 100) - ((b.depth - 100) % 100)
		//b.depth += 10 //items or are depth=110
	}

	return nil
}

func (b *WinPaintBuff) DrawDialogSurround(i int) {
	win, _ := b.win.GetScreenCoord()
	b.win.SetClipRect(win)
	//b.depth = 0

	//grey
	b.win.DrawRect(win.Start, win.End(), i*100+50, OsCd{0, 0, 0, 80})
}

func (b *WinPaintBuff) FinalDraw() error {

	win, _ := b.win.GetScreenCoord()
	b.win.SetClipRect(win)

	b.depth = 0
	return nil
}

func (b *WinPaintBuff) AddCrop(crop OsV4) OsV4 {

	b.win.SetClipRect(crop)

	ret := b.crop
	b.crop = crop
	return ret
}

func (b *WinPaintBuff) AddRect(coord OsV4, cd OsCd, thick int) {
	start := coord.Start
	end := coord.End()
	if thick == 0 {
		b.win.DrawRect(start, end, b.depth, cd)
	} else {
		b.win.DrawRect_border(start, end, b.depth, cd, thick)
	}

}
func (b *WinPaintBuff) AddRectRound(coord OsV4, rad int, cd OsCd, thick int) {
	b.win.DrawRectRound(coord, rad, b.depth, cd, thick, false)
}
func (b *WinPaintBuff) AddRectRoundGrad(coord OsV4, rad int, cd OsCd, thick int) {
	b.win.DrawRectRound(coord, rad, b.depth, cd, thick, true)
}

func (b *WinPaintBuff) AddLine(start OsV2, end OsV2, cd OsCd, thick int) {
	v := end.Sub(start)
	if !v.IsZero() {
		b.win.DrawLine(start, end, b.depth, thick, cd)
	}
}

func (buf *WinPaintBuff) AddBezier(a OsV2, b OsV2, c OsV2, d OsV2, cd OsCd, thick int, dash_len float32, move float32) {
	buf.win.DrawBezier(a, b, c, d, buf.depth, thick, cd, dash_len, move)
}

func (buf *WinPaintBuff) GetBezier(a OsV2, b OsV2, c OsV2, d OsV2, t float64) (OsV2f, OsV2f) {
	return buf.win.GetBezier(a, b, c, d, t)
}

func (buf *WinPaintBuff) GetPoly(points []OsV2f, width float64) *WinGphItemPoly {
	return buf.win.GetPoly(points, width)
}
func (buf *WinPaintBuff) AddPolyStart(start OsV2, poly *WinGphItemPoly, cd OsCd) {
	buf.win.DrawPolyStart(start, poly, buf.depth, cd)
}
func (buf *WinPaintBuff) AddPolyRect(rect OsV4, poly *WinGphItemPoly, cd OsCd) {
	buf.win.DrawPolyRect(rect, poly, buf.depth, cd)
}
func (buf *WinPaintBuff) AddPolyQuad(pts [4]OsV2f, uvs [4]OsV2f, poly *WinGphItemPoly, cd OsCd) {
	buf.win.DrawPolyQuad(pts, uvs, poly, buf.depth, cd)
}

func (b *WinPaintBuff) AddCircle(coord OsV4, cd OsCd, width int) {
	p := coord.Middle()
	b.win.DrawCicle(p, OsV2{coord.Size.X / 2, coord.Size.Y / 2}, b.depth, cd, width)
}

func (b *WinPaintBuff) AddImage(path WinMedia, coord OsV4, cd OsCd, align OsV2, fill bool, background bool) {
	img, err := b.win.AddImage(path) //2nd thread => black
	if err != nil {
		b.AddText(path.GetString()+" has error", InitWinFontPropsDef(b.win), coord, b.win.io.GetPalette().OnE, OsV2{1, 1}, 0, 1)
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

		if align.X == 0 {
			q.Start.X = coord.Start.X
		} else if align.X == 1 {
			q.Start.X = OsV4_centerFull(coord, q.Size).Start.X
		} else if align.X == 2 {
			q.Start.X = coord.End().X - q.Size.X
		}

		if align.Y == 0 {
			q.Start.Y = coord.Start.Y
		} else if align.Y == 1 {
			q.Start.Y = OsV4_centerFull(coord, q.Size).Start.Y
		} else if align.Y == 2 {
			q.Start.Y = coord.End().Y - q.Size.Y
		}
	}

	//draw image
	imgRectBackup := b.AddCrop(b.crop.GetIntersect(coord))
	err = img.Draw(q, b.depth-OsTrn(background, 1, 0), cd)
	if err != nil {
		fmt.Printf("Draw() failed: %v\n", err)
	}
	b.AddCrop(imgRectBackup)
}

func (b *WinPaintBuff) AddText(ln string, prop WinFontProps, coord OsV4, frontCd OsCd, align OsV2, yLine, num_lines int) {
	b.win.DrawText(ln, prop, coord, b.depth, align, frontCd, yLine, num_lines)
}

func (b *WinPaintBuff) AddTextBack(rangee OsV2, ln string, prop WinFontProps, coord OsV4, cd OsCd, align OsV2, underline bool, yLine, num_lines int) {

	if rangee.X == rangee.Y {
		return
	}

	start := b.win.GetTextStartLine(ln, prop, coord, align, num_lines)
	start.Y += yLine * prop.lineH

	var rng OsV2
	rng.X = b.win.GetTextSize(rangee.X, ln, prop).X
	rng.Y = b.win.GetTextSize(rangee.Y, ln, prop).X

	rng.Sort()

	if num_lines > 1 {
		coord.Size.Y = prop.lineH
	}

	if rng.X != rng.Y {
		if underline {
			Y := start.Y + coord.Size.Y
			b.AddRect(OsV4{Start: OsV2{start.X + rng.X, Y - 2}, Size: OsV2{rng.Y, 2}}, cd, 0)
		} else {
			c := InitOsV4(start.X+rng.X, start.Y, rng.Y-rng.X, prop.lineH)
			b.AddRect(c, cd, 0)
		}
	}
}

func (b *WinPaintBuff) AddTextCursor(text string, prop WinFontProps, coord OsV4, cd OsCd, align OsV2, cursorPos int, yLine, numLines int) OsV4 {
	b.win.cursorEdit = true
	cd.A = b.win.cursorCdA

	start := b.win.GetTextStartLine(text, prop, coord, align, numLines)
	start.Y += yLine * prop.lineH

	rngX := b.win.GetTextSize(cursorPos, text, prop).X

	c := InitOsV4(start.X+rngX, start.Y, OsMax(1, b.win.Cell()/15), prop.lineH)
	b.AddRect(c, cd, 0)

	return c
}

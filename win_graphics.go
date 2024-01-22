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
	"image"
	"image/color"
	"os"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type WinFont struct {
	path        string
	heightPx    int
	face        font.Face
	lastUseTick int64
}

func NewWinFont(path string, heightPx int) *WinFont {

	fl, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("ReadFile() failed: %v\n", err)
		return nil
	}

	ft, err := truetype.Parse(fl)
	if err != nil {
		fmt.Printf("truetype.Parse() failed: %v\n", err)
		return nil
	}
	face := truetype.NewFace(ft, &truetype.Options{Size: float64(heightPx)})
	wf := &WinFont{path: path, heightPx: heightPx, face: face}
	wf.UpdateTick()
	return wf
}

func (ft *WinFont) Destroy() {
	ft.face.Close()
}
func (ft *WinFont) UpdateTick() {
	ft.lastUseTick = OsTicks()
}
func (it *WinFont) IsUsed() bool {
	return OsIsTicksIn(it.lastUseTick, 5000) //5 sec
}

func (ft *WinFont) GetStringSize(str string, enableFormating bool) OsV2 {

	var w fixed.Int26_6 //round to int after!
	prevC := rune(-1)
	for _, c := range str {
		if prevC >= 0 {
			w += ft.face.Kern(prevC, c)
		}
		a, _ := ft.face.GlyphAdvance(c)
		w += a
		prevC = c
	}

	m := ft.face.Metrics()
	return OsV2{int(w >> 6), int(m.Ascent+m.Descent)/64 + 2}
}

func (ft *WinFont) DrawString(str string, cd OsCd, enableFormating bool) *WinGphItemText {

	size := ft.GetStringSize(str, enableFormating)
	rgba := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))

	m := ft.face.Metrics()
	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(color.NRGBA{cd.R, cd.G, cd.B, cd.A}),
		Face: ft.face,
		Dot:  fixed.Point26_6{fixed.Int26_6(0), fixed.Int26_6(m.Ascent)},
	}

	var letters []int

	prevC := rune(-1)
	for _, c := range str {
		if prevC >= 0 {
			d.Dot.X += d.Face.Kern(prevC, c)
			letters = append(letters, int(d.Dot.X>>6))
		}
		dr, mask, maskp, advance, _ := d.Face.Glyph(d.Dot, c)
		if !dr.Empty() {
			draw.DrawMask(d.Dst, dr, d.Src, image.Point{}, mask, maskp, draw.Over)
		}
		d.Dot.X += advance
		prevC = c
	}
	if prevC >= 0 {
		letters = append(letters, int(d.Dot.X>>6))
	}

	return &WinGphItemText{item: NewWinGphItem2(rgba), font: ft, text: str, cd: cd, enableFormating: enableFormating, letters: letters}
}

type WinGphItem struct {
	texture      *WinTexture
	size         OsV2
	lastDrawTick int64
}

func NewWinGphItem2(rgba *image.RGBA) *WinGphItem {

	it := &WinGphItem{}
	it.size.X = rgba.Rect.Max.X
	it.size.Y = rgba.Rect.Max.Y

	var err error
	it.texture, err = InitWinTextureFromImageRGBA(rgba)
	if err != nil {
		fmt.Printf("InitWinTextureFromImageRGBA() failed: %v\n", err)
		return nil
	}

	return it
}

func NewWinGphItem(dc *gg.Context) *WinGphItem {
	rgba, ok := dc.Image().(*image.RGBA)
	if !ok {
		fmt.Printf("Image -> RGBA conversion failed\n")
		return nil
	}
	it := NewWinGphItem2(rgba)
	it.lastDrawTick = OsTicks()
	return it
}

func (it *WinGphItem) Destroy() {
	if it.texture != nil {
		it.texture.Destroy()
	}
}

func (it *WinGphItem) IsUsed() bool {
	return OsIsTicksIn(it.lastDrawTick, 10000) //10 sec
}
func (it *WinGphItem) UpdateTick() {
	it.lastDrawTick = OsTicks()
}

func (it *WinGphItem) Draw(coord OsV4, depth int, cd OsCd) error {

	if it.texture != nil {
		it.texture.DrawQuad(coord, depth, cd)
	}

	it.UpdateTick()
	return nil
}

type WinGphItemText struct {
	item *WinGphItem

	font            *WinFont
	text            string
	cd              OsCd
	enableFormating bool

	letters []int //aggregated!
}

type WinGphItemCircle struct {
	item  *WinGphItem
	size  OsV2
	cd    OsCd
	width float64
}

type WinGph struct {
	fonts []*WinFont

	texts   []*WinGphItemText
	circles []*WinGphItemCircle
}

func NewWinGph() *WinGph {
	gph := &WinGph{}
	return gph
}
func (gph *WinGph) Destroy() {

	for _, it := range gph.fonts {
		it.Destroy()
	}

	for _, it := range gph.circles {
		it.item.Destroy()
	}
	for _, it := range gph.texts {
		it.item.Destroy()
	}
}

func (gph *WinGph) Maintenance() {

	for i := len(gph.fonts) - 1; i >= 0; i-- {
		if !gph.fonts[i].IsUsed() {
			gph.fonts[i].Destroy()
			gph.fonts = append(gph.fonts[:i], gph.fonts[i+1:]...) //remove
		}
	}

	for i := len(gph.circles) - 1; i >= 0; i-- {
		if !gph.circles[i].item.IsUsed() {
			gph.circles[i].item.Destroy()
			gph.circles = append(gph.circles[:i], gph.circles[i+1:]...) //remove
		}
	}

	for i := len(gph.texts) - 1; i >= 0; i-- {
		if !gph.texts[i].item.IsUsed() {
			gph.texts[i].item.Destroy()
			gph.texts = append(gph.texts[:i], gph.texts[i+1:]...) //remove
		}
	}

}

func (gph *WinGph) GetFont(path string, heightPx int) *WinFont {

	//find
	for _, it := range gph.fonts {
		if it.path == path && it.heightPx == heightPx {
			return it
		}
	}

	//add
	it := NewWinFont(path, heightPx)
	if it != nil {
		gph.fonts = append(gph.fonts, it)
	}
	return it
}

func (gph *WinGph) GetText(font *WinFont, text string, cd OsCd, enableFormating bool) *WinGphItemText {
	if text == "" {
		return nil
	}
	font.UpdateTick()

	//find
	for _, it := range gph.texts {
		if it.font == font && it.text == text && it.cd.Cmp(cd) && it.enableFormating == enableFormating {
			it.item.UpdateTick()
			return it
		}
	}

	//create
	it := font.DrawString(text, cd, enableFormating)
	if it != nil {
		gph.texts = append(gph.texts, it)
	}
	return it
}

func (gph *WinGph) GetTextSize(font *WinFont, max_len int, text string, cd OsCd, enableFormating bool) OsV2 {
	if text == "" {
		return OsV2{}
	}
	font.UpdateTick()

	it := gph.GetText(font, text, cd, enableFormating)
	if it == nil {
		return OsV2{0, 0}
	}
	it.item.UpdateTick()

	if max_len < 0 || max_len >= len(it.letters) {
		return it.item.size
	}

	i := max_len - 1
	if i < 0 {
		return OsV2{0, it.item.size.Y}
	}

	if i >= len(it.letters) {
		i = len(it.letters) - 1
	}

	return OsV2{it.letters[i], it.item.size.Y}
}

func (gph *WinGph) GetTextPos(font *WinFont, px int, text string, cd OsCd, enableFormating bool) int {
	if text == "" {
		return 0
	}
	font.UpdateTick()

	it := gph.GetText(font, text, cd, enableFormating)
	if it == nil {
		return 0
	}
	it.item.UpdateTick()

	for i, ad := range it.letters {
		if px < ad {
			return i
		}
	}
	return len(it.letters)
}

func (gph *WinGph) GetCircle(size OsV2, cd OsCd, width float64) *WinGphItem {

	//find
	for _, it := range gph.circles {
		if it.size.Cmp(size) && it.cd.Cmp(cd) && it.width == width {
			return it.item
		}
	}

	//create
	dc := gg.NewContext(size.X+2, size.Y+2)
	dc.SetRGBA255(int(cd.R), int(cd.G), int(cd.B), int(cd.A))
	dc.DrawEllipse(float64(size.X)/2, float64(size.Y)/2, float64(size.X)/2, float64(size.Y)/2)
	if width > 0 {
		dc.SetLineWidth(width)
	} else {
		dc.Fill()
	}

	//add
	it := NewWinGphItem(dc)
	if it != nil {
		gph.circles = append(gph.circles, &WinGphItemCircle{item: it, size: size, cd: cd, width: width})
	}
	return it
}

/*func (gph *WinGph) GetPoly(size OsV2, cd OsCd, width float64) *WinGphItem {

	//create
	dc := gg.NewContext(size.X+2, size.Y+2)
	dc.MoveTo()
	//loop
	{
		dc.LineTo()
	}
	dc.ClosePath()
	dc.Fill()

	//add
	//...
}*/

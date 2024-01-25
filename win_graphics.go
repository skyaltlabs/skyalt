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
	"image/draw"
	"os"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
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
	face := truetype.NewFace(ft, &truetype.Options{Size: float64(heightPx) /*, Hinting: font.HintingFull*/})
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

func NextPowOf2(n int) int {
	k := 1
	for k < n {
		k = k << 1
	}
	return k
}

func (ft *WinFont) DrawString(str string, enableFormating bool) *WinGphItemText {
	size := ft.GetStringSize(str, enableFormating)

	w := NextPowOf2(size.X)
	h := NextPowOf2(size.Y)

	/*rgba := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	bCd := color.NRGBA{255, 255, 255, 0}
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			rgba.Set(x, y, bCd)
		}
	}*/

	a := image.NewAlpha(image.Rect(0, 0, w, h))

	var letters []int

	m := ft.face.Metrics()
	d := &font.Drawer{
		//Dst:  rgba,
		Dst:  a,
		Src:  image.NewUniform(color.NRGBA{255, 255, 255, 255}),
		Face: ft.face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(0), Y: fixed.Int26_6(m.Ascent)},
	}

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

	return &WinGphItemText{item: NewWinGphItemAlpha(a), size: size, font: ft, text: str, enableFormating: enableFormating, letters: letters}
}

type WinGphItem struct {
	texture      *WinTexture
	lastDrawTick int64
}

func NewWinGphItemAlpha(alpha *image.Alpha) *WinGphItem {
	it := &WinGphItem{}

	var err error
	it.texture, err = InitWinTextureFromImageAlpha(alpha)
	if err != nil {
		fmt.Printf("InitWinTextureFromImageAlpha() failed: %v\n", err)
		return nil
	}

	return it
}

func NewWinGphItemRGBA(rgba *image.RGBA) *WinGphItem {
	it := &WinGphItem{}

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
	it := NewWinGphItemRGBA(rgba)
	it.lastDrawTick = OsTicks()
	return it
}

func (it *WinGphItem) Destroy() {
	if it.texture != nil {
		it.texture.Destroy()
	}
}

func (it *WinGphItem) IsUsed() bool {
	return OsIsTicksIn(it.lastDrawTick, 5000) //5 sec
}
func (it *WinGphItem) UpdateTick() {
	it.lastDrawTick = OsTicks()
}

func (it *WinGphItem) DrawCut(coord OsV4, depth int, cd OsCd) error {
	if it.texture != nil {
		uv := OsV2f{
			float32(coord.Size.X) / float32(it.texture.size.X),
			float32(coord.Size.Y) / float32(it.texture.size.Y)}
		it.texture.DrawQuadUV(coord, depth, cd, OsV2f{}, uv)
	}

	it.UpdateTick()
	return nil
}

func (it *WinGphItem) DrawUV(item_size OsV2, coord OsV4, depth int, cd OsCd, sUV, eUV OsV2f) error {
	if it.texture != nil {
		szUv := OsV2f{
			float32(item_size.X) / float32(it.texture.size.X),
			float32(item_size.Y) / float32(it.texture.size.Y)}

		//normalize by item_size
		sUV = sUV.Mul(szUv)
		eUV = eUV.Mul(szUv)

		it.texture.DrawQuadUV(coord, depth, cd, sUV, eUV)
	}

	it.UpdateTick()
	return nil
}

type WinGphItemText struct {
	item *WinGphItem
	size OsV2

	font            *WinFont
	text            string
	enableFormating bool

	letters []int //aggregated!
}

type WinGphItemCircle struct {
	item  *WinGphItem
	size  OsV2
	width float64
	arc   OsV2f
}

type WinGph struct {
	fonts []*WinFont

	texts   []*WinGphItemText
	circles []*WinGphItemCircle

	texts_num_created int
	texts_num_remove  int
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
			gph.texts_num_remove++
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

func (gph *WinGph) GetText(font *WinFont, text string, enableFormating bool) *WinGphItemText {
	if text == "" {
		return nil
	}
	font.UpdateTick()

	if len(text) > 512 {
		text = text[:512] //cut it
	}

	//find
	for _, it := range gph.texts {
		if it.font == font && it.text == text && it.enableFormating == enableFormating {
			it.item.UpdateTick()
			return it
		}
	}

	//create
	it := font.DrawString(text, enableFormating)
	if it != nil {
		gph.texts = append(gph.texts, it)
		gph.texts_num_created++
	}
	return it
}

func (gph *WinGph) GetTextSize(font *WinFont, max_len int, text string, enableFormating bool) OsV2 {
	if text == "" {
		return OsV2{}
	}
	font.UpdateTick()

	it := gph.GetText(font, text, enableFormating)
	if it == nil {
		return OsV2{0, 0}
	}
	it.item.UpdateTick()

	if max_len < 0 || max_len >= len(it.letters) {
		return it.size
	}

	i := max_len - 1
	if i < 0 {
		return OsV2{0, it.size.Y}
	}

	if i >= len(it.letters) {
		i = len(it.letters) - 1
	}

	return OsV2{it.letters[i], it.size.Y}
}

func (gph *WinGph) GetTextPos(font *WinFont, px int, text string, enableFormating bool) int {
	if text == "" {
		return 0
	}
	font.UpdateTick()

	it := gph.GetText(font, text, enableFormating)
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

func (gph *WinGph) GetCircle(size OsV2, width float64, arc OsV2f) *WinGphItemCircle {

	//find
	for _, it := range gph.circles {
		if it.size.Cmp(size) && it.width == width && it.arc.Cmp(arc) {
			return it
		}
	}

	//create
	w := NextPowOf2(size.X)
	h := NextPowOf2(size.Y)

	dc := gg.NewContext(w, h)
	dc.SetRGBA255(255, 255, 255, 255)

	rx := float64(size.X) / 2
	ry := float64(size.Y) / 2
	sx := rx
	sy := ry

	rx -= width //can be zero
	ry -= width

	if arc.X == 0 && arc.Y == 0 {
		dc.DrawEllipse(sx, sy, rx, ry)
	} else {
		dc.NewSubPath()
		dc.MoveTo(sx, sy) //LineTo
		dc.DrawEllipticalArc(sx, sx, rx, ry, float64(arc.X), float64(arc.Y))
		dc.ClosePath()
	}

	if width > 0 {
		dc.SetLineWidth(width)
		dc.Stroke()
	} else {
		dc.Fill()
	}

	//dc.SavePNG("out.png")

	rect := image.Rect(0, 0, w, h)
	dst := image.NewAlpha(rect)
	draw.Draw(dst, rect, dc.Image(), rect.Min, draw.Src)

	//add
	var circle *WinGphItemCircle
	it := NewWinGphItemAlpha(dst)
	if it != nil {
		circle = &WinGphItemCircle{item: it, size: size, width: width, arc: arc}
		gph.circles = append(gph.circles, circle)
	}
	return circle
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

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
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type WinFontProps struct {
	weight int
	textH  int
	lineH  int

	italic bool

	enableFormating bool
}

// textH & lineH are in <0-1> range
func InitWinFontProps(weight int, textH, lineH float64, italic bool, enableFormating bool, win *Win) WinFontProps {
	if weight <= 0 {
		weight = 400
	}

	if textH <= 0 {
		textH = 0.37
	}
	tPx := int(float64(win.Cell()) * textH)

	if lineH <= 0 {
		lineH = 0.7
	}
	lPx := int(float64(win.Cell()) * lineH)

	return WinFontProps{weight: weight, textH: tPx, lineH: lPx, italic: italic, enableFormating: enableFormating}
}

func InitWinFontPropsHeight(textH float64, win *Win) WinFontProps {
	return InitWinFontProps(0, textH, 0, false, true, win)
}

func InitWinFontPropsDef(win *Win) WinFontProps {
	return InitWinFontProps(0, 0, 0, false, true, win)
}

func WinFontProps_NumRows(str string) int {
	return OsRoundUp(float64(OsText_NumLines(str))*0.7 + 0.1)
}

func (a *WinFontProps) Cmp(b *WinFontProps) bool {
	return a.weight == b.weight &&
		a.textH == b.textH &&
		a.lineH == b.lineH &&
		a.italic == b.italic &&
		a.enableFormating == b.enableFormating
}

type WinFontFace struct {
	face        font.Face
	lastUseTick int64
}

func NewWinFontFace(prop *WinFontProps) *WinFontFace {

	var name string
	switch prop.weight {
	case 100:
		name = OsTrnString(!prop.italic, "Inter-Thin", "Inter-ThinItalic")
	case 200:
		name = OsTrnString(!prop.italic, "Inter-ExtraLight", "Inter-ExtraLightItalic")
	case 300:
		name = OsTrnString(!prop.italic, "Inter-Light", "Inter-LightItalic")
	case 400:
		name = OsTrnString(!prop.italic, "Inter-Regular", "Inter-Italic") //default
	case 500:
		name = OsTrnString(!prop.italic, "Inter-Medium", "Inter-MediumItalic")
	case 600:
		name = OsTrnString(!prop.italic, "Inter-SemiBold", "Inter-SemiBoldItalic")
	case 700:
		name = OsTrnString(!prop.italic, "Inter-Bold", "Inter-BoldItalic")
	case 800:
		name = OsTrnString(!prop.italic, "Inter-ExtraBold", "Inter-ExtraBoldItalic")
	case 900:
		name = OsTrnString(!prop.italic, "Inter-Black", "Inter-BlackItalic")
	}

	if name == "" {
		fmt.Printf("Unknown wieght %d\n", prop.weight)
		return nil
	}

	path := "apps/base/resources/Inter/" + name + ".ttf"

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

	face := truetype.NewFace(ft, &truetype.Options{Size: float64(prop.textH)}) //Hinting: font.HintingFull

	return &WinFontFace{face: face}
}
func (ff *WinFontFace) Destroy() {
	ff.face.Close()
}

func (ff *WinFontFace) UpdateTick() {
	ff.lastUseTick = OsTicks()
}
func (ff *WinFontFace) IsUsed() bool {
	return OsIsTicksIn(ff.lastUseTick, 5000) //5 sec
}

type WinFont struct {
	faces      [9]*WinFontFace
	faces_ital [9]*WinFontFace
}

func (ft *WinFont) Destroy() {
	for i, f := range ft.faces {
		if f != nil {
			f.Destroy()
			ft.faces[i] = nil
		}
	}
	for i, f := range ft.faces_ital {
		if f != nil {
			f.Destroy()
			ft.faces_ital[i] = nil
		}
	}
}

func (ft *WinFont) GetFace(prop *WinFontProps) *WinFontFace {

	var ret *WinFontFace

	w := (prop.weight / 100) - 1
	w = OsClamp(w, 0, 8)

	if prop.italic {
		if ft.faces_ital[w] == nil {
			ft.faces_ital[w] = NewWinFontFace(prop)
		}
		ret = ft.faces_ital[w]
	} else {
		if ft.faces[w] == nil {
			ft.faces[w] = NewWinFontFace(prop)
		}
		ret = ft.faces[w]
	}

	ret.UpdateTick()

	return ret
}

func (ft *WinFont) Maintenance() {
	for i := len(ft.faces) - 1; i >= 0; i-- {
		if ft.faces[i] != nil && !ft.faces[i].IsUsed() {
			ft.faces[i].Destroy()
			ft.faces[i] = nil
		}
	}

	for i := len(ft.faces_ital) - 1; i >= 0; i-- {
		if ft.faces_ital[i] != nil && !ft.faces_ital[i].IsUsed() {
			ft.faces_ital[i].Destroy()
			ft.faces_ital[i] = nil
		}
	}
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

	prop WinFontProps
	text string

	letters []int //aggregated!
}

type WinGphItemCircle struct {
	item  *WinGphItem
	size  OsV2
	width float64
	arc   OsV2f
	grad  float64
}
type WinGphItemPoly struct {
	item   *WinGphItem
	points []OsV2f
	size   OsV2
	width  float64
}

func (poly *WinGphItemPoly) CmpPoints(points []OsV2f) bool {
	if len(poly.points) != len(points) {
		return false
	}

	for i, a := range poly.points {
		if !a.Cmp(points[i]) {
			return false
		}
	}
	return true
}

type WinGph struct {
	fonts []*WinFont //array index = textH

	texts   []*WinGphItemText
	circles []*WinGphItemCircle
	polys   []*WinGphItemPoly

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
	for _, it := range gph.polys {
		it.item.Destroy()
	}
}

func (gph *WinGph) Maintenance() {

	for i := len(gph.fonts) - 1; i >= 0; i-- {
		gph.fonts[i].Maintenance()
	}

	for i := len(gph.circles) - 1; i >= 0; i-- {
		if !gph.circles[i].item.IsUsed() {
			gph.circles[i].item.Destroy()
			gph.circles = append(gph.circles[:i], gph.circles[i+1:]...) //remove
		}
	}

	for i := len(gph.polys) - 1; i >= 0; i-- {
		if !gph.polys[i].item.IsUsed() {
			gph.polys[i].item.Destroy()
			gph.polys = append(gph.polys[:i], gph.polys[i+1:]...) //remove
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

func (gph *WinGph) GetFont(prop *WinFontProps) *WinFont {

	for i := len(gph.fonts); i < prop.textH+1; i++ {
		gph.fonts = append(gph.fonts, &WinFont{})
	}

	return gph.fonts[prop.textH]
}

func (gph *WinGph) GetTextSize(prop WinFontProps, max_len int, text string) OsV2 {
	if text == "" {
		return OsV2{}
	}

	it := gph.GetText(prop, text)
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

func (gph *WinGph) GetTextPos(prop WinFontProps, px int, text string) int {
	if text == "" {
		return 0
	}

	it := gph.GetText(prop, text)
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

func (gph *WinGph) GetText(prop WinFontProps, text string) *WinGphItemText {
	if text == "" {
		return nil
	}

	if len(text) > 512 {
		text = text[:512] //cut it
	}

	//find
	for _, it := range gph.texts {
		if it.prop.Cmp(&prop) && it.text == text {
			it.item.UpdateTick()
			return it
		}
	}

	//create
	it := gph.drawString(prop, text)
	if it != nil {
		gph.texts = append(gph.texts, it)
		gph.texts_num_created++
	}
	return it
}

func (gph *WinGph) GetCircle(size OsV2, width float64, arc OsV2f) *WinGphItemCircle {

	//find
	for _, it := range gph.circles {
		if it.size.Cmp(size) && it.width == width && it.arc.Cmp(arc) && it.grad == 0 {
			return it
		}
	}

	//create
	w := OsNextPowOf2(size.X)
	h := OsNextPowOf2(size.Y)

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

func (gph *WinGph) GetCircleGrad(size OsV2, arc OsV2f, alpha float64) *WinGphItemCircle {

	//find
	for _, it := range gph.circles {
		if it.size.Cmp(size) && it.width == 0 && it.arc.Cmp(arc) && it.grad == alpha {
			return it
		}
	}

	//create
	w := OsNextPowOf2(size.X)
	h := OsNextPowOf2(size.Y)

	dc := gg.NewContext(w, h)

	rx := float64(size.X) / 2
	ry := float64(size.Y) / 2
	//sx := rx
	//sy := ry

	grad := gg.NewRadialGradient(rx, ry, 0, rx, ry, rx)
	grad.AddColorStop(0, color.RGBA{255, 255, 255, 255})
	grad.AddColorStop(1, color.RGBA{255, 255, 255, 0})

	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, float64(size.X), float64(size.Y))
	dc.Fill()

	//dc.SavePNG("out.png")

	rect := image.Rect(0, 0, w, h)
	dst := image.NewAlpha(rect)
	draw.Draw(dst, rect, dc.Image(), rect.Min, draw.Src)

	//add
	var circle *WinGphItemCircle
	it := NewWinGphItemAlpha(dst)
	if it != nil {
		circle = &WinGphItemCircle{item: it, size: size, width: 0, arc: arc, grad: alpha}
		gph.circles = append(gph.circles, circle)
	}
	return circle
}

func (gph *WinGph) GetPoly(points []OsV2f, width float64) *WinGphItemPoly {

	if len(points) == 0 {
		return nil
	}

	//find
	for _, it := range gph.polys {
		if it.width == width && it.CmpPoints(points) {
			return it
		}
	}

	//get size
	min := points[0]
	max := points[0]
	for _, p := range points {
		min = min.Min(p)
		max = max.Max(p)
	}

	min = min.toV2().toV2f()
	max = max.toV2().Add(OsV2{1, 1}).toV2f()

	size := max.Sub(min).toV2()
	if size.X == 0 || size.Y == 0 {
		return nil
	}
	w := OsNextPowOf2(size.X)
	h := OsNextPowOf2(size.Y)

	//create
	dc := gg.NewContext(w, h)
	for _, p := range points {
		dc.LineTo(float64(p.X-min.X), float64(p.Y-min.Y))
	}
	dc.ClosePath()

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
	var poly *WinGphItemPoly
	it := NewWinGphItemAlpha(dst)
	if it != nil {
		poly = &WinGphItemPoly{item: it, points: points, size: size, width: width}
		gph.polys = append(gph.polys, poly)
	}
	return poly

}

func (gph *WinGph) processLetter(text string, orig_prop *WinFontProps, prop *WinFontProps, skip *int) bool {

	if *skip > 0 {
		*skip -= 1
		return false
	}

	//new line = reset
	if strings.HasPrefix(text, "\n") {
		prop.weight = orig_prop.weight
		prop.italic = false
		prop.textH = orig_prop.textH
	}

	//bold + italic
	if strings.HasPrefix(text, "***") || strings.HasPrefix(text, "___") {
		prop.weight = OsTrn(prop.weight != orig_prop.weight, orig_prop.weight, orig_prop.weight*3/2) //bold
		prop.italic = !prop.italic
		*skip = 2
		return false
	}

	//bold
	if strings.HasPrefix(text, "**") {
		prop.weight = OsTrn(prop.weight != orig_prop.weight, orig_prop.weight, orig_prop.weight*3/2)
		*skip = 1
		return false
	}

	//italic
	if strings.HasPrefix(text, "__") {
		prop.italic = !prop.italic
		*skip = 1
		return false
	}

	//smaller
	if strings.HasPrefix(text, "###") {
		prop.textH = OsTrn(prop.textH != orig_prop.textH, orig_prop.textH, int(float64(orig_prop.textH)*0.9))
		*skip = 2
		return false
	}

	//taller
	if strings.HasPrefix(text, "##") {
		prop.textH = OsTrn(prop.textH != orig_prop.textH, orig_prop.textH, int(float64(orig_prop.textH)*1.2))
		*skip = 1
		return false
	}

	return true
}

func (gph *WinGph) GetStringSize(prop WinFontProps, str string) (OsV2, fixed.Int26_6) {

	var w fixed.Int26_6 //round to int after!
	prevCh := rune(-1)

	var maxH int
	var maxAscent fixed.Int26_6
	skip := 0
	act_prop := prop
	i := 0
	for p, ch := range str {
		if prop.enableFormating && !gph.processLetter(str[p:], &prop, &act_prop, &skip) {
			i++
			continue
		}

		isTab := (ch == '\t')
		if isTab {
			ch = ' '
		}

		face := gph.GetFont(&act_prop).GetFace(&act_prop).face

		if prevCh >= 0 {
			w += face.Kern(prevCh, ch)
		}
		advance, _ := face.GlyphAdvance(ch)
		if isTab {
			advance *= 8
		}

		w += advance
		prevCh = ch

		m := face.Metrics()
		maxH = OsMax(maxH, int(m.Ascent+m.Descent)>>6)
		if m.Ascent > maxAscent {
			maxAscent = m.Ascent
		}
		i++
	}

	return OsV2{int(w >> 6), maxH + 2}, maxAscent
}

func (gph *WinGph) drawString(prop WinFontProps, str string) *WinGphItemText {
	size, maxAscent := gph.GetStringSize(prop, str)

	w := OsNextPowOf2(size.X)
	h := OsNextPowOf2(size.Y)

	a := image.NewAlpha(image.Rect(0, 0, w, h))

	var letters []int
	d := &font.Drawer{
		//Dst:  rgba,
		Dst: a,
		Src: image.NewUniform(color.NRGBA{255, 255, 255, 255}),
		//Face: ft.face,
		Dot: fixed.Point26_6{X: fixed.Int26_6(0), Y: fixed.Int26_6(maxAscent)}, //Y: fixed.Int26_6(size.Y << 6)
	}

	prevCh := rune(-1)

	skip := 0
	act_prop := prop
	i := 0
	for p, ch := range str {
		if prop.enableFormating && !gph.processLetter(str[p:], &prop, &act_prop, &skip) {
			i++
			continue
		}

		isTab := (ch == '\t')
		if isTab {
			ch = ' '
		}

		d.Face = gph.GetFont(&act_prop).GetFace(&act_prop).face

		if prevCh >= 0 {
			d.Dot.X += d.Face.Kern(prevCh, ch)
			letters = append(letters, int(d.Dot.X>>6))
		}
		dr, mask, maskp, advance, _ := d.Face.Glyph(d.Dot, ch)
		if !dr.Empty() {
			draw.DrawMask(d.Dst, dr, d.Src, image.Point{}, mask, maskp, draw.Over)
		}
		if isTab {
			advance *= 8
		}

		d.Dot.X += advance
		prevCh = ch
	}

	if prevCh >= 0 {
		letters = append(letters, int(d.Dot.X>>6))
	}

	return &WinGphItemText{item: NewWinGphItemAlpha(a), size: size, prop: prop, text: str, letters: letters}
}

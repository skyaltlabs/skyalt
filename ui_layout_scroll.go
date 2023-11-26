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

type UiLayoutScroll struct {
	wheel int // pixel move

	data_height   int
	screen_height int

	clickRel int

	timeWheel int64

	show   bool
	narrow bool

	attach *UiLayoutScroll
}

func (scroll *UiLayoutScroll) Init() {

	scroll.clickRel = 0
	scroll.wheel = 0
	scroll.data_height = 1
	scroll.screen_height = 1
	scroll.timeWheel = 0
	scroll.show = true
	scroll.narrow = false
}

func (scroll *UiLayoutScroll) _getWheel(wheel int) int {

	if scroll.data_height > scroll.screen_height {
		return OsClamp(wheel, 0, (scroll.data_height - scroll.screen_height))
	}
	return 0
}

func (scroll *UiLayoutScroll) GetWheel() int {
	return scroll._getWheel(scroll.wheel)
}

func (scroll *UiLayoutScroll) SetWheel(wheelPixel int) bool {

	oldWheel := scroll.wheel

	scroll.wheel = wheelPixel
	scroll.wheel = scroll.GetWheel() //clamp by boundaries

	if oldWheel != scroll.wheel {
		scroll.timeWheel = OsTicks()

		if scroll.attach != nil {
			scroll.attach.wheel = scroll.wheel
		}
	}

	return oldWheel != scroll.wheel
}

func (scroll *UiLayoutScroll) Is() bool {
	return scroll.show && scroll.data_height > scroll.screen_height
}
func (scroll *UiLayoutScroll) IsPure() bool {
	return scroll.data_height > scroll.screen_height
}

// shorter bool,
func (scroll *UiLayoutScroll) GetScrollBackCoordV(coord OsV4, win *Win) OsV4 {
	WIDTH := scroll._GetWidth(win)
	H := 0 // OsTrn(shorter, WIDTH, 0)
	return OsV4{OsV2{coord.Start.X + coord.Size.X, coord.Start.Y}, OsV2{WIDTH, scroll.screen_height - H}}
}
func (scroll *UiLayoutScroll) GetScrollBackCoordH(coord OsV4, win *Win) OsV4 {
	WIDTH := scroll._GetWidth(win)
	H := 0 //OsTrn(shorter, WIDTH, 0)
	return OsV4{OsV2{coord.Start.X, coord.Start.Y + coord.Size.Y}, OsV2{scroll.screen_height - H, WIDTH}}
}

func (scroll *UiLayoutScroll) _GetWidth(win *Win) int {
	widthWin := win.Cell() / 2
	if scroll.narrow {
		return OsMax(4, widthWin/10)
	}
	return widthWin
}

func (scroll *UiLayoutScroll) _UpdateV(coord OsV4, win *Win) OsV4 {

	var outSlider OsV4
	if scroll.data_height <= scroll.screen_height {
		outSlider.Start = coord.Start

		outSlider.Size.X = scroll._GetWidth(win)
		outSlider.Size.Y = coord.Size.Y // self.screen_height
	} else {
		outSlider.Start.X = coord.Start.X
		outSlider.Start.Y = coord.Start.Y + int(float32(coord.Size.Y)*(float32(scroll.GetWheel())/float32(scroll.data_height)))

		outSlider.Size.X = scroll._GetWidth(win)
		outSlider.Size.Y = int(float32(coord.Size.Y) * (float32(scroll.screen_height) / float32(scroll.data_height)))
	}
	return outSlider
}

func (scroll *UiLayoutScroll) _UpdateH(start OsV2, win *Win) OsV4 {

	var outSlider OsV4
	if scroll.data_height <= scroll.screen_height {
		outSlider.Start = start

		outSlider.Size.X = scroll.screen_height
		outSlider.Size.Y = scroll._GetWidth(win)
	} else {
		outSlider.Start.X = start.X + int(float32(scroll.screen_height)*(float32(scroll.GetWheel())/float32(scroll.data_height)))
		outSlider.Start.Y = start.Y

		outSlider.Size.X = int(float32(scroll.screen_height) * (float32(scroll.screen_height) / float32(scroll.data_height)))
		outSlider.Size.Y = scroll._GetWidth(win)
	}
	return outSlider
}

func (scroll *UiLayoutScroll) _GetSlideCd(win *Win) OsCd {

	cd_slide := win.io.GetPalette().GetGrey(0.5)
	if scroll.data_height <= scroll.screen_height {
		cd_slide = win.io.GetPalette().OnB.Aprox(cd_slide, 0.5) // disable
	}

	return cd_slide
}

func (scroll *UiLayoutScroll) DrawV(coord OsV4, showBackground bool, buff *WinPaintBuff) {
	//buff := app.root.buff

	slider := scroll._UpdateV(coord, buff.win)

	slider = slider.AddSpace(OsMax(1, slider.Size.X/5))

	// make scroll visible if there is a lot of records(items)
	if slider.Size.Y == 0 {
		c := buff.win.Cell() / 4
		slider.Start.Y -= c / 2
		slider.Size.Y += c
	}

	if showBackground {
		buff.AddRect(coord, buff.win.io.GetPalette().OnB.SetAlpha(30), 0)
	}
	buff.AddRect(slider, scroll._GetSlideCd(buff.win), 0)
}

func (scroll *UiLayoutScroll) DrawH(coord OsV4, showBackground bool, buff *WinPaintBuff) {

	slider := scroll._UpdateH(coord.Start, buff.win)

	slider = slider.AddSpace(OsMax(1, slider.Size.Y/5))

	// make scroll visible if there is a lot of records(items)
	if slider.Size.Y == 0 {

		c := buff.win.Cell() / 4
		slider.Start.X -= c / 2
		slider.Size.X += c
	}

	if showBackground {
		buff.AddRect(coord, buff.win.io.GetPalette().OnB.SetAlpha(30), 0)
	}
	buff.AddRect(slider, scroll._GetSlideCd(buff.win), 0)
}

func (scroll *UiLayoutScroll) _GetTempScroll(srcl int, win *Win) int {

	return win.Cell() * srcl
}

func (scroll *UiLayoutScroll) IsMove(packLayout *UiLayoutDiv, ui *Ui, wheel_add int, deep int, onlyH bool) bool {

	inside := packLayout.CropWithScroll(ui.buff.win).Inside(ui.buff.win.io.touch.pos)
	if inside {

		//test childs
		for _, div := range packLayout.childs {
			if !onlyH && div.data.scrollV.IsMove(div, ui, wheel_add, deep+1, onlyH) {
				return deep > 0 //bottom layer must return false(can't scroll, because upper layer can scroll)
			}
			if div.data.scrollH.IsMove(div, ui, wheel_add, deep+1, onlyH) {
				return deep > 0 //bottom layer must return false(can't scroll, because upper layer can scroll)
			}
		}

		if scroll.IsPure() {
			curr := scroll.GetWheel()
			return scroll._getWheel(curr+wheel_add) != curr //can move => true
		}
	}

	return false
}

func (scroll *UiLayoutScroll) TouchV(packLayout *UiLayoutDiv, ui *Ui) {

	win := ui.buff.win

	canUp := scroll.IsMove(packLayout, ui, -1, 0, false)
	canDown := scroll.IsMove(packLayout, ui, +1, 0, false)
	if win.io.touch.wheel != 0 && !win.io.keys.shift {
		if (win.io.touch.wheel < 0 && canUp) || (win.io.touch.wheel > 0 && canDown) {
			if scroll.SetWheel(scroll.GetWheel() + scroll._GetTempScroll(win.io.touch.wheel, win)) {
				win.io.touch.wheel = 0 // let child scroll
			}
		}
	}

	if !ui.touch.IsAnyActive() && !win.io.keys.shift {
		if win.io.keys.arrowU && canUp {
			if scroll.SetWheel(scroll.GetWheel() - win.Cell()) {
				win.io.keys.arrowU = false
			}
		}
		if win.io.keys.arrowD && canDown {
			if scroll.SetWheel(scroll.GetWheel() + win.Cell()) {
				win.io.keys.arrowD = false
			}
		}

		if win.io.keys.home && canUp {
			if scroll.SetWheel(0) {
				win.io.keys.home = false
			}
		}
		if win.io.keys.end && canDown {
			if scroll.SetWheel(scroll.data_height) {
				win.io.keys.end = false
			}
		}

		if win.io.keys.pageU && canUp {
			if scroll.SetWheel(scroll.GetWheel() - scroll.screen_height) {
				win.io.keys.pageU = false
			}
		}
		if win.io.keys.pageD && canDown {
			if scroll.SetWheel(scroll.GetWheel() + scroll.screen_height) {
				win.io.keys.pageD = false
			}
		}
	}

	if !scroll.Is() {
		return
	}

	scrollCoord := packLayout.data.scrollV.GetScrollBackCoordV(packLayout.crop, win)
	if scrollCoord.Inside(win.io.touch.pos) {
		win.PaintCursor("default")
	}

	sliderFront := scroll._UpdateV(scrollCoord, win)
	midSlider := sliderFront.Size.Y / 2

	isTouched := ui.touch.IsFnMove(nil, packLayout, nil, nil)
	if win.io.touch.start {
		isTouched = sliderFront.Inside(win.io.touch.pos)
		scroll.clickRel = win.io.touch.pos.Y - sliderFront.Start.Y - midSlider // rel to middle of front slide
	}

	if isTouched { // click on slider
		mid := float32((win.io.touch.pos.Y - scrollCoord.Start.Y) - midSlider - scroll.clickRel)
		scroll.SetWheel(int((mid / float32(scrollCoord.Size.Y)) * float32(scroll.data_height)))

	} else if win.io.touch.start && scrollCoord.Inside(win.io.touch.pos) && !sliderFront.Inside(win.io.touch.pos) { // click(once) on background
		mid := float32((win.io.touch.pos.Y - scrollCoord.Start.Y) - midSlider)
		scroll.SetWheel(int((mid / float32(scrollCoord.Size.Y)) * float32(scroll.data_height)))

		// switch to 'click on slider'
		isTouched = true
		scroll.clickRel = 0
	}

	if isTouched {
		ui.touch.Set(nil, packLayout, nil, nil)
	}

	scroll.attach = nil //reset
}

func (scroll *UiLayoutScroll) TouchH(needShiftWheel bool, packLayout *UiLayoutDiv, ui *Ui) {
	win := ui.buff.win

	canLeft := scroll.IsMove(packLayout, ui, -1, 0, win.io.keys.shift)
	canRight := scroll.IsMove(packLayout, ui, +1, 0, win.io.keys.shift)
	if win.io.touch.wheel != 0 && (!needShiftWheel || win.io.keys.shift) {
		if (win.io.touch.wheel < 0 && canLeft) || (win.io.touch.wheel > 0 && canRight) {
			if scroll.SetWheel(scroll.GetWheel() + scroll._GetTempScroll(win.io.touch.wheel, win)) {
				win.io.touch.wheel = 0 // let child scroll
			}
		}
	}

	if !ui.touch.IsAnyActive() && (!needShiftWheel || win.io.keys.shift) {
		if win.io.keys.arrowL && canLeft {
			if scroll.SetWheel(scroll.GetWheel() - win.Cell()) {
				win.io.keys.arrowL = false
			}
		}
		if win.io.keys.arrowR && canRight {
			if scroll.SetWheel(scroll.GetWheel() + win.Cell()) {
				win.io.keys.arrowR = false
			}
		}

		if win.io.keys.home && canLeft {
			if scroll.SetWheel(0) {
				win.io.keys.home = false
			}
		}
		if win.io.keys.end && canRight {
			if scroll.SetWheel(scroll.data_height) {
				win.io.keys.end = false
			}
		}

		if win.io.keys.pageU && canLeft {
			if scroll.SetWheel(scroll.GetWheel() - scroll.screen_height) {
				win.io.keys.pageU = false
			}
		}
		if win.io.keys.pageD && canRight {
			if scroll.SetWheel(scroll.GetWheel() + scroll.screen_height) {
				win.io.keys.pageD = false
			}
		}
	}

	if !scroll.Is() {
		return
	}

	scrollCoord := packLayout.data.scrollV.GetScrollBackCoordH(packLayout.crop, win)
	if scrollCoord.Inside(win.io.touch.pos) {
		win.PaintCursor("default")
	}

	sliderFront := scroll._UpdateH(scrollCoord.Start, win)
	midSlider := sliderFront.Size.X / 2

	isTouched := ui.touch.IsFnMove(nil, nil, packLayout, nil)
	if win.io.touch.start {
		isTouched = sliderFront.Inside(win.io.touch.pos)
		scroll.clickRel = win.io.touch.pos.X - sliderFront.Start.X - midSlider // rel to middle of front slide
	}

	if isTouched { // click on slider
		mid := float32((win.io.touch.pos.X - scrollCoord.Start.X) - midSlider - scroll.clickRel)
		scroll.SetWheel(int((mid / float32(scroll.screen_height)) * float32(scroll.data_height)))
	} else if win.io.touch.start && scrollCoord.Inside(win.io.touch.pos) && !sliderFront.Inside(win.io.touch.pos) { // click(once) on background
		mid := float32((win.io.touch.pos.X - scrollCoord.Start.X) - midSlider)
		scroll.SetWheel(int((mid / float32(scroll.screen_height)) * float32(scroll.data_height)))

		// switch to 'click on slider'
		isTouched = true
		scroll.clickRel = 0
	}

	if isTouched {
		ui.touch.Set(nil, nil, packLayout, nil)
	}

}

func (scroll *UiLayoutScroll) TryDragScroll(fast_dt int, sign int, win *Win) bool {
	wheelOld := scroll.GetWheel()

	dt := int64((1.0 / 2.0) / float32(fast_dt) * 1000)

	if OsTicks()-scroll.timeWheel > dt {
		scroll.SetWheel(scroll.GetWheel() + scroll._GetTempScroll(sign, win))
	}

	return scroll.GetWheel() != wheelOld
}

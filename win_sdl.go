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
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const SKYALT_INI_PATH = "ini.json"

func InitSDLGlobal() error {
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return fmt.Errorf("sdl.Init() failed: %w", err)
	}

	err = ttf.Init()
	if err != nil {
		return fmt.Errorf("ttf.Init() failed: %w", err)
	}

	n, err := sdl.GetNumVideoDisplays()
	if err != nil {
		return fmt.Errorf("GetNumVideoDisplays() failed: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("0 video displays")
	}

	return nil
}
func DestroySDLGlobal() {
	ttf.Quit()
	sdl.Quit()
}

type Ui struct {
	io *IO

	window *sdl.Window
	render sdl.GLContext

	winVisible bool

	lastClickUp OsV2
	numClicks   uint8

	fullscreen              bool
	recover_fullscreen_size OsV2

	cursors  []Cursor
	cursorId int

	cursorEdit          bool
	cursorTimeStart     float64
	cursorTimeEnd       float64
	cursorTimeLastBlink float64
	cursorCdA           byte

	images []*Image

	fonts *UiFonts

	particles   *Particles
	startupAnim bool

	stat       UiStat
	start_time int64
}

func NewUi() (*Ui, error) {
	ui := &Ui{}
	var err error

	ui.startupAnim = !OsFileExists(SKYALT_INI_PATH)

	ui.fonts = NewFonts()

	ui.io, err = NewIO()
	if err != nil {
		return nil, fmt.Errorf("NewIO() failed: %w", err)
	}
	err = ui.io.Open(SKYALT_INI_PATH)
	if err != nil {
		return nil, fmt.Errorf("Open() failed: %w", err)
	}

	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "2")

	// create SDL
	ui.window, err = sdl.CreateWindow("SkyAlt", int32(ui.io.ini.WinX), int32(ui.io.ini.WinY), int32(ui.io.ini.WinW), int32(ui.io.ini.WinH), sdl.WINDOW_RESIZABLE|sdl.WINDOW_OPENGL)
	if err != nil {
		return nil, fmt.Errorf("CreateWindow() failed: %w", err)
	}

	ui.render, err = ui.window.GLCreateContext()
	if err != nil {
		return nil, fmt.Errorf("GLCreateContext() failed: %w", err)
	}
	err = gl.Init()
	if err != nil {
		return nil, fmt.Errorf("gl.Init() failed: %w", err)
	}

	sdl.EventState(sdl.DROPFILE, sdl.ENABLE)
	sdl.StartTextInput()

	// cursors
	ui.cursors = append(ui.cursors, Cursor{"default", sdl.SYSTEM_CURSOR_ARROW, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_ARROW)})
	ui.cursors = append(ui.cursors, Cursor{"hand", sdl.SYSTEM_CURSOR_HAND, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_HAND)})
	ui.cursors = append(ui.cursors, Cursor{"ibeam", sdl.SYSTEM_CURSOR_IBEAM, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_IBEAM)})
	ui.cursors = append(ui.cursors, Cursor{"cross", sdl.SYSTEM_CURSOR_CROSSHAIR, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_CROSSHAIR)})

	ui.cursors = append(ui.cursors, Cursor{"res_col", sdl.SYSTEM_CURSOR_SIZEWE, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZEWE)})
	ui.cursors = append(ui.cursors, Cursor{"res_row", sdl.SYSTEM_CURSOR_SIZENS, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZENS)})
	ui.cursors = append(ui.cursors, Cursor{"res_nwse", sdl.SYSTEM_CURSOR_SIZENESW, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZENESW)}) // bug(already fixed) in SDL: https://github.com/libsdl-org/SDL/issues/2123
	ui.cursors = append(ui.cursors, Cursor{"res_nesw", sdl.SYSTEM_CURSOR_SIZENWSE, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZENWSE)})
	ui.cursors = append(ui.cursors, Cursor{"move", sdl.SYSTEM_CURSOR_SIZEALL, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZEALL)})

	ui.cursors = append(ui.cursors, Cursor{"wait", sdl.SYSTEM_CURSOR_WAITARROW, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_WAITARROW)})
	ui.cursors = append(ui.cursors, Cursor{"no", sdl.SYSTEM_CURSOR_NO, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_NO)})

	if ui.startupAnim {
		ui.SetProgress(5)
	}

	return ui, nil
}

func (ui *Ui) Destroy() error {
	var err error

	ui.io.Save(SKYALT_INI_PATH)

	if ui.particles != nil {
		ui.particles.Destroy()
	}

	err = ui.io.Destroy()
	if err != nil {
		return fmt.Errorf("IO.Destroy() failed: %w", err)
	}

	for _, cur := range ui.cursors {
		sdl.FreeCursor(cur.cursor)
	}

	ui.fonts.Destroy()

	sdl.GLDeleteContext(ui.render)

	err = ui.window.Destroy()
	if err != nil {
		return fmt.Errorf("Window.Destroy() failed: %w", err)
	}

	return nil
}

func (ui *Ui) SetProgress(time_sec float32) {

	if ui.particles == nil {
		var err error
		ui.particles, err = NewParticles(ui)
		if err != nil {
			fmt.Printf("NewParticles() failed: %v\n", err)
			return
		}
	}
	ui.particles.StartAnim(time_sec)
}

func IsCtrlActive() bool {
	state := sdl.GetKeyboardState()
	return state[sdl.SCANCODE_LCTRL] != 0 || state[sdl.SCANCODE_RCTRL] != 0
}

func IsShiftActive() bool {
	state := sdl.GetKeyboardState()
	return state[sdl.SCANCODE_LSHIFT] != 0 || state[sdl.SCANCODE_RSHIFT] != 0
}

func IsAltActive() bool {
	state := sdl.GetKeyboardState()
	return state[sdl.SCANCODE_LALT] != 0 || state[sdl.SCANCODE_RALT] != 0
}

func (ui *Ui) GetMousePosition() OsV2 {

	x, y, _ := sdl.GetGlobalMouseState()

	w, h := ui.window.GetPosition()

	return OsV2_32(x, y).Sub(OsV2_32(w, h))
}

func (ui *Ui) GetOutputSize() (int, int) {
	w, h := ui.window.GLGetDrawableSize()
	return int(w), int(h)
}

func (ui *Ui) GetScreenCoord() (OsV4, error) {

	w, h := ui.GetOutputSize()
	return OsV4{Start: OsV2{}, Size: OsV2{w, h}}, nil
}

func (ui *Ui) SaveScreenshot() error {

	screen, err := ui.GetScreenCoord()
	if err != nil {
		return fmt.Errorf("GetScreenCoord() failed: %w", err)
	}

	surface, err := sdl.CreateRGBSurface(0, int32(screen.Size.X), int32(screen.Size.Y), 32, 0, 0, 0, 0)
	if err != nil {
		return fmt.Errorf("CreateRGBSurface() failed: %w", err)
	}
	defer surface.Free()

	//copies pixels
	gl.ReadPixels(0, 0, int32(screen.Size.X), int32(screen.Size.Y), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&surface.Pixels()[0])) //int(surface.Pitch) ...
	if err != nil {
		return fmt.Errorf("ReadPixels() failed: %w", err)
	}
	img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{int(surface.W), int(surface.H)}})
	for y := int32(0); y < surface.H; y++ {
		for x := int32(0); x < surface.W; x++ {
			b := surface.Pixels()[y*surface.W*4+x*4+0] //blue 1st
			g := surface.Pixels()[y*surface.W*4+x*4+1]
			r := surface.Pixels()[y*surface.W*4+x*4+2] //red last
			img.SetRGBA(int(x), int(y), color.RGBA{r, g, b, 255})
		}
	}

	// creates file
	file, err := os.Create("screenshot_" + time.Now().Format("2006-1-2_15-4-5") + ".png")
	if err != nil {
		return fmt.Errorf("Create() failed: %w", err)
	}
	defer file.Close()

	//saves PNG
	err = png.Encode(file, img)
	if err != nil {
		return fmt.Errorf("Encode() failed: %w", err)
	}

	return nil
}

func (ui *Ui) NumTextures() int {
	n := 0
	for _, it := range ui.images {
		if it.texture != nil {
			n++
		}
	}
	return n
}

func (ui *Ui) GetImagesBytes() int {
	n := 0
	for _, it := range ui.images {
		n += it.GetBytes()
	}
	return n
}

func (ui *Ui) FindImage(path MediaPath) *Image {
	for _, it := range ui.images {
		if it.path.Cmp(&path) {
			return it
		}
	}
	return nil
}

func (ui *Ui) AddImage(path MediaPath) (*Image, error) {

	img := ui.FindImage(path)

	if img == nil {
		var err error
		img, err = NewImage(path)
		if err != nil {
			return nil, fmt.Errorf("NewImage() failed: %w", err)
		}

		if img != nil {
			ui.images = append(ui.images, img)
		}
	}

	return img, nil
}

func (ui *Ui) Event() (bool, bool, error) {

	io := ui.io
	inputChanged := false

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() { // some cases have RETURN(don't miss it in tick), some (can be missed in tick)!

		switch val := event.(type) {
		case sdl.QuitEvent:
			fmt.Println("Exiting ..")
			return false, false, nil

		case sdl.WindowEvent:
			switch val.Event {
			case sdl.WINDOWEVENT_SIZE_CHANGED:
				inputChanged = true
			case sdl.WINDOWEVENT_MOVED:
				inputChanged = true
			case sdl.WINDOWEVENT_SHOWN:
				ui.winVisible = true
				inputChanged = true
			case sdl.WINDOWEVENT_HIDDEN:
				ui.winVisible = false
				inputChanged = true
			}

		case sdl.MouseMotionEvent:
			inputChanged = true

		case sdl.MouseButtonEvent:

			ui.numClicks = val.Clicks
			if val.Clicks > 1 {
				if ui.lastClickUp.Distance(OsV2_32(val.X, val.Y)) > float32(ui.Cell())/5 { //7px error space
					ui.numClicks = 1
				}
			}

			io.touch.pos = OsV2_32(val.X, val.Y)
			io.touch.rm = (val.Button != sdl.ButtonLeft)
			//io.touch.numClicks = val.Clicks

			if val.Type == sdl.MOUSEBUTTONDOWN {
				io.touch.start = true
				sdl.CaptureMouse(true) // keep getting info even mouse is outside window

			} else if val.Type == sdl.MOUSEBUTTONUP {

				ui.lastClickUp = io.touch.pos
				io.touch.end = true
				sdl.CaptureMouse(false)
			}
			return true, true, nil

		case sdl.MouseWheelEvent:

			if IsCtrlActive() { // zoom

				if val.Y > 0 {
					io.SetDPI(io.GetDPI() + 3)
				}
				if val.Y < 0 {
					io.SetDPI(io.GetDPI() - 3)
				}
			} else {
				io.touch.wheel = -int(val.Y) // divide by -WHEEL_DELTA
			}
			return true, true, nil

		case sdl.KeyboardEvent:
			if val.Type == sdl.KEYDOWN {

				if IsCtrlActive() {
					if val.Keysym.Sym == sdl.K_PLUS || val.Keysym.Sym == sdl.K_KP_PLUS {
						io.SetDPI(io.GetDPI() + 3)
					}
					if val.Keysym.Sym == sdl.K_MINUS || val.Keysym.Sym == sdl.K_KP_MINUS {
						io.SetDPI(io.GetDPI() - 3)
					}
					if val.Keysym.Sym == sdl.K_0 || val.Keysym.Sym == sdl.K_KP_0 {
						dpi, err := _IO_getDPI()
						if err == nil {
							io.SetDPI(dpi)
						}
					}
				}

				keys := &io.keys

				keys.esc = val.Keysym.Sym == sdl.K_ESCAPE
				keys.enter = (val.Keysym.Sym == sdl.K_RETURN || val.Keysym.Sym == sdl.K_RETURN2 || val.Keysym.Sym == sdl.K_KP_ENTER)

				keys.arrowU = val.Keysym.Sym == sdl.K_UP
				keys.arrowD = val.Keysym.Sym == sdl.K_DOWN
				keys.arrowL = val.Keysym.Sym == sdl.K_LEFT
				keys.arrowR = val.Keysym.Sym == sdl.K_RIGHT
				keys.home = val.Keysym.Sym == sdl.K_HOME
				keys.end = val.Keysym.Sym == sdl.K_END
				keys.pageU = val.Keysym.Sym == sdl.K_PAGEUP
				keys.pageD = val.Keysym.Sym == sdl.K_PAGEDOWN

				keys.copy = val.Keysym.Sym == sdl.K_COPY || (IsCtrlActive() && val.Keysym.Sym == sdl.K_c)
				keys.cut = val.Keysym.Sym == sdl.K_CUT || (IsCtrlActive() && val.Keysym.Sym == sdl.K_x)
				keys.paste = val.Keysym.Sym == sdl.K_PASTE || (IsCtrlActive() && val.Keysym.Sym == sdl.K_v)
				keys.selectAll = val.Keysym.Sym == sdl.K_SELECT || (IsCtrlActive() && val.Keysym.Sym == sdl.K_a)
				keys.backward = val.Keysym.Sym == sdl.K_AC_FORWARD || (IsCtrlActive() && !IsShiftActive() && val.Keysym.Sym == sdl.K_z)
				keys.forward = val.Keysym.Sym == sdl.K_AC_BACK || (IsCtrlActive() && val.Keysym.Sym == sdl.K_y) || (IsCtrlActive() && IsShiftActive() && val.Keysym.Sym == sdl.K_z)

				keys.tab = val.Keysym.Sym == sdl.K_TAB

				keys.delete = val.Keysym.Sym == sdl.K_DELETE
				keys.backspace = val.Keysym.Sym == sdl.K_BACKSPACE

				keys.f1 = val.Keysym.Sym == sdl.K_F1
				keys.f2 = val.Keysym.Sym == sdl.K_F2
				keys.f3 = val.Keysym.Sym == sdl.K_F3
				keys.f4 = val.Keysym.Sym == sdl.K_F4
				keys.f5 = val.Keysym.Sym == sdl.K_F5
				keys.f6 = val.Keysym.Sym == sdl.K_F6
				keys.f7 = val.Keysym.Sym == sdl.K_F7
				keys.f8 = val.Keysym.Sym == sdl.K_F8
				keys.f9 = val.Keysym.Sym == sdl.K_F9
				keys.f10 = val.Keysym.Sym == sdl.K_F10
				keys.f11 = val.Keysym.Sym == sdl.K_F11
				keys.f12 = val.Keysym.Sym == sdl.K_F12

				let := val.Keysym.Sym
				if OsIsTextWord(rune(let)) || let == ' ' {
					if IsCtrlActive() {
						keys.ctrlChar = string(let) //string([]byte{byte(let)})
					}
					if IsAltActive() {
						keys.altChar = string(let)
					}
				}

				keys.hasChanged = true
			}
			return true, true, nil

		case sdl.TextInputEvent:
			if !(IsCtrlActive() && len(val.Text) > 0 && val.Text[0] == ' ') { // ignore ctrl+space
				io.keys.text += string(val.Text[:])
				io.keys.hasChanged = true
			}
			return true, true, nil

		case sdl.DropEvent:
			io.touch.drop_path = val.File
			io.touch.drop_name = filepath.Base(val.File)
			io.touch.drop_ext = filepath.Ext(val.File)
			return true, true, nil

		}
	}

	return true, inputChanged, nil
}

func (ui *Ui) Maintenance() {

	for i := len(ui.images) - 1; i >= 0; i-- {
		ok, _ := ui.images[i].Maintenance(ui)
		if !ok {
			ui.images = append(ui.images[:i], ui.images[i+1:]...)
		}
	}

	ui.fonts.Maintenance()
}

func (ui *Ui) needRedraw(inputChanged bool) bool {
	if ui == nil {
		return true
	}

	if ui.cursorEdit {
		if inputChanged {
			ui.cursorEdit = false
		}

		tm := OsTime()

		if inputChanged {
			ui.cursorTimeEnd = tm + 5 //startAfterSleep/continue blinking after mouse move
		}

		if (tm - ui.cursorTimeStart) < 0.5 {
			//star
			ui.cursorCdA = 255
		} else if tm > ui.cursorTimeEnd {
			//sleep
			if ui.cursorCdA == 0 { //last draw must be full
				ui.cursorCdA = 255
				inputChanged = true //redraw
			}
		} else if (tm - ui.cursorTimeLastBlink) > 0.5 {
			//blink swap
			if ui.cursorCdA > 0 {
				ui.cursorCdA = 0
			} else {
				ui.cursorCdA = 255
			}
			inputChanged = true //redraw
			ui.cursorTimeLastBlink = tm
		}
	}

	return inputChanged
}

func (ui *Ui) UpdateIO() (bool, bool, error) {
	if ui == nil {
		return true, false, nil
	}

	ui.fullscreen = ui.io.ini.Fullscreen

	ok, winChanged, err := ui.Event()
	if err != nil {
		return ok, true, fmt.Errorf("Event() failed: %w", err)
	}
	if !ok {
		return false, winChanged, nil
	}

	if ui.needRedraw(winChanged) {
		winChanged = true
	}

	// update Ui
	io := ui.io

	{
		start := OsV2_32(ui.window.GetPosition())
		size := OsV2_32(ui.window.GetSize())
		io.ini.WinX = start.X
		io.ini.WinY = start.Y
		io.ini.WinW = size.X
		io.ini.WinH = size.Y
	}

	io.SetDeviceDPI()

	if !io.touch.start && !io.touch.end && !io.touch.rm {
		io.touch.pos = ui.GetMousePosition()
	}
	io.touch.numClicks = ui.numClicks
	if io.touch.end {
		ui.numClicks = 0
	}

	io.keys.shift = IsShiftActive()
	io.keys.alt = IsAltActive()
	io.keys.ctrl = IsCtrlActive()

	if io.keys.f2 {
		io.ini.Stats = !io.ini.Stats // switch
	}

	if io.keys.f3 {
		io.ini.Grid = !io.ini.Grid // switch
	}

	if io.keys.f8 {
		err := ui.SaveScreenshot()
		if err != nil {
			return true, true, fmt.Errorf("SaveScreenshot() failed: %w", err)
		}
	}

	if io.keys.f11 {
		io.ini.Fullscreen = !io.ini.Fullscreen // switch
	}

	if io.keys.paste {
		text, err := sdl.GetClipboardText()
		if err != nil {
			fmt.Println("Error: UpdateIO.GetClipboardText() failed: %w", err)
		}
		io.keys.clipboard = strings.Trim(text, "\r")

	}

	ui.cursorId = 0

	return true, winChanged, nil
}

func (ui *Ui) StartRender(clearCd OsCd) error {
	if ui == nil {
		return nil
	}

	//GL setup
	{
		screen, err := ui.GetScreenCoord()
		if err != nil {
			return fmt.Errorf("GetScreenCoord() failed: %w", err)
		}

		gl.Enable(gl.SCISSOR_TEST)

		gl.Enable(gl.DEPTH_TEST)
		gl.ClearColor(float32(clearCd.R)/255, float32(clearCd.G)/255, float32(clearCd.B)/255, float32(clearCd.A)/255)
		gl.ClearDepth(1)
		gl.DepthFunc(gl.LEQUAL)
		gl.Viewport(0, 0, int32(screen.Size.X), int32(screen.Size.Y))

		gl.MatrixMode(gl.PROJECTION)
		gl.LoadIdentity()
		gl.Ortho(0, float64(screen.Size.X), float64(screen.Size.Y), 0, -1000, 1000)
		gl.MatrixMode(gl.MODELVIEW)
		gl.LoadIdentity()

		gl.Enable(gl.POINT_SMOOTH)
		gl.Hint(gl.POINT_SMOOTH_HINT, gl.NICEST)
		gl.Enable(gl.LINE_SMOOTH)
		gl.Hint(gl.LINE_SMOOTH_HINT, gl.NICEST)
		//gl.Enable(gl.POLYGON_SMOOTH)
		//gl.Hint(gl.POLYGON_SMOOTH_HINT, gl.NICEST)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

		gl.Enable(gl.TEXTURE_2D)
	}

	w, h := ui.GetOutputSize()
	gl.Scissor(0, 0, int32(w), int32(h))

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	ui.start_time = OsTicks()
	return nil
}

func (ui *Ui) EndRender(present bool) error {
	if ui == nil {
		return nil
	}

	if ui.particles != nil && ui.particles.num_draw > 0 {
		if !ui.particles.Tick(ui) || (ui.startupAnim && ui.io.touch.start) {
			ui.particles.Clear()
			ui.startupAnim = false
		}
	}

	ui.stat.Update(int(OsTicks() - ui.start_time))
	if ui.io.ini.Stats {
		ui.renderStats()
	}

	if present {
		ui.window.GLSwap()
	}

	if ui.cursorId >= 0 {
		if ui.cursorId >= len(ui.cursors) {
			return errors.New("cursorID is out of range")
		}
		sdl.SetCursor(ui.cursors[ui.cursorId].cursor)
	}

	if ui.fullscreen != ui.io.ini.Fullscreen {
		fullFlag := uint32(0)
		if ui.io.ini.Fullscreen {
			ui.recover_fullscreen_size = OsV2_32(ui.window.GetSize())
			fullFlag = uint32(sdl.WINDOW_FULLSCREEN_DESKTOP)
		}
		err := ui.window.SetFullscreen(fullFlag)
		if err != nil {
			return fmt.Errorf("SetFullscreen() failed: %w", err)
		}
		if fullFlag == 0 {
			ui.window.SetSize(ui.recover_fullscreen_size.Get32()) //otherwise, wierd bug happens
		}
	}

	if len(ui.io.keys.clipboard) > 0 {
		sdl.SetClipboardText(ui.io.keys.clipboard)
	}

	ui.io.ResetTouchAndKeys()

	ui.Maintenance()
	return nil
}

func (ui *Ui) renderStats() error {

	font := ui.fonts.Get(SKYALT_FONT_PATH)
	textH := ui.io.GetDPI() / 6

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	text := fmt.Sprintf("FPS(worst: %.1f, best: %.1f, avg: %.1f), Memory(%d imgs: %.2fMB, process: %.2fMB), Threads(%d)",
		ui.stat.out_worst_fps, ui.stat.out_best_fps, ui.stat.out_avg_fps,
		ui.NumTextures(), float64(ui.GetImagesBytes())/1024/1024, float64(mem.Sys)/1024/1024,
		runtime.NumGoroutine())

	sz, _ := font.GetTextSize(text, g_Font_DEFAULT_Weight, textH, int(float32(textH)*1.2), true)

	cq := OsV4{ui.io.GetCoord().Middle().Sub(sz.MulV(0.5)), sz}

	ui.SetClipRect(cq)
	depth := 990 //...
	ui.DrawRect(cq.Start, cq.End(), depth, ui.io.GetPalette().B)
	err := font.Print(text, g_Font_DEFAULT_Weight, textH, cq, depth, OsV2{0, 1}, OsCd{255, 50, 50, 255}, nil, true, true, ui)
	if err != nil {
		fmt.Printf("Print() failed: %v\n", err)
	}

	return nil
}

func (ui *Ui) SetClipRect(coord OsV4) {
	if ui == nil {
		return
	}

	_, winH := ui.GetOutputSize()
	gl.Scissor(int32(coord.Start.X), int32(winH-(coord.Start.Y+coord.Size.Y)), int32(coord.Size.X), int32(coord.Size.Y))
}

func (ui *Ui) PaintCursor(name string) error {
	if ui == nil {
		return nil
	}

	for i, cur := range ui.cursors {
		if strings.EqualFold(cur.name, name) {
			ui.cursorId = i
			return nil
		}
	}

	return errors.New("Cursor(" + name + ") not found: ")
}

func (ui *Ui) DrawPointStart() {
	gl.Begin(gl.POINTS)
}
func (ui *Ui) DrawPointCdI(pos OsV2, depth int, cd OsCd) {
	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)
	gl.Vertex3i(int32(pos.X), int32(pos.Y), int32(depth))
}
func (ui *Ui) DrawPointCdF(pos OsV2f, depth int, cd OsCd) {
	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)
	gl.Vertex3f(float32(pos.X), float32(pos.Y), float32(depth))
}

func (ui *Ui) DrawPointEnd() {
	gl.End()
}

func (ui *Ui) DrawRect(start OsV2, end OsV2, depth int, cd OsCd) {
	if start.X != end.X && start.Y != end.Y {
		gl.Color4ub(cd.R, cd.G, cd.B, cd.A)

		//gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		//gl.Disable(gl.POLYGON_SMOOTH) //without, it show transparent diagonal "space"

		gl.Begin(gl.QUADS)
		{
			gl.Vertex3f(float32(start.X), float32(start.Y), float32(depth))
			gl.Vertex3f(float32(end.X), float32(start.Y), float32(depth))
			gl.Vertex3f(float32(end.X), float32(end.Y), float32(depth))
			gl.Vertex3f(float32(start.X), float32(end.Y), float32(depth))
		}
		gl.End()

		//gl.Enable(gl.POLYGON_SMOOTH)
	}
}

func (ui *Ui) DrawRect_border(start OsV2, end OsV2, depth int, cd OsCd, thick int) {
	ui.DrawRect(start, OsV2{end.X, start.Y + thick}, depth, cd) // top
	ui.DrawRect(OsV2{start.X, end.Y - thick}, end, depth, cd)   // bottom
	ui.DrawRect(start, OsV2{start.X + thick, end.Y}, depth, cd) // left
	ui.DrawRect(OsV2{end.X - thick, start.Y}, end, depth, cd)   // right
}

func (ui *Ui) DrawCicle(mid OsV2, rad OsV2, depth int, cd OsCd, thick int) {
	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)
	//gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	if thick > 0 {
		gl.LineWidth(float32(thick))
		gl.Begin(gl.LINE_LOOP)
	} else {
		gl.Begin(gl.TRIANGLE_FAN)
		gl.Vertex3f(float32(mid.X), float32(mid.Y), float32(depth)) //center
	}

	step := 2 * math.Pi / 100 //quality ...
	for i := float64(0); i < 2*math.Pi; i += step {
		gl.Vertex3f(float32(mid.X)+float32(math.Cos(i))*float32(rad.X), float32(mid.Y)+float32(math.Sin(i))*float32(rad.Y), float32(depth))
	}

	gl.End()
}

func (ui *Ui) DrawLine(start OsV2, end OsV2, depth int, thick int, cd OsCd) {

	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)

	v := end.Sub(start)
	if !v.IsZero() {

		if start.Y == end.Y {
			ui.DrawRect(start, OsV2{end.X, start.Y + thick}, depth, cd) // horizontal
		} else if start.X == end.X {
			ui.DrawRect(start, OsV2{start.X + thick, end.Y}, depth, cd) // vertical
		} else {
			gl.LineWidth(float32(thick))
			gl.Begin(gl.LINES)
			gl.Vertex3f(float32(start.X), float32(start.Y), float32(depth))
			gl.Vertex3f(float32(end.X), float32(end.Y), float32(depth))
			gl.End()
		}
	}
}

func (ui *Ui) SetTextCursorMove() {
	ui.cursorTimeStart = OsTime()
	ui.cursorTimeEnd = ui.cursorTimeStart + 5
	ui.cursorCdA = 255
}

func (ui *Ui) Cell() int {
	return ui.io.Cell()
}

func (ui *Ui) RenderTile(text string, coord OsV4, priorUp bool, cd OsCd, font *UiFont) error {
	if ui == nil {
		return nil
	}

	textH := ui.io.GetDPI() / 7

	num_lines := strings.Count(text, "\n") + 1
	cq := coord
	lineH := int(float32(textH) * 1.7)
	cq.Size, _ = font.GetTextSize(text, g_Font_DEFAULT_Weight, textH, lineH, true)
	cq = cq.AddSpaceX((lineH - textH) / -2)

	// user can set priority(up, down, etc.) ...
	cq = OsV4_relativeSurround(coord, cq, OsV4{OsV2{}, OsV2{X: ui.io.ini.WinW, Y: ui.io.ini.WinH}}, priorUp)

	ui.SetClipRect(cq)
	depth := 900 //...
	ui.DrawRect(cq.Start, cq.End(), depth, ui.io.GetPalette().B)
	ui.DrawRect_border(cq.Start, cq.End(), depth, ui.io.GetPalette().OnB, 1)
	cq.Size.Y /= num_lines
	err := font.Print(text, g_Font_DEFAULT_Weight, textH, cq, depth, OsV2{1, 1}, cd, nil, true, true, ui)
	if err != nil {
		fmt.Printf("Print() failed: %v\n", err)
	}

	return err
}

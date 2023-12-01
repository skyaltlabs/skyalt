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

type Win struct {
	io *WinIO

	window *sdl.Window
	render sdl.GLContext

	winVisible bool

	lastClickUp OsV2
	numClicks   uint8

	fullscreen              bool
	recover_fullscreen_size OsV2

	cursors  []WinCursor
	cursorId int

	//blinking cursor
	cursorEdit          bool
	cursorTimeStart     float64
	cursorTimeEnd       float64
	cursorTimeLastBlink float64
	cursorCdA           byte

	images []*WinImage

	fonts *WinFonts

	particles   *WinParticles
	startupAnim bool

	stat       WinStats
	start_time int64
}

func NewWin() (*Win, error) {
	win := &Win{}
	var err error

	win.startupAnim = !OsFileExists(SKYALT_INI_PATH)

	win.fonts = NewWinFonts()

	win.io, err = NewWinIO()
	if err != nil {
		return nil, fmt.Errorf("NewIO() failed: %w", err)
	}
	err = win.io.Open(SKYALT_INI_PATH)
	if err != nil {
		return nil, fmt.Errorf("Open() failed: %w", err)
	}

	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "2")

	// create SDL
	win.window, err = sdl.CreateWindow("SkyAlt", int32(win.io.ini.WinX), int32(win.io.ini.WinY), int32(win.io.ini.WinW), int32(win.io.ini.WinH), sdl.WINDOW_RESIZABLE|sdl.WINDOW_OPENGL)
	if err != nil {
		return nil, fmt.Errorf("CreateWindow() failed: %w", err)
	}

	win.render, err = win.window.GLCreateContext()
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
	win.cursors = append(win.cursors, WinCursor{"default", sdl.SYSTEM_CURSOR_ARROW, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_ARROW)})
	win.cursors = append(win.cursors, WinCursor{"hand", sdl.SYSTEM_CURSOR_HAND, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_HAND)})
	win.cursors = append(win.cursors, WinCursor{"ibeam", sdl.SYSTEM_CURSOR_IBEAM, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_IBEAM)})
	win.cursors = append(win.cursors, WinCursor{"cross", sdl.SYSTEM_CURSOR_CROSSHAIR, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_CROSSHAIR)})

	win.cursors = append(win.cursors, WinCursor{"res_col", sdl.SYSTEM_CURSOR_SIZEWE, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZEWE)})
	win.cursors = append(win.cursors, WinCursor{"res_row", sdl.SYSTEM_CURSOR_SIZENS, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZENS)})
	win.cursors = append(win.cursors, WinCursor{"res_nwse", sdl.SYSTEM_CURSOR_SIZENESW, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZENESW)}) // bug(already fixed) in SDL: https://github.com/libsdl-org/SDL/issues/2123
	win.cursors = append(win.cursors, WinCursor{"res_nesw", sdl.SYSTEM_CURSOR_SIZENWSE, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZENWSE)})
	win.cursors = append(win.cursors, WinCursor{"move", sdl.SYSTEM_CURSOR_SIZEALL, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZEALL)})

	win.cursors = append(win.cursors, WinCursor{"wait", sdl.SYSTEM_CURSOR_WAITARROW, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_WAITARROW)})
	win.cursors = append(win.cursors, WinCursor{"no", sdl.SYSTEM_CURSOR_NO, sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_NO)})

	if win.startupAnim {
		win.SetProgress(5)
	}

	return win, nil
}

func (win *Win) Destroy() error {
	var err error

	win.io.Save(SKYALT_INI_PATH)

	if win.particles != nil {
		win.particles.Destroy()
	}

	err = win.io.Destroy()
	if err != nil {
		return fmt.Errorf("IO.Destroy() failed: %w", err)
	}

	for _, cur := range win.cursors {
		sdl.FreeCursor(cur.cursor)
	}

	win.fonts.Destroy()

	sdl.GLDeleteContext(win.render)

	err = win.window.Destroy()
	if err != nil {
		return fmt.Errorf("Window.Destroy() failed: %w", err)
	}

	return nil
}

func (win *Win) SetProgress(time_sec float32) {

	if win.particles == nil {
		var err error
		win.particles, err = NewWinParticles(win)
		if err != nil {
			fmt.Printf("NewParticles() failed: %v\n", err)
			return
		}
	}
	win.particles.StartAnim(time_sec)
}

func IsSpaceActive() bool {
	state := sdl.GetKeyboardState()
	return state[sdl.SCANCODE_SPACE] != 0
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

func (win *Win) GetMousePosition() OsV2 {

	x, y, _ := sdl.GetGlobalMouseState()

	w, h := win.window.GetPosition()

	return OsV2_32(x, y).Sub(OsV2_32(w, h))
}

func (win *Win) GetOutputSize() (int, int) {
	w, h := win.window.GLGetDrawableSize()
	return int(w), int(h)
}

func (win *Win) GetScreenCoord() (OsV4, error) {

	w, h := win.GetOutputSize()
	return OsV4{Start: OsV2{}, Size: OsV2{w, h}}, nil
}

func (win *Win) SaveScreenshot() error {

	screen, err := win.GetScreenCoord()
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

func (win *Win) NumTextures() int {
	n := 0
	for _, it := range win.images {
		if it.texture != nil {
			n++
		}
	}
	return n
}

func (win *Win) GetImagesBytes() int {
	n := 0
	for _, it := range win.images {
		n += it.GetBytes()
	}
	return n
}

func (win *Win) FindImage(path WinMediaPath) *WinImage {
	for _, it := range win.images {
		if it.path.Cmp(&path) {
			return it
		}
	}
	return nil
}

func (win *Win) AddImage(path WinMediaPath) (*WinImage, error) {

	img := win.FindImage(path)

	if img == nil {
		var err error
		img, err = NewWinImage(path)
		if err != nil {
			return nil, fmt.Errorf("NewImage() failed: %w", err)
		}

		if img != nil {
			win.images = append(win.images, img)
		}
	}

	return img, nil
}

func (win *Win) Event() (bool, bool, error) {

	io := win.io
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
				win.winVisible = true
				inputChanged = true
			case sdl.WINDOWEVENT_HIDDEN:
				win.winVisible = false
				inputChanged = true
			}

		case sdl.MouseMotionEvent:
			inputChanged = true

		case sdl.MouseButtonEvent:

			win.numClicks = val.Clicks
			if val.Clicks > 1 {
				if win.lastClickUp.Distance(OsV2_32(val.X, val.Y)) > float32(win.Cell())/5 { //7px error space
					win.numClicks = 1
				}
			}

			io.touch.pos = OsV2_32(val.X, val.Y)
			io.touch.rm = (val.Button != sdl.ButtonLeft)
			//io.touch.numClicks = val.Clicks

			if val.Type == sdl.MOUSEBUTTONDOWN {
				io.touch.start = true
				sdl.CaptureMouse(true) // keep getting info even mouse is outside window

			} else if val.Type == sdl.MOUSEBUTTONUP {

				win.lastClickUp = io.touch.pos
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
				keys.space = val.Keysym.Sym == sdl.K_SPACE

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

func (win *Win) Maintenance() {

	for i := len(win.images) - 1; i >= 0; i-- {
		ok, _ := win.images[i].Maintenance(win)
		if !ok {
			win.images = append(win.images[:i], win.images[i+1:]...)
		}
	}

	win.fonts.Maintenance()
}

func (win *Win) needRedraw(inputChanged bool) bool {
	if win == nil {
		return true
	}

	if win.cursorEdit {
		if inputChanged {
			win.cursorEdit = false
		}

		tm := OsTime()

		if inputChanged {
			win.cursorTimeEnd = tm + 5 //startAfterSleep/continue blinking after mouse move
		}

		if (tm - win.cursorTimeStart) < 0.5 {
			//star
			win.cursorCdA = 255
		} else if tm > win.cursorTimeEnd {
			//sleep
			if win.cursorCdA == 0 { //last draw must be full
				win.cursorCdA = 255
				inputChanged = true //redraw
			}
		} else if (tm - win.cursorTimeLastBlink) > 0.5 {
			//blink swap
			if win.cursorCdA > 0 {
				win.cursorCdA = 0
			} else {
				win.cursorCdA = 255
			}
			inputChanged = true //redraw
			win.cursorTimeLastBlink = tm
		}
	}

	return inputChanged
}

func (win *Win) UpdateIO() (bool, bool, error) {
	if win == nil {
		return true, false, nil
	}

	win.fullscreen = win.io.ini.Fullscreen

	run, redraw, err := win.Event()
	if err != nil {
		return run, true, fmt.Errorf("Event() failed: %w", err)
	}
	if !run {
		return false, redraw, nil
	}

	if win.needRedraw(redraw) {
		redraw = true
	}

	// update Win
	io := win.io

	{
		start := OsV2_32(win.window.GetPosition())
		size := OsV2_32(win.window.GetSize())
		io.ini.WinX = start.X
		io.ini.WinY = start.Y
		io.ini.WinW = size.X
		io.ini.WinH = size.Y
	}

	io.SetDeviceDPI()

	if !io.touch.start && !io.touch.end && !io.touch.rm {
		io.touch.pos = win.GetMousePosition()
	}
	io.touch.numClicks = win.numClicks
	if io.touch.end {
		win.numClicks = 0
	}

	io.keys.shift = IsShiftActive()
	io.keys.alt = IsAltActive()
	io.keys.ctrl = IsCtrlActive()
	io.keys.space = IsSpaceActive()

	if io.keys.f2 {
		io.ini.Stats = !io.ini.Stats // switch
	}

	if io.keys.f3 {
		io.ini.Grid = !io.ini.Grid // switch
	}

	if io.keys.f8 {
		err := win.SaveScreenshot()
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

	win.cursorId = 0

	return true, redraw, nil
}

func (win *Win) StartRender(clearCd OsCd) error {
	if win == nil {
		return nil
	}

	//GL setup
	{
		screen, err := win.GetScreenCoord()
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

	w, h := win.GetOutputSize()
	gl.Scissor(0, 0, int32(w), int32(h))

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	win.start_time = OsTicks()
	return nil
}

func (win *Win) EndRender(present bool) error {
	if win == nil {
		return nil
	}

	if win.particles != nil && win.particles.num_draw > 0 {
		if !win.particles.Tick(win) || (win.startupAnim && win.io.touch.start) {
			win.particles.Clear()
			win.startupAnim = false
		}
	}

	win.stat.Update(int(OsTicks() - win.start_time))
	if win.io.ini.Stats {
		win.renderStats()
	}

	if present {
		win.window.GLSwap()
	}

	if win.cursorId >= 0 {
		if win.cursorId >= len(win.cursors) {
			return errors.New("cursorID is out of range")
		}
		sdl.SetCursor(win.cursors[win.cursorId].cursor)
	}

	if win.fullscreen != win.io.ini.Fullscreen {
		fullFlag := uint32(0)
		if win.io.ini.Fullscreen {
			win.recover_fullscreen_size = OsV2_32(win.window.GetSize())
			fullFlag = uint32(sdl.WINDOW_FULLSCREEN_DESKTOP)
		}
		err := win.window.SetFullscreen(fullFlag)
		if err != nil {
			return fmt.Errorf("SetFullscreen() failed: %w", err)
		}
		if fullFlag == 0 {
			win.window.SetSize(win.recover_fullscreen_size.Get32()) //otherwise, wierd bug happens
		}
	}

	if len(win.io.keys.clipboard) > 0 {
		sdl.SetClipboardText(win.io.keys.clipboard)
	}

	win.io.ResetTouchAndKeys()

	win.Maintenance()
	return nil
}

func (win *Win) renderStats() error {

	font := win.fonts.Get(SKYALT_FONT_PATH)
	textH := win.io.GetDPI() / 6

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	text := fmt.Sprintf("FPS(worst: %.1f, best: %.1f, avg: %.1f), Memory(%d imgs: %.2fMB, process: %.2fMB), Threads(%d)",
		win.stat.out_worst_fps, win.stat.out_best_fps, win.stat.out_avg_fps,
		win.NumTextures(), float64(win.GetImagesBytes())/1024/1024, float64(mem.Sys)/1024/1024,
		runtime.NumGoroutine())

	sz, _ := font.GetTextSize(text, g_WinFont_DEFAULT_Weight, textH, int(float32(textH)*1.2), true)

	cq := OsV4{win.io.GetCoord().Middle().Sub(sz.MulV(0.5)), sz}

	win.SetClipRect(cq)
	depth := 990 //...
	win.DrawRect(cq.Start, cq.End(), depth, win.io.GetPalette().B)
	err := font.Print(text, g_WinFont_DEFAULT_Weight, textH, cq, depth, OsV2{0, 1}, OsCd{255, 50, 50, 255}, nil, true, true, win)
	if err != nil {
		fmt.Printf("Print() failed: %v\n", err)
	}

	return nil
}

func (win *Win) SetClipRect(coord OsV4) {
	if win == nil {
		return
	}

	_, winH := win.GetOutputSize()
	gl.Scissor(int32(coord.Start.X), int32(winH-(coord.Start.Y+coord.Size.Y)), int32(coord.Size.X), int32(coord.Size.Y))
}

func (win *Win) PaintCursor(name string) error {
	if win == nil {
		return nil
	}

	for i, cur := range win.cursors {
		if strings.EqualFold(cur.name, name) {
			win.cursorId = i
			return nil
		}
	}

	return errors.New("Cursor(" + name + ") not found: ")
}

func (win *Win) DrawPointStart() {
	gl.Begin(gl.POINTS)
}
func (win *Win) DrawPointCdI(pos OsV2, depth int, cd OsCd) {
	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)
	gl.Vertex3i(int32(pos.X), int32(pos.Y), int32(depth))
}
func (win *Win) DrawPointCdF(pos OsV2f, depth int, cd OsCd) {
	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)
	gl.Vertex3f(float32(pos.X), float32(pos.Y), float32(depth))
}

func (win *Win) DrawPointEnd() {
	gl.End()
}

func (win *Win) DrawRect(start OsV2, end OsV2, depth int, cd OsCd) {
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

func (win *Win) DrawRect_border(start OsV2, end OsV2, depth int, cd OsCd, thick int) {
	win.DrawRect(start, OsV2{end.X, start.Y + thick}, depth, cd) // top
	win.DrawRect(OsV2{start.X, end.Y - thick}, end, depth, cd)   // bottom
	win.DrawRect(start, OsV2{start.X + thick, end.Y}, depth, cd) // left
	win.DrawRect(OsV2{end.X - thick, start.Y}, end, depth, cd)   // right
}

func (win *Win) DrawCicle(mid OsV2, rad OsV2, depth int, cd OsCd, thick int) {
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

func (win *Win) DrawLine(start OsV2, end OsV2, depth int, thick int, cd OsCd) {

	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)

	v := end.Sub(start)
	if !v.IsZero() {

		if start.Y == end.Y {
			win.DrawRect(start, OsV2{end.X, start.Y + thick}, depth, cd) // horizontal
		} else if start.X == end.X {
			win.DrawRect(start, OsV2{start.X + thick, end.Y}, depth, cd) // vertical
		} else {
			gl.LineWidth(float32(thick))
			gl.Begin(gl.LINES)
			gl.Vertex3f(float32(start.X), float32(start.Y), float32(depth))
			gl.Vertex3f(float32(end.X), float32(end.Y), float32(depth))
			gl.End()
		}
	}
}

func (win *Win) DrawBezier(a OsV2, b OsV2, c OsV2, d OsV2, depth int, thick int, cd OsCd, dash bool) {

	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)

	aa := a.toV2f()
	bb := b.toV2f()
	cc := c.toV2f()
	dd := d.toV2f()

	gl.LineWidth(float32(thick))
	if dash {
		gl.Begin(gl.LINES)
	} else {
		gl.Begin(gl.LINE_STRIP)
	}
	{
		N := float64(20)
		div := 1 / N
		for t := float64(0); t <= 1.001; t += div {
			af := aa.MulV(float32(math.Pow(t, 3)))
			bf := bb.MulV(float32(3 * math.Pow(t, 2) * (1 - t)))
			cf := cc.MulV(float32(3 * t * math.Pow((1-t), 2)))
			df := dd.MulV(float32(math.Pow((1 - t), 3)))

			r := af.Add(bf).Add(cf).Add(df)

			gl.Vertex3f(r.X, r.Y, float32(depth))
		}
	}
	gl.End()
}

func (win *Win) DrawTriangle(a OsV2, b OsV2, c OsV2, depth int, cd OsCd) {

	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)

	gl.Begin(gl.TRIANGLES)
	gl.Vertex3f(float32(a.X), float32(a.Y), float32(depth))
	gl.Vertex3f(float32(b.X), float32(b.Y), float32(depth))
	gl.Vertex3f(float32(c.X), float32(c.Y), float32(depth))
	gl.End()
}

func (win *Win) SetTextCursorMove() {
	win.cursorTimeStart = OsTime()
	win.cursorTimeEnd = win.cursorTimeStart + 5
	win.cursorCdA = 255
}

func (win *Win) Cell() int {
	return win.io.Cell()
}

func (win *Win) RenderTile(text string, coord OsV4, priorUp bool, cd OsCd, font *WinFont) error {
	if win == nil {
		return nil
	}

	textH := win.io.GetDPI() / 7

	num_lines := strings.Count(text, "\n") + 1
	cq := coord
	lineH := int(float32(textH) * 1.7)
	cq.Size, _ = font.GetTextSize(text, g_WinFont_DEFAULT_Weight, textH, lineH, true)
	cq = cq.AddSpaceX((lineH - textH) / -2)

	// user can set priority(up, down, etc.) ...
	cq = OsV4_relativeSurround(coord, cq, OsV4{OsV2{}, OsV2{X: win.io.ini.WinW, Y: win.io.ini.WinH}}, priorUp)

	win.SetClipRect(cq)
	depth := 900 //...
	win.DrawRect(cq.Start, cq.End(), depth, win.io.GetPalette().B)
	win.DrawRect_border(cq.Start, cq.End(), depth, win.io.GetPalette().OnB, 1)
	cq.Size.Y /= num_lines
	err := font.Print(text, g_WinFont_DEFAULT_Weight, textH, cq, depth, OsV2{1, 1}, cd, nil, true, true, win)
	if err != nil {
		fmt.Printf("Print() failed: %v\n", err)
	}

	return err
}

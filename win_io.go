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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

type WinKeys struct {
	hasChanged bool

	text string

	ctrlChar string
	altChar  string

	clipboard string

	shift  bool
	ctrl   bool
	alt    bool
	esc    bool
	enter  bool
	arrowU bool
	arrowD bool
	arrowL bool
	arrowR bool
	home   bool
	end    bool
	pageU  bool
	pageD  bool

	tab   bool
	space bool

	delete    bool
	backspace bool

	copy      bool
	cut       bool
	paste     bool
	selectAll bool

	backward bool
	forward  bool

	f1  bool
	f2  bool
	f3  bool
	f4  bool
	f5  bool
	f6  bool
	f7  bool
	f8  bool
	f9  bool
	f10 bool
	f11 bool
	f12 bool
}

type WinTouch struct {
	pos       OsV2
	wheel     int
	numClicks int
	start     bool
	end       bool
	rm        bool // right/middle button

	drop_path string

	wheel_last_sec float64
}

type WinCursor struct {
	name   string
	tp     sdl.SystemCursor
	cursor *sdl.Cursor
}

type WinIni struct {
	Dpi         int
	Dpi_default int

	Threads int

	DateFormat string

	Fullscreen bool
	Stats      bool
	Grid       bool

	Languages              []string
	WinX, WinY, WinW, WinH int

	Theme             string
	CustomPalette     WinCdPalette
	UseDarkTheme      bool
	UseDarkThemeStart int //hours from midnight
	UseDarkThemeEnd   int

	Offline bool
	MicOff  bool

	OpenAI_key string
}

type WinIO struct {
	touch WinTouch
	keys  WinKeys
	ini   WinIni

	palettes []WinCdPalette //don't save, only custom colors ...

}

func NewWinIO() (*WinIO, error) {
	var io WinIO

	io.palettes = append(io.palettes, InitWinCdPalette_light())
	io.palettes = append(io.palettes, InitWinCdPalette_dark())

	err := io._IO_setDefault()
	if err != nil {
		return nil, fmt.Errorf("_IO_setDefault() failed: %w", err)
	}

	return &io, nil
}

func (io *WinIO) Destroy() error {
	return nil
}

func (io *WinIO) ResetTouchAndKeys() {
	io.touch = WinTouch{}
	io.keys = WinKeys{}
}

func (io *WinIO) Open(path string) error {
	//create ini if not exist
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("OpenFile() failed: %w", err)
	}
	f.Close()

	//load ini
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ReadFile() failed: %w", err)
	}

	if len(file) > 0 {
		err = json.Unmarshal(file, &io.ini)
		if err != nil {
			return fmt.Errorf("Unmarshal() failed: %w", err)
		}
	}

	err = io._IO_setDefault()
	if err != nil {
		return fmt.Errorf("_IO_setDefault() failed: %w", err)
	}
	return nil
}

func (io *WinIO) Save(path string) error {

	file, err := json.MarshalIndent(&io.ini, "", "")
	if err != nil {
		return fmt.Errorf("MarshalIndent() failed: %w", err)
	}

	err = os.WriteFile(path, file, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile() failed: %w", err)
	}
	return nil
}

func _IO_getDPI() (int, error) {
	dpi, _, _, err := sdl.GetDisplayDPI(0)
	if err != nil {
		return 0, fmt.Errorf("GetDisplayDPI() failed: %w", err)
	}
	return int(dpi), nil
}

func (io *WinIO) _IO_setDefault() error {

	//isDefault := (io.ini.Dpi_default == 0)

	io.SetDeviceDPI()

	//dpi
	if io.ini.Dpi == 0 {
		dpi, err := _IO_getDPI()
		if err != nil {
			return fmt.Errorf("_IO_getDPI() failed: %w", err)
		}
		io.ini.Dpi = dpi
	}

	//timezone
	//if isDefault {
	//	io.ini.TimeZone = OsTimeZone()
	//}

	//date
	if io.ini.DateFormat == "" {
		io.ini.DateFormat = OsTrnString((OsTimeZone() <= -3 && OsTimeZone() >= -10), "us", "eu")
	}

	//theme
	if io.ini.Theme == "" {
		io.ini.Theme = "light"
	}

	if io.ini.CustomPalette.P.A == 0 {
		io.ini.CustomPalette = InitWinCdPalette_light()
	}

	//languages
	if len(io.ini.Languages) == 0 {
		io.ini.Languages = append(io.ini.Languages, "en")
	}

	//window coord
	if io.ini.WinW == 0 || io.ini.WinH == 0 {
		io.ini.WinX = 50
		io.ini.WinY = 50
		io.ini.WinW = 1280
		io.ini.WinH = 720
	}

	if io.ini.Threads <= 0 {
		io.ini.Threads = 1 //runtime.NumCPU()
	}

	return nil
}

func (io *WinIO) GetLanguagesString() string {
	str := ""
	for _, l := range io.ini.Languages {
		str += l
	}
	return str
}

func (io *WinIO) GetDPI() int {
	return OsClamp(io.ini.Dpi, 30, 5000)
}
func (io *WinIO) SetDPI(dpi int) {
	io.ini.Dpi = OsClamp(dpi, 30, 5000)
}

func (io *WinIO) Cell() int {
	return int(float32(io.GetDPI()) / 2.5)
}

func (io *WinIO) GetPalette() *WinCdPalette {

	theme := io.ini.Theme

	hour := time.Now().Hour()

	if io.ini.UseDarkTheme {
		if (io.ini.UseDarkThemeStart < io.ini.UseDarkThemeEnd && hour >= io.ini.UseDarkThemeStart && hour < io.ini.UseDarkThemeEnd) ||
			(io.ini.UseDarkThemeStart > io.ini.UseDarkThemeEnd && (hour >= io.ini.UseDarkThemeStart || hour < io.ini.UseDarkThemeEnd)) {
			theme = "dark"
		}
	}

	switch theme {
	case "light":
		return &io.palettes[0]
	case "dark":
		return &io.palettes[1]
	}

	return &io.ini.CustomPalette
}

func (io *WinIO) GetCoord() OsV4 {
	return OsV4{Start: OsV2{}, Size: OsV2{X: io.ini.WinW, Y: io.ini.WinH}}
}

func (io *WinIO) SetDeviceDPI() error {
	dpi, err := _IO_getDPI()
	if err != nil {
		return fmt.Errorf("_IO_getDPI() failed: %w", err)
	}
	io.ini.Dpi_default = dpi
	return nil
}

type WinCdPalette struct {
	P, S, T, E, B           OsCd
	OnP, OnS, OnT, OnE, OnB OsCd
}

const (
	CdPalette_White = uint8(0)

	CdPalette_P = uint8(1)
	CdPalette_S = uint8(2)
	CdPalette_T = uint8(3)
	CdPalette_E = uint8(4)
	CdPalette_B = uint8(5)
)

// light
func InitWinCdPalette_light() WinCdPalette {
	var pl WinCdPalette
	//Primary
	pl.P = InitOsCd32(37, 100, 120, 255)
	pl.OnP = InitOsCdWhite()
	//Secondary
	pl.S = InitOsCd32(85, 95, 100, 255)
	pl.OnS = InitOsCdWhite()
	//Tertiary
	pl.T = InitOsCd32(90, 95, 115, 255)
	pl.OnT = InitOsCdWhite()
	//Err
	pl.E = InitOsCd32(180, 40, 30, 255)
	pl.OnE = InitOsCdWhite()
	//Surface(background)
	pl.B = InitOsCd32(250, 250, 250, 255)
	pl.OnB = InitOsCd32(25, 27, 30, 255)
	return pl
}

// dark
func InitWinCdPalette_dark() WinCdPalette {
	var pl WinCdPalette
	pl.P = InitOsCd32(150, 205, 225, 255)
	pl.OnP = InitOsCd32(0, 50, 65, 255)

	pl.S = InitOsCd32(190, 200, 205, 255)
	pl.OnS = InitOsCd32(40, 50, 55, 255)

	pl.T = InitOsCd32(195, 200, 220, 255)
	pl.OnT = InitOsCd32(75, 35, 50, 255)

	pl.E = InitOsCd32(240, 185, 180, 255)
	pl.OnE = InitOsCd32(45, 45, 65, 255)

	pl.B = InitOsCd32(25, 30, 30, 255)
	pl.OnB = InitOsCd32(230, 230, 230, 255)
	return pl
}

func (pl *WinCdPalette) GetGrey(t float32) OsCd {
	return pl.S.Aprox(pl.OnS, t)
}

func (pl *WinCdPalette) GetCdOver(cd OsCd, inside bool, active bool) OsCd {
	if active {
		if inside {
			cd = cd.Aprox(pl.OnS, 0.4)
		} else {
			cd = cd.Aprox(pl.OnS, 0.3)
		}
	} else {
		if inside {
			cd = cd.Aprox(pl.S, 0.2)
		}
	}

	return cd
}

func (pl *WinCdPalette) GetCd2(cd OsCd, fade, enable, inside, active bool) OsCd {
	if fade || !enable {
		cd.A = 100
	}
	if enable {
		cd = pl.GetCdOver(cd, inside, active)
	}
	return cd
}

func (pl *WinCdPalette) GetCdI(i uint8) (OsCd, OsCd) {
	switch i {
	case CdPalette_White:
		return InitOsCdWhite(), InitOsCd32(0, 0, 0, 255)
	case CdPalette_P:
		return pl.P, pl.OnP
	case CdPalette_S:
		return pl.S, pl.OnS
	case CdPalette_T:
		return pl.T, pl.OnT
	case CdPalette_E:
		return pl.E, pl.OnE
	case CdPalette_B:
		return pl.B, pl.OnB
	}

	return pl.P, pl.OnP
}

func (pl *WinCdPalette) GetCd(i uint8, fade, enable, inside, active bool) (OsCd, OsCd) {

	cd, onCd := pl.GetCdI(i)

	cd = pl.GetCd2(cd, fade, enable, inside, active)
	onCd = pl.GetCd2(onCd, fade, enable, inside, active)

	return cd, onCd
}

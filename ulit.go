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
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
)

func OsTicks() int64 {
	return time.Now().UnixMilli()
}

func OsIsTicksIn(start_ticks int64, delay_ms int) bool {
	return OsTicks() < (start_ticks + int64(delay_ms))
}

func OsTime() float64 {
	return float64(OsTicks()) / 1000
}

func OsTimeZone() int {
	_, zn := time.Now().Zone()
	return zn / 3600
}

// Ternary operator
func OsTrn(question bool, ret_true int, ret_false int) int {
	if question {
		return ret_true
	}
	return ret_false
}

func OsTrnFloat(question bool, ret_true float64, ret_false float64) float64 {
	if question {
		return ret_true
	}
	return ret_false
}
func OsTrnString(question bool, ret_true string, ret_false string) string {
	if question {
		return ret_true
	}
	return ret_false
}

func OsTrnBool(question bool, ret_true bool, ret_false bool) bool {
	if question {
		return ret_true
	}
	return ret_false
}

func OsMax(x, y int) int {
	if x < y {
		return y
	}
	return x
}
func OsMin(x, y int) int {
	if x > y {
		return y
	}
	return x
}
func OsClamp(v, min, max int) int {
	return OsMin(OsMax(v, min), max)
}

func OsAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func OsMaxFloat(x, y float64) float64 {
	if x < y {
		return y
	}
	return x
}
func OsMinFloat(x, y float64) float64 {
	if x > y {
		return y
	}
	return x
}

func OsClampFloat(v, min, max float64) float64 {
	return OsMinFloat(OsMaxFloat(v, min), max)
}

func OsFAbs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func OsRoundDown(v float64) float64 {
	return float64(int(v))
}
func OsRoundUp(v float64) int {
	if v > OsRoundDown(v) {
		return int(v + 1)
	}
	return int(v)
}

func OsRoundHalf(v float64) float64 {
	return OsRoundDown(v + OsTrnFloat(v < 0, -0.5, 0.5))
}

type OsV2f struct {
	X float32
	Y float32
}

func (a OsV2f) Add(b OsV2f) OsV2f {
	return OsV2f{a.X + b.X, a.Y + b.Y}
}
func (a OsV2f) Sub(b OsV2f) OsV2f {
	return OsV2f{a.X - b.X, a.Y - b.Y}
}
func (a OsV2f) MulV(t float32) OsV2f {
	return OsV2f{a.X * t, a.Y * t}
}
func (a OsV2f) toV2() OsV2 {
	return OsV2{int(a.X), int(a.Y)}
}
func (a OsV2f) Cmp(b OsV2f) bool {
	return a.X == b.X && a.Y == b.Y
}

type OsV2 struct {
	X int
	Y int
}

func OsV2_32(x, y int32) OsV2 {
	return OsV2{int(x), int(y)}
}

func (v *OsV2) Get32() (int32, int32) {
	return int32(v.X), int32(v.Y)
}

func (v *OsV2) EqAdd(val OsV2) {
	v.X += val.X
	v.Y += val.Y
}
func (v *OsV2) EqSub(vel OsV2) {
	v.X -= vel.X
	v.Y -= vel.Y
}

func (a OsV2) Add(b OsV2) OsV2 {
	return OsV2{a.X + b.X, a.Y + b.Y}
}
func (a OsV2) Sub(b OsV2) OsV2 {
	return OsV2{a.X - b.X, a.Y - b.Y}
}
func (a OsV2) MulV(t float32) OsV2 {
	return OsV2{int(float32(a.X) * t), int(float32(a.Y) * t)}
}

func (a OsV2) Aprox(b OsV2, t float32) OsV2 {
	return a.Add(b.Sub(a).MulV(t))
}

func (a OsV2) toV2f() OsV2f {
	return OsV2f{float32(a.X), float32(a.Y)}
}

func (a OsV2) Cmp(b OsV2) bool {
	return a.X == b.X && a.Y == b.Y
}

func (start OsV2) Inside(end OsV2, test OsV2) bool {
	return test.X >= start.X && test.Y >= start.Y && test.X < end.X && test.Y < end.Y
}

func (a OsV2) Min(b OsV2) OsV2 {
	return OsV2{OsMin(a.X, b.X), OsMin(a.Y, b.Y)}
}

func (a OsV2) Max(b OsV2) OsV2 {
	return OsV2{OsMax(a.X, b.X), OsMax(a.Y, b.Y)}
}

func (v OsV2) Is() bool {
	return v.X != 0 && v.Y != 0
}

func (v OsV2) IsZero() bool {
	return v.X == 0 && v.Y == 0
}

func (v *OsV2) Switch() {
	*v = OsV2{v.Y, v.X}
}

func (v *OsV2) Sort() {
	if v.X > v.Y {
		v.Switch()
	}
}

func (v OsV2) Len() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}

func (a OsV2) Distance(b OsV2) float32 {
	return a.Sub(b).Len()
}

func OsV2_Intersect(a OsV2, b OsV2) OsV2 {
	v := OsV2{OsMax(a.X, b.X), OsMax(a.Y, b.Y)}

	if v.X > v.Y {
		return OsV2{}
	}
	return v
}

func OsV2_InRatio(rect OsV2, orig OsV2) OsV2 {
	rectRatio := float32(rect.X) / float32(rect.Y)
	origRatio := float32(orig.X) / float32(orig.Y)

	var ratio float32
	if origRatio > rectRatio {
		ratio = float32(rect.X) / float32(orig.X)
	} else {
		ratio = float32(rect.Y) / float32(orig.Y)
	}
	return orig.MulV(ratio)
}

func OsV2_OutRatio(rect OsV2, orig OsV2) OsV2 {
	rectRatio := float32(rect.X) / float32(rect.Y)
	origRatio := float32(orig.X) / float32(orig.Y)

	var ratio float32
	if origRatio < rectRatio {
		ratio = float32(rect.X) / float32(orig.X)
	} else {
		ratio = float32(rect.Y) / float32(orig.Y)
	}
	return orig.MulV(ratio)
}

type OsV4 struct {
	Start OsV2
	Size  OsV2
}

func InitOsV4(x, y, w, h int) OsV4 {
	return OsV4{OsV2{x, y}, OsV2{w, h}}
}

func InitOsV4Mid(mid OsV2, size OsV2) OsV4 {
	return InitOsV4(mid.X-size.X/2, mid.Y-size.Y/2, size.X, size.Y)
}

func InitOsV4ab(a OsV2, b OsV2) OsV4 {
	st := OsV2{OsMin(a.X, b.X), OsMin(a.Y, b.Y)}
	sz := OsV2{OsAbs(a.X - b.X), OsAbs(a.Y - b.Y)}
	return InitOsV4(st.X, st.Y, sz.X, sz.Y)
}

func (v OsV4) End() OsV2 {
	return OsV2{v.Start.X + v.Size.X, v.Start.Y + v.Size.Y}
}

func (v OsV4) Is() bool {
	return v.Size.Is()
}

func (v OsV4) IsZero() bool {
	return v.Size.IsZero()
}

func (v OsV4) GetPos(x, y float64) OsV2 {
	return OsV2{v.Start.X + int(float64(v.Size.X)*x), v.Start.Y + int(float64(v.Size.Y)*y)}
}

func (q OsV4) AddSpaceX(space int) OsV4 {
	space *= 2
	if space > q.Size.X {
		space = q.Size.X
	}
	return InitOsV4(q.Start.X+space/2, q.Start.Y, q.Size.X-space, q.Size.Y)
}

func (q OsV4) AddSpaceY(space int) OsV4 {
	space *= 2
	if space > q.Size.Y {
		space = q.Size.Y
	}
	return InitOsV4(q.Start.X, q.Start.Y+space/2, q.Size.X, q.Size.Y-space)
}

func (q OsV4) AddSpace(space int) OsV4 {
	r := q.AddSpaceX(space)
	return r.AddSpaceY(space)
}

func (q OsV4) Inner(top, bottom, left, right int) OsV4 {
	for q.Size.X < (left + right) { //for!
		left--
		right--
	}
	for q.Size.Y < (top + bottom) { //for!
		top--
		bottom--
	}
	return InitOsV4(q.Start.X+left, q.Start.Y+top, q.Size.X-(left+right), q.Size.Y-(top+bottom))
}

func (v OsV4) Middle() OsV2 {
	return v.Start.Add(v.Size.MulV(0.5))
}

func (v OsV4) Inside(test OsV2) bool {
	return v.Start.Inside(v.End(), test)
}
func (a OsV4) Cmp(b OsV4) bool {
	return a.Start.Cmp(b.Start) && a.Size.Cmp(b.Size)
}

func OsV4_center(out OsV4, in OsV2) OsV4 {
	r := OsV4{out.Start, in}

	if out.Size.X > in.X {
		r.Start.X += (out.Size.X - in.X) / 2
	}
	if out.Size.Y > in.Y {
		r.Start.Y += (out.Size.Y - in.Y) / 2
	}
	return r
}

func OsV4_centerFull(out OsV4, in OsV2) OsV4 {
	r := OsV4{out.Start, in}

	if out.Size.X != in.X {
		r.Start.X += (out.Size.X - in.X) / 2
	}
	if out.Size.Y != in.Y {
		r.Start.Y += (out.Size.Y - in.Y) / 2
	}
	return r
}

func (a OsV4) Area() int {
	return a.Size.X * a.Size.Y
}

func (a OsV4) Extend(b OsV4) OsV4 {

	start := OsV2{OsMin(a.Start.X, b.Start.X), OsMin(a.Start.Y, b.Start.Y)}

	ae := a.End()
	be := b.End()

	end := OsV2{OsMax(ae.X, be.X), OsMax(ae.Y, be.Y)}

	return OsV4{start, end.Sub(start)}
}

func (a OsV4) Extend2(q OsV4, v OsV2) OsV4 {

	start := OsV2{OsMin(q.Start.X, v.X), OsMin(q.Start.Y, v.Y)}

	end := q.End()
	end.X = OsMax(end.X, v.X)
	end.Y = OsMax(end.Y, v.Y)

	return OsV4{start, end.Sub(start)}
}

func (a OsV4) HasCover(b OsV4) bool {
	q := a.Extend(b)
	return q.Size.X < (a.Size.X+b.Size.X) && q.Size.Y < (a.Size.Y+b.Size.Y)
}

func (qA OsV4) GetIntersect(qB OsV4) OsV4 {

	if qA.HasCover(qB) {
		v_start := qA.Start.Max(qB.Start)
		v_end := qA.End().Min(qB.End())

		return OsV4{v_start, v_end.Sub(v_start)}
	}
	return OsV4{Start: qA.Start}
}

func (qA OsV4) HasIntersect(qB OsV4) bool {

	q := qA.GetIntersect(qB)
	return q.Is()
}

func OsV4_relativeSurround(src OsV4, dst OsV4, screen OsV4, priorUp bool) OsV4 {

	q := dst
	q.Start = q.Start.Sub(screen.Start)

	srcStart := src.Start.Sub(screen.Start)
	srcSize := src.Size

	up := srcStart.Y > (screen.Size.Y - srcStart.Y - srcSize.Y)
	if !up && priorUp {
		up = srcStart.Y > q.Size.Y //check enough space
	}

	right := (srcStart.X+q.Size.X < screen.Size.X)

	if right {
		q.Start.X = srcStart.X
		if q.Start.X+q.Size.X > screen.Size.X {
			q.Size.X = screen.Size.X - q.Start.X
		}
	} else {
		q.Start.X = srcStart.X + srcSize.X - q.Size.X
		if q.Start.X < 0 {
			q.Size.X = srcStart.X + srcSize.X
			q.Start.X = 0
		}
	}

	if up {
		q.Start.Y = srcStart.Y - q.Size.Y
		if q.Start.Y < 0 {
			q.Size.Y = srcStart.Y
			q.Start.Y = 0
		}
	} else {
		q.Start.Y = srcStart.Y + srcSize.Y

		if q.Start.Y+q.Size.Y > screen.Size.Y {
			q.Size.Y = screen.Size.Y - q.Start.Y
		}
	}

	q.Start = q.Start.Add(screen.Start)
	return q
}

func (v *OsV4) RelativePos(p OsV2) OsV2f {
	s := p.Sub(v.Start)
	return OsV2f{float32(s.X) / float32(v.Size.X), float32(s.Y) / float32(v.Size.Y)}
}

func (v *OsV4) Relative(q OsV4) (x, y, w, h float32) {
	s := v.RelativePos(q.Start)
	e := v.RelativePos(q.End())
	return s.X, s.Y, (e.X - s.X), (e.Y - s.Y)
}

func (v OsV4) Cut(x, y, w, h float64) OsV4 {

	return InitOsV4(
		v.Start.X+int(float64(v.Size.X)*x),
		v.Start.Y+int(float64(v.Size.Y)*y),
		int(float64(v.Size.X)*w),
		int(float64(v.Size.Y)*h))
}

func (v OsV4) CutEx(x, y, w, h float64, space, spaceX, spaceY int) OsV4 {

	v = v.Cut(x, y, w, h)
	v = v.AddSpaceX(spaceX)
	v = v.AddSpaceY(spaceY)
	return v.AddSpace(space)
}

type OsCd struct {
	R byte
	G byte
	B byte
	A byte
}

func InitOsCd32(r, g, b, a uint32) OsCd {
	return OsCd{byte(r), byte(g), byte(b), byte(a)}
}
func InitOsCdWhite() OsCd {
	return InitOsCd32(255, 255, 255, 255)
}

func (cd OsCd) SetAlpha(a byte) OsCd {
	cd.A = a
	return cd
}

func (cd OsCd) MultAlpha(a byte) OsCd {
	return cd.Aprox(InitOsCdWhite(), float32(a)/255.0)
}
func (a OsCd) Cmp(b OsCd) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}

func (s OsCd) Aprox(e OsCd, t float32) OsCd {
	var self OsCd
	self.R = byte(float32(s.R) + (float32(e.R)-float32(s.R))*t)
	self.G = byte(float32(s.G) + (float32(e.G)-float32(s.G))*t)
	self.B = byte(float32(s.B) + (float32(e.B)-float32(s.B))*t)
	self.A = byte(float32(s.A) + (float32(e.A)-float32(s.A))*t)
	return self
}

type HSL struct {
	H int
	S float64
	L float64
}

func _hueToRGB(v1, v2, vH float64) float64 {
	if vH < 0 {
		vH++
	}
	if vH > 1 {
		vH--
	}

	if (6 * vH) < 1 {
		return v1 + (v2-v1)*6*vH
	} else if (2 * vH) < 1 {
		return v2
	} else if (3 * vH) < 2 {
		return v1 + (v2-v1)*((2.0/3)-vH)*6
	}

	return v1
}

func INTtoRGB(v uint32) OsCd {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)

	return OsCd{R: b[0], G: b[1], B: b[2], A: b[3]}
}
func (rgb OsCd) RGBtoINT() uint32 {
	b := []byte{rgb.R, rgb.G, rgb.B, rgb.A}
	return binary.LittleEndian.Uint32(b)
}

func (hsl HSL) HSLtoRGB() OsCd {
	cd := OsCd{A: 255}

	if hsl.S == 0 {
		ll := hsl.L * 255
		ll = float64(OsClampFloat(float64(ll), 0, 255))

		cd.R = uint8(ll)
		cd.G = uint8(ll)
		cd.B = uint8(ll)
	} else {
		var v2 float64
		if hsl.L < 0.5 {
			v2 = (hsl.L * (1 + hsl.S))
		} else {
			v2 = ((hsl.L + hsl.S) - (hsl.L * hsl.S))
		}
		v1 := 2*hsl.L - v2

		hue := float64(hsl.H) / 360
		cd.R = uint8(255 * _hueToRGB(v1, v2, hue+(1.0/3)))
		cd.G = uint8(255 * _hueToRGB(v1, v2, hue))
		cd.B = uint8(255 * _hueToRGB(v1, v2, hue-(1.0/3)))
	}

	return cd
}

func (cd OsCd) RGBtoHSL() HSL {
	var hsl HSL

	r := float64(cd.R) / 255
	g := float64(cd.G) / 255
	b := float64(cd.B) / 255

	min := OsMinFloat(OsMinFloat(r, g), b)
	max := OsMaxFloat(OsMaxFloat(r, g), b)
	delta := max - min

	hsl.L = (max + min) / 2

	if delta == 0 {
		hsl.H = 0
		hsl.S = 0
	} else {
		if hsl.L <= 0.5 {
			hsl.S = delta / (max + min)
		} else {
			hsl.S = delta / (2 - max - min)
		}

		var hue float64
		if r == max {
			hue = ((g - b) / 6) / delta
		} else if g == max {
			hue = (1.0 / 3) + ((b-r)/6)/delta
		} else {
			hue = (2.0 / 3) + ((r-g)/6)/delta
		}

		if hue < 0 {
			hue += 1
		}
		if hue > 1 {
			hue -= 1
		}

		hsl.H = int(hue * 360)
	}

	return hsl
}

func HEXtoRGBwithCheck(hex string, defaultCd OsCd) OsCd {
	if len(hex) == 6 || (len(hex) == 7 && hex[0] == '#') {
		return HEXtoRGB(hex)
	}
	return defaultCd
}

func HEXtoRGB(hex string) OsCd {
	cd := OsCd{A: 255}

	if len(hex) == 0 {
		return cd
	}

	if hex[0] == '#' {
		hex = hex[1:] //skip
	}

	if len(hex) < 2 {
		return cd
	}
	r, _ := strconv.ParseInt(hex[:2], 16, 16)
	cd.R = uint8(r)
	hex = hex[2:]

	if len(hex) < 2 {
		return cd
	}
	g, _ := strconv.ParseInt(hex[:2], 16, 16)
	cd.G = uint8(g)
	hex = hex[2:]

	if len(hex) < 2 {
		return cd
	}
	b, _ := strconv.ParseInt(hex[:2], 16, 16)
	cd.B = uint8(b)

	return cd
}

func (cd OsCd) RGBtoHEX() string {
	return fmt.Sprintf("#%02x%02x%02x", cd.R, cd.G, cd.B)
}

func OsFileGetNameWithoutExt(fileName string) string {

	fileName = filepath.Base(fileName)
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	return fileName
}

func OsFileExists(fileName string) bool {
	info, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func OsFolderExists(fileName string) bool {
	info, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func OsFolderCreate(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func OsFolderRemove(path string) error {
	return os.RemoveAll(path)
}
func OsFileRemove(path string) error {
	return os.Remove(path)
}
func OsFileRename(path string, newPath string) error {
	return os.Rename(path, newPath)
}

func OsFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return -1, err
	}
	return info.Size(), nil
}

func OsFileCopy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

const g_special = "ěščřžťďňĚŠČŘŽŤĎŇáíýéóúůÁÍÝÉÓÚŮÄäÖöÜüẞß"

func OsIsTextWord(ch rune) bool {
	return ((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || (ch == '_')) || strings.ContainsRune(g_special, ch)
}

func OsUlit_OpenBrowser(url string) error {

	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

func OsUlit_GetUID() (string, error) {

	device, err := os.Hostname()
	if err != nil {
		return "", err
	}

	h, err := InitOsHash([]byte(device))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.h), nil
}

const OsHash_SIZE = 32

type OsHash struct {
	h [OsHash_SIZE]byte
}

func InitOsHash(src []byte) (OsHash, error) {

	if len(src) == 0 {
		return OsHash{}, nil //zeros
	}

	h := sha256.New()
	_, err := h.Write(src)
	if err != nil {
		return OsHash{}, err
	}

	return OsHash{h: [OsHash_SIZE]byte(h.Sum(nil))}, nil
}

func InitOsHashCopy(src_hash []byte) OsHash {
	var h OsHash
	if len(src_hash) == OsHash_SIZE {
		copy(h.h[:], src_hash[:])
	}
	//else empty
	return h
}

func (a *OsHash) Cmp(b *OsHash) bool {
	return bytes.Equal(a.h[:], b.h[:])
}

func (a OsHash) CmpBytes(b []byte) bool {
	if len(b) == OsHash_SIZE {
		return bytes.Equal(a.h[:], b)
	}

	return a.Cmp(&OsHash{}) //is empty
}

func (h *OsHash) GetInt64() int64 {
	return int64(binary.LittleEndian.Uint64(h.h[:]))
}

type OsFileList struct {
	Name  string
	IsDir bool
	Subs  []OsFileList
}

func OsFileListBuild(path string, name string, isDir bool) OsFileList {
	var fl OsFileList
	fl.Name = name
	fl.IsDir = isDir

	if isDir {
		dir, err := os.ReadDir(path)
		if err == nil {
			for _, file := range dir {
				fl.Subs = append(fl.Subs, OsFileListBuild(path+"/"+file.Name(), file.Name(), file.IsDir()))
			}
		}
	}
	return fl
}

func (fl *OsFileList) FindInSubs(name string, isDir bool) int {
	for i, f := range fl.Subs {
		if f.Name == name && f.IsDir == isDir {
			return i
		}
	}
	return -1
}

func OsText_RawToPrint(str string) string {

	v := strings.Clone(str)

	v = strings.ReplaceAll(v, "\\n", "\n")
	v = strings.ReplaceAll(v, "\\t", "\t")
	v = strings.ReplaceAll(v, "\\\"", "\"")

	return v
}

func OsText_PrintToRaw(str string) string {

	v := strings.Clone(str)

	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\t", "\\t")
	v = strings.ReplaceAll(v, "\"", "\\\"")

	return v
}

func Os_StartProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	pprof.StartCPUProfile(f)
	return nil
}
func Os_StopProfile() {
	pprof.StopCPUProfile()
}

type OsBlob struct {
	data []byte
	hash OsHash
}

func InitOsBlob(blob []byte) OsBlob {
	var b OsBlob
	b.data = blob
	b.hash, _ = InitOsHash(blob) //err ...
	return b
}

func (b *OsBlob) Len() int {
	return len(b.data)
}

func (a *OsBlob) CmpHash(b *OsBlob) bool {
	return a.hash.Cmp(&b.hash)
}

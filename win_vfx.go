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
	"math"
	"math/rand"
)

const Win_SKYALT_LOGO = "apps/base/resources/logo.png"

type WinParticles struct {
	win *Win

	poses  []OsV2f
	vels   []OsV2f
	alphas []float32
	num    int
	time   float64
	noiseX *WinNoise
	noiseY *WinNoise

	num_draw int

	logo *WinTexture
	img  image.Image

	anim_max_time float32 // zero = deactivated
	anim_act_time float32

	done    float32
	oldDone float32
}

func NewWinParticles(win *Win, time_sec float32) (*WinParticles, error) {
	var ptcs WinParticles
	ptcs.win = win

	var err error
	ptcs.logo, ptcs.img, err = InitWinTextureFromFile(Win_SKYALT_LOGO)
	if err != nil {
		return nil, fmt.Errorf("CreateTextureFromImage() failed: %w", err)
	}

	ptcs.noiseX = NewWinNoise(ptcs.logo.size)
	ptcs.noiseY = NewWinNoise(ptcs.logo.size)

	ptcs.anim_max_time = time_sec
	ptcs.anim_act_time = 0
	ptcs.Emit()

	return &ptcs, nil
}

func (ptcs *WinParticles) Destroy() error {
	ptcs.Clear()

	ptcs.logo.Destroy()

	ptcs.noiseX = nil
	ptcs.noiseY = nil
	return nil
}

func (ptcs *WinParticles) Clear() {
	ptcs.poses = nil
	ptcs.vels = nil
	ptcs.alphas = nil
	ptcs.num = 0

	ptcs.time = 0
}

func (ptcs *WinParticles) Tick(win *Win) bool {
	ptcs.Update()
	_, err := ptcs.Draw(InitOsCdWhite(), win)
	if err != nil {
		fmt.Printf("Particles.Draw() failed: %v\n", err)
		return false
	}
	return ptcs.num_draw > 0
}

func (ptcs *WinParticles) GetImageAlpha(x, y int) float32 {
	_, _, _, cdA := ptcs.img.At(x, y).RGBA()
	return float32(cdA >> 8)
}

func (ptcs *WinParticles) Emit() error {
	ptcs.Clear()

	// get num particles
	n := 0
	for y := 0; y < ptcs.logo.size.Y; y++ {
		for x := 0; x < ptcs.logo.size.X; x++ {
			if ptcs.GetImageAlpha(x, y) != 0 { //get alpha
				n++
			}
		}
	}

	const SUBS = 2
	n *= (SUBS * SUBS)

	// alloc
	ptcs.poses = make([]OsV2f, n)
	ptcs.vels = make([]OsV2f, n)
	ptcs.alphas = make([]float32, n)
	ptcs.num = n
	ptcs.time = 0

	st := OsV2{OsAbs(int(rand.Int31() % 255)), OsAbs(int(rand.Int31() % 255))}
	ptcs.noiseX.Draw(st, 35, 6, 0.5)
	st = st.Add(ptcs.logo.size)
	ptcs.noiseY.Draw(st, 35, 6, 0.5)

	// set data
	n = 0
	for y := 0; y < ptcs.logo.size.Y; y++ {
		for x := 0; x < ptcs.logo.size.X; x++ {
			cdA := ptcs.GetImageAlpha(x, y)
			if cdA != 0 {
				for i := 0; i < SUBS; i++ {
					for j := 0; j < SUBS; j++ {
						ptcs.poses[n] = OsV2f{float32(x) + 1.0/float32(SUBS)*float32(i), float32(y) + 1.0/float32(SUBS)*float32(j)}
						ptcs.vels[n] = OsV2f{0, 0}
						ptcs.alphas[n] = float32(cdA) / float32(-255.0) // negative = not active
						n++
					}
				}
			}
		}
	}
	ptcs.num_draw = n

	return nil
}

func (ptcs *WinParticles) UpdateTime() float64 {
	t := OsTime()
	dt := t - ptcs.time
	ptcs.time = t
	if dt > 0.04 { // first tick
		dt = 0.04 // at least 25fps
	}
	return dt
}

func (ptcs *WinParticles) UpdateDone(DT float32) float32 {

	if ptcs.anim_max_time != 0 {
		ptcs.anim_act_time += DT
		ptcs.done = ptcs.anim_act_time / ptcs.anim_max_time
	}

	if ptcs.done < ptcs.oldDone {
		ptcs.Emit()
	}

	ptcs.oldDone = ptcs.done
	return ptcs.done
}

func (ptcs *WinParticles) GetLogoCoord() (OsV4, error) {
	w, h := ptcs.win.GetOutputSize()

	screen := OsV2{int(w), int(h)}
	SX := float32(screen.X) / 4

	size := OsV2{int(SX), int(SX * float32(ptcs.logo.size.Y) / float32(ptcs.logo.size.X))}
	start := screen.Sub(size).MulV(0.5)

	return OsV4{start, size}, nil
}

func (ptcs *WinParticles) GetPosSmoothRepeat(noise *WinNoise, p OsV2) byte {

	// repeat
	x := OsAbs(p.X) % ptcs.logo.size.X
	y := OsAbs(p.Y) % ptcs.logo.size.Y

	// smooth - revers odd
	if (p.X/ptcs.logo.size.X)%2 != 0 {
		x = ptcs.logo.size.X - x - 1
	}
	if (p.Y/ptcs.logo.size.Y)%2 != 0 {
		y = ptcs.logo.size.Y - y - 1
	}

	return noise.noise[y*ptcs.logo.size.X+x]
}

func (ptcs *WinParticles) Draw(cd_theme OsCd, win *Win) (bool, error) {

	front_cd := OsCd{50, 50, 50, 255}

	coord, err := ptcs.GetLogoCoord()
	if err != nil {
		return false, fmt.Errorf("Draw() GetLogoCoord() failed: %w", err)
	}

	ptcs.logo.DrawQuad(coord, 0, cd_theme)
	{
		w := OsMaxFloat32(0, (1 - ptcs.oldDone))
		cutCoord := coord
		cutCoord.Size.X = int(float32(cutCoord.Size.X) * w)
		ptcs.logo.DrawQuadUV(cutCoord, 0, front_cd, OsV2f{0, 0}, OsV2f{w, 1})
	}

	if ptcs.num_draw == 0 {
		return false, nil
	}

	ratio := OsV2f{float32(coord.Size.X) / float32(ptcs.logo.size.X), float32(coord.Size.Y) / float32(ptcs.logo.size.Y)}
	last_p := OsV2{0, 0}

	win.DrawPointStart()
	for i := 0; i < ptcs.num; i++ {

		a := OsFAbs(ptcs.alphas[i])
		if ptcs.alphas[i] > 0 && a > 0.01 && !last_p.Cmp(ptcs.poses[i].toV2()) { // one particles per pixel(in row)
			var p OsV2f
			p.X = float32(coord.Start.X) + ptcs.poses[i].X*ratio.X
			p.Y = float32(coord.Start.Y) + ptcs.poses[i].Y*ratio.Y

			front_cd.A = uint8(a * 255)
			win.DrawPointCdF(p, 1, front_cd)

			last_p = ptcs.poses[i].toV2()
		}
	}
	win.DrawPointEnd()

	return ptcs.num_draw > 0, nil
}

func (ptcs *WinParticles) Update() {
	DT := float32(ptcs.UpdateTime())
	FADE := DT * 0.4

	done := ptcs.UpdateDone(DT)
	edge := float32(ptcs.logo.size.X) * (1.0 - done)

	if int(edge) == ptcs.logo.size.X { // edge is on right side = nothing to simulate
		return
	}

	ptcs.num_draw = 0
	for i := 0; i < ptcs.num; i++ {
		p := &ptcs.poses[i]
		alp := &ptcs.alphas[i]

		if p.X > edge {
			*alp = OsFAbs(*alp) // activate
		}

		if *alp > 0 {
			v := &ptcs.vels[i]

			ppp := p.toV2()
			acc := OsV2f{float32(ptcs.GetPosSmoothRepeat(ptcs.noiseX, ppp)) - 128, float32(ptcs.GetPosSmoothRepeat(ptcs.noiseY, ppp)) - 128}
			acc = acc.MulV(2.5)
			//acc.x *= 2.0 //boost ->

			*v = v.Add(acc.MulV(DT)) // v += acc * dt
			*p = p.Add(v.MulV(DT))   // p += v * dt

			*alp -= FADE
			if *alp < 0 {
				*alp = 0
			}
		}

		if *alp != 0 {
			ptcs.num_draw++
		}
	}

	if ptcs.num_draw == 0 { // done
		ptcs.Clear()
	}
}

type WinNoise struct {
	noise []byte
	size  OsV2
}

func NewWinNoise(size OsV2) *WinNoise {
	var self WinNoise
	self.size = size
	self.noise = make([]byte, size.X*size.Y)
	return &self
}

func (ns *WinNoise) Draw(offset OsV2, zoom int, octaves int, p float32) {
	for y := 0; y < ns.size.Y; y++ {
		for x := 0; x < ns.size.X; x++ {
			getnoise := float32(0)
			for i := 0; i < octaves-1; i++ {
				frequency := float32(math.Pow(2.0, float64(i)))
				amplitude := float32(math.Pow(float64(p), float64(i)))
				getnoise += _WinNoise_get(float32(x+offset.X)*frequency/float32(zoom), float32(y+offset.Y)/float32(zoom)*frequency) * amplitude
			}
			getnoise = (getnoise + 1) * 0.5
			if getnoise < 0 {
				getnoise = 0
			}
			if getnoise > 1 {
				getnoise = 1
			}

			ns.noise[y*ns.size.X+x] = byte(255.0 * getnoise)
		}
	}
}

func _WinNoise_find(x float32, y float32) float32 {
	n := int(x + y*57)
	n = (n << 13) ^ n
	nn := (n*(n*n*60493+19990303) + 1376312589) & 0x7fffffff
	return 1.0 - (float32(nn) / 1073741824.0)
}

func _WinNoise_interpolate(a float32, b float32, x float32) float32 {
	ft := float32(x) * 3.1415927
	f := float32(1.0-math.Cos(float64(ft))) * 0.5
	return a*(1.0-f) + b*f
}

func _WinNoise_get(x float32, y float32) float32 {

	floorx := float32(int(x))
	floory := float32(int(y))

	s := _WinNoise_find(floorx, floory)
	t := _WinNoise_find(floorx+1, floory)
	u := _WinNoise_find(floorx, floory+1)
	v := _WinNoise_find(floorx+1, floory+1)

	i1 := _WinNoise_interpolate(s, t, x-floorx)
	i2 := _WinNoise_interpolate(u, v, x-floorx)
	return _WinNoise_interpolate(i1, i2, y-floory)
}

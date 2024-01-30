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

//This code is from: https://github.com/zeozeozeo/gomplerate/blob/main/resample.go
//Changed: float64 -> float32
//Changed: remove 'resampler.Channels'

func OsAudio_resample(srcSample, dstSamples int, data []float32) []float32 {
	// Need at least 16 samples to resample a channel
	if len(data) <= 16 {
		return make([]float32, len(data))
	}

	// The samples we can use to resample
	availSamples := len(data) - 16

	// The resample step between new samples
	channelFrom := float32(srcSample) // / float32(resampler.Channels)
	channelTo := float32(dstSamples)  // / float32(resampler.Channels)
	step := channelFrom / channelTo

	output := []float32{}

	// Resample each position from x0
	for x := step; x < float32(availSamples); x += step {
		xi0 := float32(uint64(x))
		xi := []float32{xi0, xi0 + 1, xi0 + 2, xi0 + 3}
		yi0 := uint64(xi0)
		yi := []float32{
			float32(data[yi0]),
			float32(data[yi0+1]),
			float32(data[yi0+2]),
			float32(data[yi0+3]),
		}
		xo := []float32{x}
		yo := []float32{0.0}
		if err := spline(xi, yi, xo, yo); err != nil {
			return data[:]
		}

		output = append(output, yo[0]/float32(0x7FFF))
	}
	return output
}

func spline(xi, yi, xo, yo []float32) (err error) {
	if len(xi) != 4 {
		return fmt.Errorf("invalid xi")
	}
	if len(yi) != 4 {
		return fmt.Errorf("invalid yi")
	}
	if len(xo) == 0 {
		return fmt.Errorf("invalid xo")
	}
	if len(yo) != len(xo) {
		return fmt.Errorf("invalid yo")
	}

	x0, x1, x2, x3 := xi[0], xi[1], xi[2], xi[3]
	y0, y1, y2, y3 := yi[0], yi[1], yi[2], yi[3]
	h0, h1, h2, _, u1, l2, _ := splineLU(xi)
	c1, c2 := splineC1(yi, h0, h1), splineC2(yi, h1, h2)
	m1, m2 := splineM1(c1, c2, u1, l2), splineM2(c1, c2, u1, l2) // m0=m3=0

	for k, v := range xo {
		if v <= x1 {
			yo[k] = splineZ0(m1, h0, x0, x1, y0, y1, v)
		} else if v <= x2 {
			yo[k] = splineZ1(m1, m2, h1, x1, x2, y1, y2, v)
		} else {
			yo[k] = splineZ2(m2, h2, x2, x3, y2, y3, v)
		}
	}

	return
}

func splineZ0(m1, h0, x0, x1, y0, y1, x float32) float32 {
	v0 := float32(0.0)
	v1 := (x - x0) * (x - x0) * (x - x0) * m1 / (6 * h0)
	v2 := -1.0 * y0 * (x - x1) / h0
	v3 := (y1 - h0*h0*m1/6) * (x - x0) / h0
	return v0 + v1 + v2 + v3
}

func splineZ1(m1, m2, h1, x1, x2, y1, y2, x float32) float32 {
	v0 := -1.0 * (x - x2) * (x - x2) * (x - x2) * m1 / (6 * h1)
	v1 := (x - x1) * (x - x1) * (x - x1) * m2 / (6 * h1)
	v2 := -1.0 * (y1 - h1*h1*m1/6) * (x - x2) / h1
	v3 := (y2 - h1*h1*m2/6) * (x - x1) / h1
	return v0 + v1 + v2 + v3
}

func splineZ2(m2, h2, x2, x3, y2, y3, x float32) float32 {
	v0 := -1.0 * (x - x3) * (x - x3) * (x - x3) * m2 / (6 * h2)
	v1 := float32(0.0)
	v2 := -1.0 * (y2 - h2*h2*m2/6) * (x - x3) / h2
	v3 := y3 * (x - x2) / h2
	return v0 + v1 + v2 + v3
}

func splineM1(c1, c2, u1, l2 float32) float32 {
	return (c1/u1 - c2/2) / (2/u1 - l2/2)
}

func splineM2(c1, c2, u1, l2 float32) float32 {
	return (c1/2 - c2/l2) / (u1/2 - 2/l2)
}

func splineC1(yi []float32, h0, h1 float32) float32 {
	y0, y1, y2, _ := yi[0], yi[1], yi[2], yi[3]
	return 6.0 / (h0 + h1) * ((y2-y1)/h1 - (y1-y0)/h0)
}

func splineC2(yi []float32, h1, h2 float32) float32 {
	_, y1, y2, y3 := yi[0], yi[1], yi[2], yi[3]
	return 6.0 / (h1 + h2) * ((y3-y2)/h2 - (y2-y1)/h1)
}

func splineLU(xi []float32) (h0, h1, h2, l1, u1, l2, u2 float32) {
	x0, x1, x2, x3 := xi[0], xi[1], xi[2], xi[3]

	h0, h1, h2 = x1-x0, x2-x1, x3-x2

	l1 = h0 / (h1 + h0)
	u1 = h1 / (h1 + h0)

	l2 = h1 / (h2 + h1)
	u2 = h2 / (h2 + h1)

	return
}

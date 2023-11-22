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

type WinStats struct {
	sum_frames int
	sum_time   int

	last_show_time int64

	min_dt int
	max_dt int

	out_worst_fps float64
	out_best_fps  float64
	out_avg_fps   float64
}

func (sts *WinStats) Update(dt int) {

	time := OsTicks()

	sts.sum_time += OsMax(0, dt)

	if dt < sts.min_dt {
		sts.min_dt = dt
	}
	if dt > sts.max_dt {
		sts.max_dt = dt
	}

	if sts.last_show_time+1000 < time { // every 1sec

		sts.out_best_fps = OsTrnFloat(sts.min_dt > 0, 1/(float64(sts.min_dt)/1000.0), 1000)
		sts.out_worst_fps = OsTrnFloat(sts.max_dt > 0, 1/(float64(sts.max_dt)/1000.0), 1000)
		sts.out_avg_fps = OsTrnFloat(sts.sum_time > 0, float64(sts.sum_frames)/(float64(sts.sum_time)/1000.0), 1000)

		sts.sum_frames = 0
		sts.sum_time = 0
		sts.last_show_time = time
		sts.min_dt = 10000
		sts.max_dt = 0
	}

	sts.sum_frames++
}

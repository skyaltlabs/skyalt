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

type UiStat struct {
	sum_frames int
	sum_time   int

	last_show_time int64

	min_dt int
	max_dt int

	out_worst_fps float64
	out_best_fps  float64
	out_avg_fps   float64
}

func (ui *UiStat) Update(dt int) {

	time := OsTicks()

	ui.sum_time += OsMax(0, dt)

	if dt < ui.min_dt {
		ui.min_dt = dt
	}
	if dt > ui.max_dt {
		ui.max_dt = dt
	}

	if ui.last_show_time+1000 < time { // every 1sec

		ui.out_best_fps = OsTrnFloat(ui.min_dt > 0, 1/(float64(ui.min_dt)/1000.0), 1000)
		ui.out_worst_fps = OsTrnFloat(ui.max_dt > 0, 1/(float64(ui.max_dt)/1000.0), 1000)
		ui.out_avg_fps = OsTrnFloat(ui.sum_time > 0, float64(ui.sum_frames)/(float64(ui.sum_time)/1000.0), 1000)

		ui.sum_frames = 0
		ui.sum_time = 0
		ui.last_show_time = time
		ui.min_dt = 10000
		ui.max_dt = 0
	}

	ui.sum_frames++
}

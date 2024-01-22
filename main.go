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
	"time"
)

func main() {
	//Profiling
	//Os_StartProfile("cpu.prof")
	//defer Os_StopProfile()

	InitImageGlobal()

	//SDL
	err := InitSDLGlobal()
	if err != nil {
		fmt.Printf("InitSDLGlobal() failed: %v\n", err)
		return
	}
	defer DestroySDLGlobal()

	//Databases
	err = InitSQLiteGlobal()
	if err != nil {
		fmt.Printf("InitSQLiteGlobal() failed: %v\n", err)
		return
	}

	disk, err := NewDisk("disk")
	if err != nil {
		fmt.Printf("NewDbs() failed: %v\n", err)
		return
	}
	defer disk.Destroy()

	//Window(GL)
	win, err := NewWin(disk)
	if err != nil {
		fmt.Printf("NewUi() failed: %v\n", err)
		return
	}
	defer win.Destroy()

	//UI
	ui, err := NewUi(win, SABase_GetPathLayout())
	if err != nil {
		fmt.Printf("NewUi() failed: %v\n", err)
		return
	}
	defer ui.Destroy()

	//Base app
	base, err := NewSABase(ui)
	if err != nil {
		fmt.Printf("NewSAApp() failed: %v\n", err)
		return
	}
	defer base.Destroy()

	//Main loop
	run := true
	for run {
		disk.net.online = !win.io.ini.Offline //update

		var redraw bool
		run, redraw, err = win.UpdateIO()
		if err != nil {
			fmt.Printf("UpdateIO() failed: %v\n", err)
			return
		}

		if ui.UpdateTile(ui.buff.win) {
			redraw = true
		}

		if redraw {
			win.StartRender(OsCd{220, 220, 220, 255})

			ui.StartRender()
			if !base.Render(ui) {
				run = false
			}
			ui.EndRender()

			win.EndRender(true)
		} else {
			time.Sleep(20 * time.Millisecond)
		}

		disk.Tick()
	}
}

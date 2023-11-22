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

func main() {
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
	dbs, err := NewDbs("databases")
	if err != nil {
		fmt.Printf("NewDbs() failed: %v\n", err)
		return
	}
	defer dbs.Destroy()

	//Window(GL)
	win, err := NewWin()
	if err != nil {
		fmt.Printf("NewUi() failed: %v\n", err)
		return
	}
	defer win.Destroy()

	//Main loop
	run := true
	for run {
		run, _, err = win.UpdateIO()
		if err != nil {
			fmt.Printf("UpdateIO() failed: %v\n", err)
			return
		}

		win.StartRender(OsCd{220, 220, 220, 255})

		dbs.Tick()
		//time.Sleep(2 * time.Millisecond) //render app ...

		win.EndRender(true)
	}
}

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
	"sync"
	"sync/atomic"
)

type SANodeExe struct {
	app  *SAApp
	list []*SANode

	max_threads      int
	numActiveThreads atomic.Int64
	wg               sync.WaitGroup
}

func NewSANodeExe(app *SAApp, max_threads int) *SANodeExe {
	exe := &SANodeExe{}
	exe.app = app
	exe.max_threads = max_threads

	app.root.buildList(&exe.list)

	app.root.markUnusedAttrs()

	return exe
}

func (exe *SANodeExe) GetStatDone() float64 {

	done := 0.0
	sum := 0.0
	for _, n := range exe.list {
		sum += n.exeTimeSec
		if n.state.Load() != SANode_STATE_DONE {
			done += n.exeTimeSec * n.progress
		} else {
			done += n.exeTimeSec
		}
	}

	return OsTrnFloat(sum > 0, done/sum, 0)
}

func (exe *SANodeExe) Stop() {

	exe.app.base.server.Interrupt() //close connections

	exe.wg.Wait()
}

func (exe *SANodeExe) Tick(app *SAApp) bool {

	active := false

	for _, it := range exe.list {

		if it.state.Load() == SANode_STATE_WAITING {
			active = true

			if it.IsReadyToBeExe() {
				if !it.Bypass && (!app.IDE || app.EnableExecution) { //ignore in releaseMode

					it.ExecuteGui(false)

					//maximum concurent threads
					if exe.numActiveThreads.Load() >= int64(exe.max_threads) {
						return true
					}

					//run it
					it.state.Store(SANode_STATE_RUNNING)
					exe.wg.Add(1)
					go func(ww *SANode) {
						exe.numActiveThreads.Add(1)
						ww.Execute()
						exe.wg.Done()
						exe.numActiveThreads.Add(-1)

						ww.state.Store(SANode_STATE_DONE)
					}(it)
				} else {
					it.state.Store(SANode_STATE_DONE) //done
				}
			}
		}
	}

	//done
	//if !active {
	//app.root.removeUnusedAttrs()
	//}

	return active
}

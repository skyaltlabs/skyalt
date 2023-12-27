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
	list []*SANode

	max_threads      int
	numActiveThreads atomic.Int64
	wg               sync.WaitGroup
}

func NewSANodeExe(root *SANode, max_threads int) *SANodeExe {
	exe := &SANodeExe{}
	exe.max_threads = max_threads

	root.buildList(&exe.list)

	return exe
}

func (exe *SANodeExe) Stop() {

	//close connections
	for _, it := range exe.list {
		if it.conn != nil {
			it.conn.Destroy()
			it.conn = nil
		}
	}

	exe.wg.Wait()
}

func (exe *SANodeExe) Tick(server *SANodeServer) bool {

	active := false

	for _, it := range exe.list {

		if it.state.Load() == SANode_STATE_WAITING {
			active = true

			if it.IsReadyToBeExe() {
				if it.IsExe() {
					//maximum concurent threads
					if exe.numActiveThreads.Load() >= int64(exe.max_threads) {
						return true
					}

					if it.conn == nil {
						it.conn = server.Start(it.Exe)
					}

					//run it
					it.state.Store(SANode_STATE_RUNNING)
					exe.wg.Add(1)
					go func(ww *SANode) {
						exe.numActiveThreads.Add(1)
						ww.execute()
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

	return active
}

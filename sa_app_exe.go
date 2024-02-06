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
	"sync/atomic"
	"time"
)

type SAAppExe struct {
	app  *SAApp
	todo []*SANode

	wip  *SANode
	exit atomic.Bool
	done atomic.Bool

	time float64

	setAttrs []SASetAttr
}

func NewSAAppExe(app *SAApp) *SAAppExe {
	exe := &SAAppExe{}
	exe.app = app
	return exe
}
func (exe *SAAppExe) Destroy() {
	if exe.wip != nil {
		exe.exit.Store(true)

		//wait
		fmt.Print("Waiting for SAAppExe thread to close ...")
		for !exe.done.Load() {
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Println("done")
	}
}

func (exe *SAAppExe) AddSetAttr(attr *SANodeAttr, value string) {
	exe.setAttrs = append(exe.setAttrs, SASetAttr{attr: attr, value: value})
	//exe.app.SetExecute()
}

func (exe *SAAppExe) Add(src *SANode) error {

	dst, err := src.Copy()
	if err != nil {
		return err
	}

	exe.todo = append(exe.todo, dst)

	return nil
}

func (exe *SAAppExe) Tick() *SANode {

	var doneNode *SANode

	if exe.wip != nil {
		if exe.done.Load() { //is finished
			doneNode = exe.wip
			exe.wip = nil
		}
	}

	if exe.wip == nil && len(exe.todo) > 0 {
		exe.wip = exe.todo[0]
		exe.todo = exe.todo[1:]

		exe.done.Store(false)
		go exe.run() //2nd thread

		doneNode = nil //don't return it, because there is newer one
	}

	return doneNode
}

func (exe *SAAppExe) run() {
	fmt.Println("------------")
	st := OsTime()

	exe.wip.PrepareExe() //.state = WAITING(to be executed)

	exe.wip.ParseExpresions()
	exe.wip.CheckForLoops()

	var list []*SANode
	exe.wip.buildSubList(&list)
	exe.wip.markUnusedAttrs()

	exe.ExecuteList(list)

	exe.wip.PostExe()

	if len(exe.setAttrs) > 0 {
		for _, st := range exe.setAttrs {
			st.attr.SetExpString(st.value, false)
		}
		exe.setAttrs = nil
	}

	exe.time = OsTime() - st
	exe.done.Store(true)
}

func (exe *SAAppExe) ExecuteList(list []*SANode) {

	active := true
	for active && !exe.exit.Load() {
		active = false

		for _, it := range list {

			if it.state.Load() == SANode_STATE_WAITING {
				active = true

				if it.IsReadyToBeExe() {

					//execute expression
					for _, v := range it.Attrs {
						if v.errExp != nil {
							continue
						}
						v.ExecuteExpression()
					}

					if !it.Bypass {
						it.state.Store(SANode_STATE_RUNNING)
						it.Execute()

					}
					it.state.Store(SANode_STATE_DONE) //done
				}
			}
		}
	}
}

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
	app *SAApp

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

func (exe *SAAppExe) UpdateProgress(dst *SANode) {
	if exe.wip == nil {
		return
	}
	dst.UpdateProgress(exe.wip)
}

func (exe *SAAppExe) AddSetAttr(attr *SANodeAttr, value string) {
	exe.setAttrs = append(exe.setAttrs, InitSASetAttr(attr, value))
}

func (exe *SAAppExe) Run(src *SANode) error {

	if exe.wip != nil {
		return fmt.Errorf("busy")
	}

	dst, err := src.Copy()
	if err != nil {
		return err
	}

	exe.wip = dst
	exe.done.Store(false)
	go exe.run() //2nd thread

	return nil
}

func (exe *SAAppExe) Tick() (*SANode, []SASetAttr) {

	var doneNode *SANode
	var doneSets []SASetAttr

	if exe.wip != nil {
		if exe.done.Load() { //is finished
			doneNode = exe.wip
			doneSets = exe.setAttrs
			exe.wip = nil
			exe.setAttrs = nil
		}
	}

	return doneNode, doneSets
}

func (exe *SAAppExe) run() {
	st := OsTime()

	exe.wip.PrepareExe() //.state = WAITING(to be executed)

	exe.wip.ParseExpresions()
	exe.wip.CheckForLoops()

	var list []*SANode
	exe.wip.buildSubList(&list)
	exe.wip.markUnusedAttrs()

	exe.ExecuteList(list)

	exe.wip.PostExe()

	exe.time = OsTime() - st
	exe.done.Store(true)
}

func (exe *SAAppExe) ExecuteList(list []*SANode) {

	active := true
	for active && !exe.exit.Load() {
		active = false

		for _, it := range list {

			if it.state == SANode_STATE_WAITING {
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
						it.state = SANode_STATE_RUNNING
						it.Execute()

					}
					it.state = SANode_STATE_DONE //done
				}
			}
		}
	}

	fmt.Printf("Executed() done\n")
}

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
	"encoding/json"
	"fmt"
	"os"
)

type UiLayoutAppItemResize struct {
	Name  string
	Value float32
}

type UiLayoutAppItem struct {
	Hash                   uint64
	ScrollVpos, ScrollHpos int
	Cols_resize            []UiLayoutAppItemResize
	Rows_resize            []UiLayoutAppItemResize
}

type UiLayoutApp struct {
	items []*UiLayoutAppItem

	name string //only for print logs
	logs []string
}

func NewUiLayoutApp(name string, path string) *UiLayoutApp {
	var app UiLayoutApp
	app.name = name

	//load from file
	{
		js, err := os.ReadFile(path)
		if err == nil {
			err = app.Open(js)
			if err != nil {
				fmt.Printf("warnning: Open() failed: %v\n", err)
			}
		}
	}

	return &app
}
func (app *UiLayoutApp) Destroy() {

}
func (app *UiLayoutApp) AddLogErr(err error) bool {
	if err != nil {
		//print
		fmt.Printf("Error(%s): %v\n", app.name, err)

		//add
		app.logs = append(app.logs, err.Error())
		return true
	}
	return false
}

func (app *UiLayoutApp) Open(js []byte) error {
	//extract
	err := json.Unmarshal(js, &app.items)
	if err != nil {
		return fmt.Errorf("Unmarshal() failed: %w", err)
	}

	return nil
}

func (app *UiLayoutApp) Save() ([]byte, error) {
	file, err := json.MarshalIndent(&app.items, "", "")
	if err != nil {
		return nil, fmt.Errorf("MarshalIndent() failed: %w", err)
	}
	return file, nil
}

func (app *UiLayoutApp) FindGlobalScrollHash(hash uint64) *UiLayoutAppItem {
	if app == nil {
		return nil
	}

	for _, it := range app.items {
		if it.Hash == hash {
			return it
		}
	}

	return nil
}

func (app *UiLayoutApp) AddGlobalScrollHash(hash uint64) *UiLayoutAppItem {
	if app == nil {
		return nil
	}

	sc := app.FindGlobalScrollHash(hash)
	if sc != nil {
		return sc
	}

	nw := &UiLayoutAppItem{Hash: hash}
	app.items = append(app.items, nw)
	return nw
}

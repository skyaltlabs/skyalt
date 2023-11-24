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

type UiLayoutAppBase struct {
	Hostname string
	Items    []*UiLayoutAppItem
}

type UiLayoutApp struct {
	settings UiLayoutAppBase //this device

	name string //only for print logs
	logs []string
}

func NewUiLayoutApp(name string, js []byte) (*UiLayoutApp, error) {
	var app UiLayoutApp
	app.name = name

	hostname, bases, found_i, err := LayoutApp_parseBases(js)
	if err != nil {
		return nil, fmt.Errorf("LayoutApp_parseBases() failed: %w", err)
	}

	if found_i < 0 {
		//new
		app.settings.Hostname = hostname
	} else {
		//copy
		app.settings = bases[found_i]
	}

	return &app, nil
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

func LayoutApp_parseBases(js []byte) (string, []UiLayoutAppBase, int, error) {

	var bases []UiLayoutAppBase
	found_i := -1

	hostname, err := os.Hostname()
	if err != nil {
		return "", nil, -1, fmt.Errorf("Hostname() failed: %w", err)
	}

	if len(js) > 0 {
		//extract
		err := json.Unmarshal(js, &bases)
		if err != nil {
			return "", nil, -1, fmt.Errorf("Unmarshal() failed: %w", err)
		}

		//find
		for i, b := range bases {
			if b.Hostname == hostname {
				found_i = i
				break
			}
		}
	}

	return hostname, bases, found_i, nil
}

func (app *UiLayoutApp) Save(js []byte) ([]byte, error) {

	_, bases, found_i, err := LayoutApp_parseBases(js)
	if err != nil {
		return nil, fmt.Errorf("LayoutApp_parseBases() failed: %w", err)
	}

	if found_i < 0 {
		//new
		bases = append(bases, app.settings)
	} else {
		//copy
		bases[found_i] = app.settings
	}

	file, err := json.MarshalIndent(&bases, "", "")
	if err != nil {
		return nil, fmt.Errorf("MarshalIndent() failed: %w", err)
	}
	return file, nil
}

func (app *UiLayoutApp) FindGlobalScrollHash(hash uint64) *UiLayoutAppItem {
	if app == nil {
		return nil
	}

	for _, it := range app.settings.Items {
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
	app.settings.Items = append(app.settings.Items, nw)
	return nw
}

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
	"os"
	"path/filepath"
)

func (ui *Ui) comp_dirPicker(x, y, w, h int, path *string, pathTemp *string, selectFile bool, dialogName string, enable bool) bool {
	origPath := *path

	ui.Div_start(x, y, w, h)
	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	if ui.Comp_buttonError(0, 0, 1, 1, *path, "Select file/folder", !OsFileExists(*path), enable) > 0 {
		ui.Dialog_open(dialogName, 1)
		*pathTemp = *path
	}
	ui.Div_end()

	dialogOpen := ui.Dialog_start(dialogName)
	if dialogOpen {
		if ui.comp_dir(pathTemp) {
			*path = *pathTemp
		}

		ui.Dialog_end()
	}

	return origPath != *path
}

func (ui *Ui) comp_dir(path *string) bool {
	ok := false

	directory := filepath.Dir(*path)

	if !OsFolderExists(directory) {
		//get skyalt dir
		ex, err := os.Executable()
		if err == nil {
			directory = filepath.Dir(ex)
			*path = directory + "/"
		}
	}

	ui.Div_colMax(0, 15)
	ui.Div_rowMax(1, 10)

	//header
	ui.Div_start(0, 0, 1, 1)
	{
		ui.Div_colMax(2, 15)
		ui.Div_colMax(3, 3)

		//home
		if ui.Comp_buttonIcon(0, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/home.png"), 0.3, "Home directory", CdPalette_P, true, false) > 0 {
			dir, err := os.UserHomeDir()
			if err == nil {
				directory = dir
				*path = directory + "/"
			}
		}

		//level-up
		if ui.Comp_buttonIcon(1, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/levelup.png"), 0.3, "Jump into parent directory", CdPalette_P, true, false) > 0 {
			directory = filepath.Dir(directory)
			*path = directory + "/"
		}

		//path
		ui.Comp_editbox(2, 0, 1, 1, path, 0, 0, nil, "", false, false, false, true)

		//open
		if ui.Comp_button(3, 0, 1, 1, "Select", "", true) > 0 {
			ok = true
			ui.Dialog_close()
		}
	}
	ui.Div_end()

	//list
	ui.Div_start(0, 1, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(1, 3)
		ui.Div_colMax(2, 3)

		dir, err := os.ReadDir(directory)
		if err != nil {
			return false
		}

		for i, f := range dir {
			isDir := f.IsDir()

			iconFile := OsTrnString(isDir, "folder.png", "file.png")
			inf, _ := f.Info()

			selected := (directory + "/" + f.Name()) == *path

			if ui.Comp_buttonMenuIcon(0, i, 1, 1, f.Name(), InitWinMedia_url("file:apps/base/resources/"+iconFile), 0.2, "", true, selected) > 0 {
				if isDir {
					directory = directory + "/" + f.Name()
					*path = directory + "/"
				} else {
					*path = directory + "/" + f.Name()
				}
			}

			ui.Comp_text(1, i, 1, 1, fmt.Sprintf("%d Bytes", inf.Size()), 0)
			ui.Comp_text(2, i, 1, 1, ui.GetTextDateTime(inf.ModTime().Unix()), 0)
		}

	}
	ui.Div_end()

	//bottom
	ui.Div_start(0, 2, 1, 1)
	{
		ui.Div_colMax(0, 3)
		ui.Div_colMax(1, 3)
		ui.Div_colMax(2, 100)

		//create file
		if ui.Comp_button(0, 0, 1, 1, "Create File", "", true) > 0 {
			//os.Create()
			//...
		}

		//create folder
		if ui.Comp_button(1, 0, 1, 1, "Create Folder", "", true) > 0 {
			//...
		}

		//search
		var search string
		ui.Comp_editbox(2, 0, 1, 1, search, 0, 0, nil, "Search", false, false, false, true) //highlight ...
	}
	ui.Div_end()

	return ok
}

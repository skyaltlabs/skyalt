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

func (ui *Ui) comp_dirPicker(x, y, w, h int, path *string, selectFile bool, enable bool) bool {
	origPath := *path

	ui.Div_start(x, y, w, h)
	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	exist := OsTrnBool(selectFile, OsFileExists(*path), OsFolderExists(*path))

	if ui.Comp_buttonError(0, 0, 1, 1, *path, "Select file/folder", !exist, enable) > 0 {
		ui.Dialog_open("dir_picker", 1)
		ui.dir = UiDir{tempPath: *path} //reset
	}
	ui.Div_end()

	dialogOpen := ui.Dialog_start("dir_picker")
	if dialogOpen {
		if ui.comp_dir(selectFile) {
			*path = ui.dir.tempPath
		}

		ui.Dialog_end()
	}

	return origPath != *path
}

func (ui *Ui) comp_dir(selectFile bool) bool {
	ok := false

	directory := filepath.Dir(ui.dir.tempPath)

	if !OsFolderExists(directory) {
		//get skyalt dir
		ex, err := os.Executable()
		if err == nil {
			directory = filepath.Dir(ex)
			ui.dir.tempPath = directory + "/"
		}
	}

	ui.Div_colMax(0, 15)
	ui.Div_rowMax(1, 10)

	//header
	ui.Div_start(0, 0, 1, 1)
	{
		ui.Div_colMax(3, 15)
		ui.Div_colMax(4, 3)

		//root
		if ui.Comp_buttonText(0, 0, 1, 1, "/", "Root directory", "", true, false) > 0 {
			directory = ""
			ui.dir.tempPath = directory + "/"
		}

		//home
		if ui.Comp_buttonIcon(1, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/home.png"), 0.3, "Home directory", CdPalette_P, true, false) > 0 {
			dir, err := os.UserHomeDir()
			if err == nil {
				directory = dir
				ui.dir.tempPath = directory + "/"
			}
		}

		//level-up
		if ui.Comp_buttonIcon(2, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/levelup.png"), 0.3, "Jump into parent directory", CdPalette_P, directory != "/", false) > 0 {
			directory = filepath.Dir(directory)
			if directory != "/" {
				ui.dir.tempPath = directory + "/"
			}
		}

		//path
		ui.Comp_editbox(3, 0, 1, 1, &ui.dir.tempPath, 0, 0, nil, "", false, false, false, true)

		//open
		if ui.Comp_button(4, 0, 1, 1, "Select", "", true) > 0 {
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

		y := 0
		for _, f := range dir {
			isDir := f.IsDir()
			if !selectFile && !isDir {
				continue //skip files when picking folder
			}

			iconFile := OsTrnString(isDir, "folder.png", "file.png")
			inf, _ := f.Info()

			selected := (directory + "/" + f.Name()) == ui.dir.tempPath

			if ui.Comp_buttonMenuIcon(0, y, 1, 1, f.Name(), InitWinMedia_url("file:apps/base/resources/"+iconFile), 0.2, "", true, selected) > 0 {
				if isDir {
					if directory != "/" {
						directory += "/"
					}
					directory += f.Name()
					ui.dir.tempPath = directory + "/"
				} else {
					if directory != "/" {
						directory += "/"
					}
					ui.dir.tempPath = directory + f.Name()
				}
			}

			ui.Comp_text(1, y, 1, 1, fmt.Sprintf("%d Bytes", inf.Size()), 0)
			ui.Comp_text(2, y, 1, 1, ui.GetTextDateTime(inf.ModTime().Unix()), 0)
			y++
		}

	}
	ui.Div_end()

	//bottom
	ui.Div_start(0, 2, 1, 1)
	{
		ui.Div_colMax(0, 3)
		ui.Div_colMax(1, 3)
		ui.Div_col(2, 1)
		ui.Div_colMax(3, 100)

		//create file
		if ui.Comp_button(0, 0, 1, 1, "Create File", "", true) > 0 {
			ui.Dialog_open("create_file", 1)
		}

		//create folder
		if ui.Comp_button(1, 0, 1, 1, "Create Folder", "", true) > 0 {
			ui.Dialog_open("create_folder", 1)
		}

		if ui.Dialog_start("create_file") {
			ui.Div_colMax(0, 5)
			ui.Div_colMax(1, 3)

			ui.Comp_editbox(0, 0, 1, 1, &ui.dir.create, 0, 0, nil, "Name", false, true, false, true)

			if ui.Comp_button(1, 0, 1, 1, "Create File", "", ui.dir.create != "") > 0 {
				pt := directory
				if pt != "/" {
					pt += "/"
				}
				pt += ui.dir.create
				f, err := os.Create(pt)
				if err == nil {
					f.Close()
				}
				ui.Dialog_close()
			}

			ui.Dialog_end()
		}

		if ui.Dialog_start("create_folder") {
			ui.Div_colMax(0, 5)
			ui.Div_colMax(1, 3)

			ui.Comp_editbox(0, 0, 1, 1, &ui.dir.create, 0, 0, nil, "Name", false, true, false, true)

			if ui.Comp_button(1, 0, 1, 1, "Create Folder", "", ui.dir.create != "") > 0 {
				pt := directory
				if pt != "/" {
					pt += "/"
				}
				pt += ui.dir.create
				os.Mkdir(pt, os.ModePerm)
				ui.Dialog_close()
			}

			ui.Dialog_end()
		}

		//search
		ui.Comp_editbox(3, 0, 1, 1, &ui.dir.search, 0, 0, nil, "Search", false, false, false, true) //highlight ..............
	}
	ui.Div_end()

	return ok
}

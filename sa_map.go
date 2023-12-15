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
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

//maybe SQL manager? ... images/map/etc. ...

//add:
//- Measure ...
//- Locators ...

// https://wiki.openstreetmap.org/wiki/Raster_tile_providers

type SAMap struct {
	lonOld, latOld, zoomOld float64
	start_pos               OsV2f
	start_tile              OsV2f
	start_zoom_time         float64

	dbs map[string]*sql.DB
}

func NewSAMap() *SAMap {
	mp := &SAMap{}
	mp.dbs = make(map[string]*sql.DB)
	return mp
}

func (mp *SAMap) Destroy() {
	for _, db := range mp.dbs {
		db.Close()
	}
}

func (mp *SAMap) GetDb(path string) (*sql.DB, error) {

	//find
	db, found := mp.dbs[path]

	if !found {
		folder := filepath.Dir(path)
		err := OsFolderCreate(folder)
		if err != nil {
			return nil, fmt.Errorf("OsFolderCreate(%s) failed: %w", folder, err)
		}

		//open
		db, err = sql.Open("sqlite3", "file:"+path)
		if err != nil {
			return nil, fmt.Errorf("sql.Open(%s) failed: %w", path, err)
		}

		_, err = db.Exec("CREATE TABLE IF NOT EXISTS tiles (name TEXT, file BLOB);")
		if err != nil {
			return nil, fmt.Errorf("CREATE TABLE for file(%s) failed: %w", path, err)
		}

		//add
		mp.dbs[path] = db
	}

	return db, nil
}

func MetersPerPixel(lat, zoom float64) float64 {
	return 156543.034 * math.Cos(lat/180*math.Pi) / math.Pow(2, zoom)
}

func LonLatToPos(lon, lat, zoom float64) OsV2f {
	x := (lon + 180) / 360 * math.Pow(2, zoom)
	y := (1 - math.Log(math.Tan(lat*math.Pi/180)+1/math.Cos(lat*math.Pi/180))/math.Pi) / 2 * math.Pow(2, zoom)
	return OsV2f{float32(x), float32(y)}
}

func PosToLonLat(pos OsV2f, zoom float64) (float64, float64) {
	lon := float64(pos.X)/math.Pow(2, zoom)*360 - 180 //long

	n := math.Pi - 2*math.Pi*float64(pos.Y)/math.Pow(2, zoom)
	lat := 180 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(n*-1))) //lat
	return lon, lat
}

func CamBbox(res OsV2f, tile float64, lon, lat, zoom float64) (OsV2f, OsV2f, OsV2f) {
	tilePos := LonLatToPos(lon, lat, zoom)
	max_res := math.Pow(2, zoom)

	var start, end, size OsV2f

	start.X = float32(OsClampFloat((float64(tilePos.X)*tile-float64(res.X)/2)/tile, 0, max_res))
	start.Y = float32(OsClampFloat((float64(tilePos.Y)*tile-float64(res.Y/2))/tile, 0, max_res))
	end.X = float32(OsClampFloat((float64(tilePos.X)*tile+float64(res.X/2))/tile, 0, max_res))
	end.Y = float32(OsClampFloat((float64(tilePos.Y)*tile+float64(res.Y/2))/tile, 0, max_res))

	size.X = end.X - start.X
	size.Y = end.Y - start.Y

	return start, end, size
}

func CamCheck(res OsV2f, tile float64, lon, lat, zoom float64) (float64, float64) {
	if res.X <= 0 || res.Y <= 0 {
		return 0, 0
	}

	bbStart, bbEnd, bbSize := CamBbox(res, tile, lon, lat, zoom)

	maxTiles := math.Pow(2, zoom)

	def_bbox_size := OsV2f{res.X / float32(tile), res.Y / float32(tile)}

	if bbStart.X <= 0 {
		bbSize.X = def_bbox_size.X
		bbStart.X = 0
	}

	if bbStart.Y <= 0 {
		bbSize.Y = def_bbox_size.Y
		bbStart.Y = 0
	}

	if bbEnd.X >= float32(maxTiles) {
		bbSize.X = def_bbox_size.X
		bbStart.X = float32(OsMaxFloat(0, maxTiles-float64(bbSize.X)))
	}

	if bbEnd.Y >= float32(maxTiles) {
		bbSize.Y = def_bbox_size.Y
		bbStart.Y = float32(OsMaxFloat(0, maxTiles-float64(bbSize.Y)))
	}

	return PosToLonLat(OsV2f{bbStart.X + bbSize.X/2, bbStart.Y + bbSize.Y/2}, zoom)
}

func zoomClamp(z float64) float64 {
	return OsClampFloat(z, 0, 19)
}

func (mp *SAMap) isZooming() (bool, float64, float64) {
	ANIM_TIME := 0.4
	dt := OsTime() - mp.start_zoom_time
	return (dt < ANIM_TIME), dt, ANIM_TIME
}

func (mp *SAMap) Render(w *SAWidget, ui *Ui, net *DiskNet) {
	w.errExe = nil

	file := w.GetAttrStringEdit("file", "maps/osm")
	url := w.GetAttrStringEdit("url", "https://tile.openstreetmap.org/{z}/{x}/{y}.png")
	copyright := w.GetAttrStringEdit("copyright", "(c)OpenStreetMap contributors")
	copyright_url := w.GetAttrStringEdit("copyright_url", "https://www.openstreetmap.org/copyright")

	file = "databases/" + file

	cam_lon := w.GetAttrFloatEdit("lon", "14.4071117049")
	cam_lat := w.GetAttrFloatEdit("lat", "50.0852013259")
	cam_zoom := w.GetAttrFloatEdit("zoom", "5")

	cam_zoom = zoomClamp(cam_zoom) //check

	zooming := 0

	lon := cam_lon
	lat := cam_lat
	zoom := cam_zoom

	db, err := mp.GetDb(file)
	if err != nil {
		w.errExe = fmt.Errorf("GetDb(%s) failed: %w", file, err)
		return
	}

	scale := float64(1)
	isZooming, dt, ANIM_TIME := mp.isZooming()
	if isZooming {
		t := dt / ANIM_TIME
		if cam_zoom > mp.zoomOld {
			scale = 1 + t
		} else {
			scale = 1 - t/2
		}
		zoom = mp.zoomOld
		lon = mp.lonOld + (cam_lon-mp.lonOld)*t
		lat = mp.latOld + (cam_lat-mp.latOld)*t
		zooming = 1

		ui.win.SetRedraw()
	}

	cell := ui.DivInfo_get(SA_DIV_GET_cell, 0)
	width := ui.DivInfo_get(SA_DIV_GET_screenWidth, 0)
	height := ui.DivInfo_get(SA_DIV_GET_screenHeight, 0)

	touch_x := float32(ui.DivInfo_get(SA_DIV_GET_touchX, 0))
	touch_y := float32(ui.DivInfo_get(SA_DIV_GET_touchY, 0))
	inside := ui.DivInfo_get(SA_DIV_GET_touchInside, 0) > 0
	active := ui.DivInfo_get(SA_DIV_GET_touchActive, 0) > 0
	end := ui.DivInfo_get(SA_DIV_GET_touchEnd, 0) > 0
	start := ui.DivInfo_get(SA_DIV_GET_touchStart, 0) > 0
	wheel := ui.DivInfo_get(SA_DIV_GET_touchWheel, 0)
	clicks := ui.DivInfo_get(SA_DIV_GET_touchClicks, 0)

	coord := OsV2f{float32(width), float32(height)}

	tile := 256 / cell * scale
	tileW := tile / width
	tileH := tile / height

	CamCheck(coord, tile, cam_lon, cam_lat, cam_zoom)
	bbStart, bbEnd, bbSize := CamBbox(coord, tile, lon, lat, zoom)

	//draw tiles
	for y := float64(int(bbStart.Y)); y < float64(bbEnd.Y); y++ {
		for x := float64(int(bbStart.X)); x < float64(bbEnd.X); x++ {
			if x < 0 || y < 0 {
				continue
			}

			tileCoord_sx := (x - float64(bbStart.X)) * tileW
			tileCoord_sy := (y - float64(bbStart.Y)) * tileH

			name := strconv.Itoa(int(zoom)) + "-" + strconv.Itoa(int(x)) + "-" + strconv.Itoa(int(y)) + ".png"
			row := db.QueryRow("SELECT rowid FROM tiles WHERE name=='" + name + "'")

			rowid := int64(-1)
			err = row.Scan(&rowid)
			if err != nil {

				//download
				u := url
				u = strings.ReplaceAll(u, "{x}", strconv.Itoa(int(x)))
				u = strings.ReplaceAll(u, "{y}", strconv.Itoa(int(y)))
				u = strings.ReplaceAll(u, "{z}", strconv.Itoa(int(zoom)))

				img, done, _, err := net.GetFile(u, "Skyalt/0.1")
				if done {
					if err == nil {
						//insert into db
						res, err := db.Exec("INSERT INTO tiles(name, file) VALUES(?, ?);", name, img)
						if err == nil {
							rowid, err = res.LastInsertId()
							if err != nil {
								w.errExe = fmt.Errorf("LastInsertId() failed: %w", err)
							}
						}
					} else {
						w.errExe = err
					}
				}

			}

			if rowid > 0 {
				file := fmt.Sprintf("blob:%s:tiles/file/%d", file, rowid)

				//extra margin will fix white spaces during zooming
				ui.Paint_file(tileCoord_sx, tileCoord_sy, tileW, tileH, float64(zooming)*-0.03, file, InitOsCd32(255, 255, 255, 255), 0, 0, false)
			}

		}
	}

	//touch
	if start && inside {
		mp.start_pos.X = touch_x //rel, not pixels!
		mp.start_pos.Y = touch_y
		mp.start_tile = LonLatToPos(lon, lat, zoom)
	}

	if wheel != 0 && inside && !isZooming {
		mp.zoomOld = cam_zoom
		cam_zoom = zoomClamp(cam_zoom - wheel)
		if mp.zoomOld != cam_zoom {
			mp.lonOld = cam_lon
			mp.latOld = cam_lat

			//where the mouse is
			if wheel < 0 {
				var pos OsV2f
				pos.X = bbStart.X + bbSize.X*touch_x
				pos.Y = bbStart.Y + bbSize.Y*touch_y
				cam_lon, cam_lat = PosToLonLat(pos, zoom)
			}

			mp.start_zoom_time = OsTime()
		}
	}

	if active {
		var move OsV2f
		move.X = mp.start_pos.X - touch_x
		move.Y = mp.start_pos.Y - touch_y

		rx := move.X * bbSize.X
		ry := move.Y * bbSize.Y

		tileX := mp.start_tile.X + rx
		tileY := mp.start_tile.Y + ry

		cam_lon, cam_lat = PosToLonLat(OsV2f{tileX, tileY}, cam_zoom)
	}

	//double click
	if clicks > 1 && end && !isZooming {
		mp.zoomOld = cam_zoom
		cam_zoom = zoomClamp(cam_zoom + 1)

		if mp.zoomOld != cam_zoom {
			mp.lonOld = cam_lon
			mp.latOld = cam_lat

			var pos OsV2f
			pos.X = bbStart.X + bbSize.X*touch_x
			pos.Y = bbStart.Y + bbSize.Y*touch_y
			cam_lon, cam_lat = PosToLonLat(pos, zoom)

			mp.start_zoom_time = OsTime()
		}
	}

	//set back
	str, edit := w.GetAttrStringPtrEdit("lon", "0")
	if edit {
		*str = strconv.FormatFloat(cam_lon, 'f', -1, 64)
	}
	str, edit = w.GetAttrStringPtrEdit("lat", "0")
	if edit {
		*str = strconv.FormatFloat(cam_lat, 'f', -1, 64)
	}
	str, edit = w.GetAttrStringPtrEdit("zoom", "0")
	if edit {
		*str = strconv.Itoa(int(cam_zoom))
	}

	fmt.Println("out", zoom)

	//copyright
	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)
	ui.Div_colMax(0, 100)
	ui.Div_col(1, 5)
	ui.Div_rowMax(0, 100)
	ui.Div_row(1, 0.5)

	ui.Comp_buttonText(1, 1, 1, 1, copyright, copyright_url, "", true, false)
}

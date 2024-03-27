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
	"math"
	"strconv"
	"strings"
)

//add:
//- Measure ...

// https://wiki.openstreetmap.org/wiki/Raster_tile_providers

type UiLayoutMap struct {
	active *UiLayoutDiv

	lonOld, latOld, zoomOld float64
	start_pos               OsV2f
	start_tile              OsV2f
	start_zoom_time         float64
}

func NewUiLayoutMap() *UiLayoutMap {
	mp := &UiLayoutMap{}
	return mp
}

func (mp *UiLayoutMap) Destroy() {
}

func (mp *UiLayoutMap) SetActive(ui *Ui) {
	mp.active = ui.GetCall().call
}

func (mp *UiLayoutMap) GetAnim(lon float64, lat float64, zoom float64, ui *Ui) (float64, float64, float64, float64, bool) {

	scale := float64(1)
	isZooming, dt, ANIM_TIME := mp.isZooming()
	if isZooming && mp.active == ui.GetCall().call {
		t := dt / ANIM_TIME
		if zoom > mp.zoomOld {
			scale = 1 + t
		} else {
			scale = 1 - t/2
		}
		zoom = mp.zoomOld
		lon = mp.lonOld + (lon-mp.lonOld)*t
		lat = mp.latOld + (lat-mp.latOld)*t

		ui.win.SetRedraw()
	}

	return lon, lat, zoom, scale, isZooming
}

func UiLayoutMap_metersPerPixel(lat, zoom float64) float64 {
	return 156543.034 * math.Cos(lat/180*math.Pi) / math.Pow(2, zoom)
}

func UiLayoutMap_lonLatToPos(lon, lat, zoom float64) OsV2f {
	x := (lon + 180) / 360 * math.Pow(2, zoom)
	y := (1 - math.Log(math.Tan(lat*math.Pi/180)+1/math.Cos(lat*math.Pi/180))/math.Pi) / 2 * math.Pow(2, zoom)
	return OsV2f{float32(x), float32(y)}
}

func UiLayoutMap_posToLonLat(pos OsV2f, zoom float64) (float64, float64) {
	lon := float64(pos.X)/math.Pow(2, zoom)*360 - 180 //long

	n := math.Pi - 2*math.Pi*float64(pos.Y)/math.Pow(2, zoom)
	lat := 180 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(n*-1))) //lat
	return lon, lat
}

func UiLayoutMap_camBbox(res OsV2f, tile float64, lon, lat, zoom float64) (OsV2f, OsV2f, OsV2f) {
	hres := res.MulV(0.5)

	tilePos := UiLayoutMap_lonLatToPos(lon, lat, zoom)
	max_res := math.Pow(2, zoom)

	var start, end OsV2f

	start.X = float32(OsClampFloat((float64(tilePos.X)*tile-float64(hres.X))/tile, 0, max_res))
	start.Y = float32(OsClampFloat((float64(tilePos.Y)*tile-float64(hres.Y))/tile, 0, max_res))
	end.X = float32(OsClampFloat((float64(tilePos.X)*tile+float64(hres.X))/tile, 0, max_res))
	end.Y = float32(OsClampFloat((float64(tilePos.Y)*tile+float64(hres.Y))/tile, 0, max_res))

	size := end.Sub(start)

	return start, end, size
}

func UiLayoutMap_camCheck(res OsV2f, tile float64, lon, lat, zoom float64) (float64, float64) {
	if res.X <= 0 || res.Y <= 0 {
		return 0, 0
	}

	bbStart, bbEnd, bbSize := UiLayoutMap_camBbox(res, tile, lon, lat, zoom)

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

	return UiLayoutMap_posToLonLat(OsV2f{bbStart.X + bbSize.X/2, bbStart.Y + bbSize.Y/2}, zoom)
}

func UiLayoutMap_zoomClamp(z float64) float64 {
	return OsClampFloat(z, 0, 19)
}

func (mp *UiLayoutMap) isZooming() (bool, float64, float64) {
	ANIM_TIME := 0.4
	dt := OsTime() - mp.start_zoom_time
	return (dt < ANIM_TIME), dt, ANIM_TIME
}

type UiCompMapLocator struct {
	Lon   float64
	Lat   float64
	Ele   float64
	Label string
	Time  string
}
type UiCompMapSegments struct {
	Label string
	Trkpt []UiCompMapLocator
}

func (ui *Ui) comp_mapLocators(cam_lon, cam_lat, cam_zoom float64, items []UiCompMapLocator, dialogName string) error {
	cell := ui.DivInfo_get(SA_DIV_GET_cell, 0)
	width := ui.DivInfo_get(SA_DIV_GET_screenWidth, 0)
	height := ui.DivInfo_get(SA_DIV_GET_screenHeight, 0)

	coord := OsV2f{float32(width), float32(height)}

	lon, lat, zoom, scale, _ := ui.mapp.GetAnim(cam_lon, cam_lat, cam_zoom, ui)

	tile := 256 / cell * scale
	tileW := tile / width
	tileH := tile / height

	UiLayoutMap_camCheck(coord, tile, lon, lat, zoom)
	bbStart, _, _ := UiLayoutMap_camBbox(coord, tile, lon, lat, zoom)

	for i, it := range items {
		p := UiLayoutMap_lonLatToPos(it.Lon, it.Lat, zoom) //...

		x := float64(p.X-bbStart.X) * tileW
		y := float64(p.Y-bbStart.Y) * tileH

		rad := 1.0
		rad_x := rad / width
		rad_y := rad / height

		ui.Div_startEx(0, 0, 1, 1, x-rad_x/2, y-rad_y, rad_x, rad_y, strconv.Itoa(i))
		{
			//ui.Paint_file(0, 0, 1, 1, 0, "file:apps/base/resources/locator.png", InitOsCd32(200, 20, 20, 255), 1, 0, false) //red

			dnm := dialogName + "_" + strconv.Itoa(i)
			if ui.Comp_buttonIcon(0, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/locator.png"), 0, it.Label, Comp_buttonProp()) > 0 {
				ui.Dialog_open(dnm, 1)
			}
			if ui.Dialog_start(dnm) {
				ui.Div_colMax(0, 5)
				ui.Comp_text(0, 0, 1, 1, it.Label, 1)
				ui.Comp_text(0, 1, 1, 1, fmt.Sprintf("Lon: %.3f", it.Lon), 0)
				ui.Comp_text(0, 2, 1, 1, fmt.Sprintf("Lat: %.3f", it.Lat), 0)

				ui.Dialog_end()
			}
		}
		ui.Div_end()
	}
	return nil
}

func (ui *Ui) comp_mapSegments(cam_lon, cam_lat, cam_zoom float64, items []UiCompMapSegments) error {
	cell := ui.DivInfo_get(SA_DIV_GET_cell, 0)
	width := ui.DivInfo_get(SA_DIV_GET_screenWidth, 0)
	height := ui.DivInfo_get(SA_DIV_GET_screenHeight, 0)

	coord := OsV2f{float32(width), float32(height)}

	lon, lat, zoom, scale, _ := ui.mapp.GetAnim(cam_lon, cam_lat, cam_zoom, ui)

	tile := 256 / cell * scale
	tileW := tile / width
	tileH := tile / height

	UiLayoutMap_camCheck(coord, tile, lon, lat, zoom)
	bbStart, _, _ := UiLayoutMap_camBbox(coord, tile, lon, lat, zoom)

	rad := 0.2
	rad_x := rad / width
	rad_y := rad / height

	for _, segs := range items {

		last_x := float64(0)
		last_y := float64(0)
		last_set := false

		for _, pt := range segs.Trkpt {
			p := UiLayoutMap_lonLatToPos(pt.Lon, pt.Lat, zoom) //...

			x := float64(p.X-bbStart.X) * tileW
			y := float64(p.Y-bbStart.Y) * tileH

			ui.Paint_circle(0, 0, 1, 1, 0, x, y, 0.1, InitOsCd32(200, 20, 20, 255), 0)

			ui.Paint_tooltip(x-rad_x/2, y-rad_y, rad_x, rad_y, fmt.Sprintf("%.2f, %.2f, %s", pt.Lon, pt.Lat, pt.Time))

			if last_set {
				ui.Paint_line(0, 0, 1, 1, 0, last_x, last_y, x, y, InitOsCd32(200, 20, 20, 255), 0.06)
			}
			last_set = true
			last_x = x
			last_y = y
		}
	}
	return nil
}

func (ui *Ui) comp_map(cam_lon, cam_lat, cam_zoom *float64, file, url, copyright, copyright_url string) (bool, error) {
	old_cam_lon := *cam_lon
	old_cam_lat := *cam_lat
	old_cam_zoom := *cam_zoom

	*cam_zoom = UiLayoutMap_zoomClamp(*cam_zoom) //check

	db, alreadyOpen, err := ui.win.disk.OpenDb(file)
	if err != nil {
		return false, fmt.Errorf("GetDb(%s) failed: %w", file, err)

	}
	if !alreadyOpen {
		_, err = db.Write("CREATE TABLE IF NOT EXISTS tiles (name TEXT, file BLOB);")
		if err != nil {
			return false, fmt.Errorf("CREATE TABLE in db(%s) failed: %w", file, err)
		}
	}

	mp := ui.mapp

	lon, lat, zoom, scale, isZooming := mp.GetAnim(*cam_lon, *cam_lat, *cam_zoom, ui)

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

	canvas_size := OsV2f{float32(width), float32(height)}

	tile := 256 / cell * scale
	tileW := tile / width
	tileH := tile / height

	UiLayoutMap_camCheck(canvas_size, tile, *cam_lon, *cam_lat, *cam_zoom)
	bbStart, bbEnd, bbSize := UiLayoutMap_camBbox(canvas_size, tile, lon, lat, zoom)

	//draw tiles
	for y := float64(int(bbStart.Y)); y < float64(bbEnd.Y); y++ {
		for x := float64(int(bbStart.X)); x < float64(bbEnd.X); x++ {
			if x < 0 || y < 0 {
				continue
			}

			tileCoord_sx := (x - float64(bbStart.X)) * tileW
			tileCoord_sy := (y - float64(bbStart.Y)) * tileH

			name := strconv.Itoa(int(zoom)) + "-" + strconv.Itoa(int(x)) + "-" + strconv.Itoa(int(y)) + ".png"
			db.Lock()
			row := db.ReadRow_unsafe("SELECT rowid FROM tiles WHERE name=='" + name + "'")
			rowid := int64(-1)
			err = row.Scan(&rowid)
			db.Unlock() //because down, there is db.Write() + ui.Paint_file() also lock db

			if err != nil {

				//download
				u := url
				u = strings.ReplaceAll(u, "{x}", strconv.Itoa(int(x)))
				u = strings.ReplaceAll(u, "{y}", strconv.Itoa(int(y)))
				u = strings.ReplaceAll(u, "{z}", strconv.Itoa(int(zoom)))

				img, done, _, err := ui.win.disk.net.GetFile(u, "Skyalt/0.1")
				if done {
					if err == nil {
						//insert into db
						res, err := db.Write("INSERT INTO tiles(name, file) VALUES(?, ?);", name, img)
						if err == nil {
							rowid, err = res.LastInsertId()
							if err != nil {
								return false, fmt.Errorf("LastInsertId() failed: %w", err)
							}
						}
					} else {
						return false, err
					}
				}

			}

			if rowid > 0 {
				//extra margin will fix white spaces during zooming
				ui.Paint_file(tileCoord_sx, tileCoord_sy, tileW, tileH, OsTrnFloat(isZooming, -0.03, 0), InitWinMedia_url(fmt.Sprintf("db:%s:tiles/file/%d", file, rowid)), InitOsCdWhite(), OsV2{0, 0}, false, false)
			}

		}
	}

	//touch
	if start && inside {
		mp.start_pos.X = touch_x //rel, not pixels!
		mp.start_pos.Y = touch_y
		mp.start_tile = UiLayoutMap_lonLatToPos(lon, lat, zoom)
	}

	if wheel != 0 && inside && !isZooming {
		mp.zoomOld = *cam_zoom
		*cam_zoom = UiLayoutMap_zoomClamp(*cam_zoom - wheel)
		if mp.zoomOld != *cam_zoom {
			mp.SetActive(ui)
			mp.lonOld = *cam_lon
			mp.latOld = *cam_lat

			//get touch lon and lat
			touch_lon, touch_lat := UiLayoutMap_posToLonLat(OsV2f{bbStart.X + bbSize.X*touch_x, bbStart.Y + bbSize.Y*touch_y}, mp.zoomOld)
			//get new zoom touch pos
			pos := UiLayoutMap_lonLatToPos(touch_lon, touch_lat, *cam_zoom)
			//get center
			pos.X -= bbSize.X * (touch_x - 0.5)
			pos.Y -= bbSize.Y * (touch_y - 0.5)
			//get new zoom lon and lat
			*cam_lon, *cam_lat = UiLayoutMap_posToLonLat(pos, *cam_zoom)

			mp.start_zoom_time = OsTime()
		}
	}

	if active {
		mp.SetActive(ui)

		var move OsV2f
		move.X = mp.start_pos.X - touch_x
		move.Y = mp.start_pos.Y - touch_y

		rx := move.X * bbSize.X
		ry := move.Y * bbSize.Y

		tileX := mp.start_tile.X + rx
		tileY := mp.start_tile.Y + ry

		*cam_lon, *cam_lat = UiLayoutMap_posToLonLat(OsV2f{tileX, tileY}, *cam_zoom)
	}

	//double click
	if clicks > 1 && end && !isZooming {
		mp.zoomOld = *cam_zoom
		*cam_zoom = UiLayoutMap_zoomClamp(*cam_zoom + 1)

		if mp.zoomOld != *cam_zoom {
			mp.SetActive(ui)

			mp.lonOld = *cam_lon
			mp.latOld = *cam_lat

			//get touch lon and lat
			touch_lon, touch_lat := UiLayoutMap_posToLonLat(OsV2f{bbStart.X + bbSize.X*touch_x, bbStart.Y + bbSize.Y*touch_y}, mp.zoomOld)
			//get new zoom touch pos
			pos := UiLayoutMap_lonLatToPos(touch_lon, touch_lat, *cam_zoom)
			//get center
			pos.X -= bbSize.X * (touch_x - 0.5)
			pos.Y -= bbSize.Y * (touch_y - 0.5)
			//get new zoom lon and lat
			*cam_lon, *cam_lat = UiLayoutMap_posToLonLat(pos, *cam_zoom)

			mp.start_zoom_time = OsTime()
		}
	}

	//copyright
	if copyright != "" {
		h := 1 / height / 2
		ui.Div_startEx(0, 0, 1, 1, 0, 1-h, 1, h, "copyright")
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)
		ui.Div_colMax(0, 100)
		ui.Div_col(1, 5)
		ui.Div_row(0, 0.5)
		ui.Comp_buttonText(1, 0, 1, 1, copyright, Comp_buttonProp().Tooltip(copyright_url))
		ui.Div_end()
	}

	return (old_cam_lon != *cam_lon || old_cam_lat != *cam_lat || old_cam_zoom != *cam_zoom), nil
}

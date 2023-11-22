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
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

func InitImageGlobal() {
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("webp", "webp", webp.Decode, webp.DecodeConfig)
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("jpg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("gif", "gif", gif.Decode, gif.DecodeConfig)
	image.RegisterFormat("tiff", "tiff", tiff.Decode, tiff.DecodeConfig)
	image.RegisterFormat("bmp", "bmp", bmp.Decode, bmp.DecodeConfig)
}

type Image struct {
	origSize   OsV2
	maxUseSize OsV2

	path            MediaPath
	blobDbLoadTicks int64
	blobHash        OsHash

	texture *UiTexture

	lastDrawTick int64
}

func NewImage(path MediaPath) (*Image, error) {

	var img Image

	img.path = path
	img.blobDbLoadTicks = -1

	//load from file, from DB later
	if path.IsFile() {
		blob, err := path.GetFileBlob()
		if err != nil {
			return nil, fmt.Errorf("GetFileBlob() failed: %w", err)
		}
		err = img.SetBlob(blob)
		if err != nil {
			return nil, fmt.Errorf("SetBlob() failed: %w", err)
		}
	}

	return &img, nil
}

func (img *Image) FreeTexture() error {
	if img.texture != nil {
		img.texture.Destroy()
	}

	img.texture = nil
	return nil
}

func (img *Image) GetBytes() int {
	if img.texture != nil {
		sz := img.texture.size
		return sz.X * sz.Y * 4

	}
	return 0
}

func (img *Image) Destroy() error {
	return img.FreeTexture()
}

func (img *Image) SetBlob(blob []byte) error {
	if len(blob) == 0 {
		return nil //empty = no error
	}

	var err error
	img.blobHash, err = InitOsHash(blob)
	if err != nil {
		return fmt.Errorf("InitOsHash() failed: %w", err)
	}

	img.texture, _, err = InitUiTextureFromBlob(blob)
	if err != nil {
		return fmt.Errorf("InitUiTextureFromBlob() failed: %w", err)
	}

	img.origSize = img.texture.size

	return nil
}

func (img *Image) NeedDbChanged(blobDbLoadTicks int64) bool {
	return img.path.IsDb() && img.blobDbLoadTicks != blobDbLoadTicks
}

func (img *Image) Maintenance(ui *Ui) (bool, error) {

	//is used
	if !img.maxUseSize.Is() && !OsIsTicksIn(img.lastDrawTick, 10000) {
		// free un-used
		if img.texture != nil && !OsIsTicksIn(img.lastDrawTick, 10000) {
			img.FreeTexture()
		}
		return false, nil
	}

	img.maxUseSize = OsV2{0, 0} // reset

	return true, nil
}

func (img *Image) Draw(coord OsV4, depth int, cd OsCd) error {

	img.maxUseSize = coord.Size.Max(img.maxUseSize)

	if img.texture != nil {
		img.texture.DrawQuad(coord, depth, cd)
	}

	img.lastDrawTick = OsTicks()
	return nil
}

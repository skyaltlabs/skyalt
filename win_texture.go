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
	"bytes"
	"image"
	"image/draw"
	"os"

	"github.com/go-gl/gl/v2.1/gl"
)

type WinTexture struct {
	id   uint32
	size OsV2
}

func InitWinTextureSize(size OsV2) (*WinTexture, error) {
	var tex WinTexture

	gl.GenTextures(1, &tex.id)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	tex.size = size
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.SRGB_ALPHA, int32(tex.size.X), int32(tex.size.Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	return &tex, nil
}

func InitWinTextureFromImageRGBA(rgba *image.RGBA) (*WinTexture, error) {
	var tex WinTexture

	gl.GenTextures(1, &tex.id)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	tex.size = OsV2{rgba.Rect.Size().X, rgba.Rect.Size().Y}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(tex.size.X), int32(tex.size.Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))

	//gl.GenerateMipmap(texture.id)
	gl.BindTexture(gl.TEXTURE_2D, 0) //UnBind

	return &tex, nil
}

func InitWinTextureFromImageAlpha(alpha *image.Alpha) (*WinTexture, error) {
	var tex WinTexture

	gl.GenTextures(1, &tex.id)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	tex.size = OsV2{alpha.Rect.Size().X, alpha.Rect.Size().Y}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.ALPHA, int32(tex.size.X), int32(tex.size.Y), 0, gl.ALPHA, gl.UNSIGNED_BYTE, gl.Ptr(alpha.Pix))

	gl.BindTexture(gl.TEXTURE_2D, 0) //UnBind

	return &tex, nil
}

func InitWinTextureFromImage(img image.Image) (*WinTexture, error) {

	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Pt(0, 0), draw.Src)

	return InitWinTextureFromImageRGBA(rgba)
}

func InitWinTextureFromBlob(blob []byte) (*WinTexture, image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(blob))
	if err != nil {
		return nil, nil, err
	}

	tex, err := InitWinTextureFromImage(img)
	return tex, img, err
}

func InitWinTextureFromFile(path string) (*WinTexture, image.Image, error) {
	imgFile, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, nil, err
	}

	tex, err := InitWinTextureFromImage(img)
	return tex, img, err
}

func (tex *WinTexture) Destroy() {
	if tex.id > 0 {
		gl.DeleteTextures(1, &tex.id)
	}
}
func _WinTexture_drawQuadNoUVs(coord OsV4, depth int) {
	s := coord.Start
	e := coord.End()

	gl.Vertex3f(float32(s.X), float32(s.Y), float32(depth))
	gl.Vertex3f(float32(e.X), float32(s.Y), float32(depth))
	gl.Vertex3f(float32(e.X), float32(e.Y), float32(depth))
	gl.Vertex3f(float32(s.X), float32(e.Y), float32(depth))
}

func (tex *WinTexture) DrawQuadUV(coord OsV4, depth int, cd OsCd, sUV, eUV OsV2f) {
	gl.Color4ub(cd.R, cd.G, cd.B, cd.A)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)

	gl.Begin(gl.QUADS)
	{
		s := coord.Start
		e := coord.End()

		gl.TexCoord2f(sUV.X, sUV.Y)
		gl.Vertex3f(float32(s.X), float32(s.Y), float32(depth))

		gl.TexCoord2f(eUV.X, sUV.Y)
		gl.Vertex3f(float32(e.X), float32(s.Y), float32(depth))

		gl.TexCoord2f(eUV.X, eUV.Y)
		gl.Vertex3f(float32(e.X), float32(e.Y), float32(depth))

		gl.TexCoord2f(sUV.X, eUV.Y)
		gl.Vertex3f(float32(s.X), float32(e.Y), float32(depth))
	}
	gl.End()

	gl.BindTexture(gl.TEXTURE_2D, 0) //Unbind
}

func (tex *WinTexture) DrawQuad(coord OsV4, depth int, cd OsCd) {
	tex.DrawQuadUV(coord, depth, cd, OsV2f{}, OsV2f{1, 1})
}

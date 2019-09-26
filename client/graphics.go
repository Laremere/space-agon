// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build js

package main

import (
	"fmt"
	"log"
	"math"
	"syscall/js"

	"github.com/googleforgames/space-agon/client/webgl"
)

type graphics struct {
	w *webgl.WebGL

	width  int
	height int

	spritesheet *webgl.Texture
	shader      *webgl.Program

	coords              []float32
	coordsBuffer        *webgl.Buffer
	textureCoords       []float32
	textureCoordsBuffer *webgl.Buffer

	written int
}

func NewGraphics() (*graphics, error) {
	g := &graphics{}

	canvas := js.Global().Get("document").Call("getElementById", "game")

	var err error
	g.w, err = webgl.InitWebgl(canvas)
	if err != nil {
		return nil, err
	}

	g.w.Enable(g.w.BLEND)
	g.w.BlendFunc(g.w.SRC_ALPHA, g.w.ONE_MINUS_SRC_ALPHA)

	spritesheetElement := js.Global().Get("document").Call("getElementById", "spritesheet")
	log.Println("WIDTH", spritesheetElement.Get("width"))
	log.Println("HEIGHT", spritesheetElement.Get("height"))
	g.spritesheet = webgl.LoadTexture(g.w, spritesheetElement)

	g.width = canvas.Get("width").Int()
	g.height = canvas.Get("height").Int()

	g.shader, err = webgl.CreateProgram(
		g.w,
		`
    attribute vec2 aVertexPosition;
    attribute vec2 aTextureCoord;

    varying highp vec2 vTextureCoord;

    void main() {
      gl_Position = vec4(aVertexPosition, 0.0, 1.0);
      vTextureCoord = aTextureCoord;
    }`,
		`
    precision highp float;

    varying highp vec2 vTextureCoord;

    uniform sampler2D uSampler;

    void main() {
      gl_FragColor = texture2D(uSampler, vTextureCoord);
    }
    `,
	)
	if err != nil {
		return nil, fmt.Errorf("Error building shader: %w", err)
	}

	g.coords = make([]float32, 3*2*2*1000)
	g.textureCoords = make([]float32, len(g.coords))
	g.written = 0

	g.coordsBuffer = g.w.CreateBuffer()
	g.w.BindBuffer(g.w.ARRAY_BUFFER, g.coordsBuffer)

	// 4 comes from 4 bytes per float32
	g.w.BufferDataSize(g.w.ARRAY_BUFFER, 4*len(g.coords), g.w.DYNAMIC_DRAW)

	g.textureCoordsBuffer = g.w.CreateBuffer()
	g.w.BindBuffer(g.w.ARRAY_BUFFER, g.textureCoordsBuffer)
	g.w.BufferDataSize(g.w.ARRAY_BUFFER, 4*len(g.textureCoords), g.w.DYNAMIC_DRAW)

	return g, nil
}

func (g *graphics) Flush() {
	if g.written == 0 {
		return
	}

	g.w.UseProgram(g.shader)

	g.w.BindBuffer(g.w.ARRAY_BUFFER, g.coordsBuffer)
	g.w.BufferSubDataF32(g.w.ARRAY_BUFFER, 0, g.coords[:g.written])
	aVertexPosition := g.w.GetAttribLocation(g.shader, "aVertexPosition")
	g.w.EnableVertexAttribArray(aVertexPosition)
	// Bind current array buffer to the given vertex attribute
	g.w.VertexAttribPointer(aVertexPosition, 2, g.w.FLOAT, false, 0, 0) // 2 = points per vertex

	g.w.BindBuffer(g.w.ARRAY_BUFFER, g.textureCoordsBuffer)
	g.w.BufferSubDataF32(g.w.ARRAY_BUFFER, 0, g.textureCoords[:g.written])
	aTextureCoord := g.w.GetAttribLocation(g.shader, "aTextureCoord")
	g.w.EnableVertexAttribArray(aTextureCoord)
	// Bind current array buffer to the given vertex attribute
	g.w.VertexAttribPointer(aTextureCoord, 2, g.w.FLOAT, false, 0, 0) // 2 = points per vertex

	g.w.ActiveTexture(g.w.TEXTURE0)
	g.w.BindTexture(g.w.TEXTURE_2D, g.spritesheet)
	uSampler := g.w.GetUniformLocation(g.shader, "uSampler")
	g.w.Uniform1i(uSampler, 0)

	g.w.DrawArrays(g.w.TRIANGLES, 0, g.written/2) // two floats per vertex

	g.written = 0

	glError := g.w.GetError()
	if glError.Int() != 0 {
		log.Println("GL Error:", g.w.GetError())
	}
}

type Sprite struct {
	textureCoords []float32
	size          float32
}

var (
	Spaceship = &Sprite{
		textureCoords: genTexCoords(0, 0, 0.25, 0.25),
		size:          1,

		//   []float32{
		//  0, 0,
		//  0, 1 / 8,
		//  1 / 8, 1 / 8,
		//  0, 0,
		//  1 / 8, 1 / 8,
		//  1 / 8, 0,

		//  // 0, 0,
		//  // 0, 1 / 2,
		//  // 1 / 2, 1 / 2,
		//  // 0, 0,
		//  // 1, 1,
		//  // 1, 0,
		// },
	}
)

func genTexCoords(xStart, yStart, xEnd, yEnd float32) []float32 {
	return []float32{
		xStart, yStart,
		xStart, yEnd,
		xEnd, yEnd,
		xStart, yStart,
		xEnd, yEnd,
		xEnd, yStart,
	}
}

//
//  0 1 : 6 7 ----------    : 10 11
//      |     \_            |
//      |       \_          |
//      |         \_        |
//      |           \_      |
//      |             \_    |
//  2 3 :     ----------4 5 : 8 9
//

// TODO: spriteId -> size and texture location, rotation
func (g *graphics) Sprite(s *Sprite, centerx, centery, rotation float32) {
	coords := g.coords[g.written : g.written+12]
	textureCoords := g.textureCoords[g.written : g.written+12]

	cosSize := s.size * float32(math.Cos(float64(rotation)+(math.Pi/4))) * math.Sqrt2 / 2
	sinSize := s.size * float32(math.Sin(float64(rotation)+(math.Pi/4))) * math.Sqrt2 / 2

	coords[0] = centerx - sinSize
	coords[1] = centery + cosSize

	coords[2] = centerx - cosSize
	coords[3] = centery - sinSize

	coords[4] = centerx + sinSize
	coords[5] = centery - cosSize
	coords[6] = coords[0]
	coords[7] = coords[1]

	coords[8] = coords[4]
	coords[9] = coords[5]

	coords[10] = centerx + cosSize
	coords[11] = centery + sinSize
	for i := 0; i < 12; i++ {
		textureCoords[i] = s.textureCoords[i]
	}

	g.written += 12

	// xMin := 10
	// width := 10
	// yMin := 10
	// height := 10

	// left := float32(xMin)
	// g.vertices[g.written+0] = left
	// g.vertices[g.written+2] = left
	// g.vertices[g.written+6] = left

	// top := float32(yMin)
	// g.vertices[g.written+1] = top
	// g.vertices[g.written+7] = top
	// g.vertices[g.written+11] = top

	// bottom := float32(yMin + height)
	// g.vertices[g.written+3] = bottom
	// g.vertices[g.written+5] = bottom
	// g.vertices[g.written+9] = bottom

	// right := float32(xMin + width)
	// g.vertices[g.written+4] = right
	// g.vertices[g.written+8] = right
	// g.vertices[g.written+10] = right

	// g.written += 12
	if g.written >= len(g.coords) {
		g.Flush()
	}

}

func (g *graphics) Clear() {
	g.w.ClearColor(0.039, 0.102, 0.247, 1)
	g.w.Clear(g.w.COLOR_BUFFER_BIT)
}

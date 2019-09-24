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
	"syscall/js"

	"github.com/googleforgames/space-agon/client/webgl"
)

func main() {
	g, err := NewGraphics()
	if err != nil {
		js.Global().Get("document").Get("body").Set("innerHTML", js.ValueOf(err.Error()))
		return
	}
	g.Clear()
	// g.Sprite(0.5, 0.5)
	g.Flush()
}

type Graphics struct {
	w *webgl.WebGL

	width  int
	height int
}

func NewGraphics() (*Graphics, error) {
	g := &Graphics{}

	canvas := js.Global().Get("document").Call("getElementById", "game")

	var err error
	g.w, err = webgl.InitWebgl(canvas)
	if err != nil {
		return nil, err
	}

	g.width = canvas.Get("width").Int()
	g.height = canvas.Get("height").Int()

	return g, nil
}

func (g *Graphics) Flush() {

}

// TODO: spriteId -> size and texture location, rotation
func (g *Graphics) Sprite(centerx, centery float64) {

}

func (g *Graphics) Clear() {
	g.w.ClearColor(0.039, 0.102, 0.247, 1)
	g.w.Clear(g.w.COLOR_BUFFER_BIT)
}

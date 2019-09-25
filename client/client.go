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
	"log"
	"syscall/js"
)

func main() {
	js.Global().Call("whenLoaded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		g, err := newGame()
		if err == nil {
			g.scheduleFrame()
		} else {
			js.Global().Get("document").Get("body").Set("innerHTML", js.ValueOf(err.Error()))
		}
		return nil
	}))

	<-make(chan struct{})
}

func newGame() (*game, error) {
	log.Println("Initiating Graphics.")
	gr, err := NewGraphics()
	if err != nil {
		return nil, err
	}

	return &game{
		gr: gr,
	}, nil
}

type game struct {
	gr *graphics
}

func (g *game) scheduleFrame() {
	js.Global().Get("window").Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		g.frame()
		return nil
	}))
}

var rotation = float32(0)

func (g *game) frame() {
	rotation += 0.01
	g.gr.Clear()
	g.gr.Sprite(Spaceship, 0.1, 0.2, rotation)
	g.gr.Flush()

	g.scheduleFrame()
}

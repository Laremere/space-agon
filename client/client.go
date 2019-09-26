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

	"github.com/googleforgames/space-agon/game"
)

func main() {
	js.Global().Call("whenLoaded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c, err := newClient()
		if err == nil {
			c.scheduleFrame()
		} else {
			js.Global().Get("document").Get("body").Set("innerHTML", js.ValueOf(err.Error()))
		}
		return nil
	}))

	<-make(chan struct{})
}

func newClient() (*client, error) {
	log.Println("Initiating Graphics.")
	gr, err := NewGraphics()
	if err != nil {
		return nil, err
	}

	return &client{
		gr:  gr,
		g:   game.NewGame(),
		inp: &game.Input{},
	}, nil
}

type client struct {
	gr  *graphics
	g   *game.Game
	inp *game.Input
}

func (c *client) scheduleFrame() {
	js.Global().Get("window").Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.frame()
		return nil
	}))
}

var rotation = float32(0)

func (c *client) frame() {

	c.inp.Up.Hold = true
	c.inp.Left.Hold = true

	c.g.Step(c.inp)

	// rotation += 0.01
	c.gr.Clear()
	// c.gr.Sprite(Spaceship, 0.1, 0.2, rotation)

	// for _, bag := range c.g.E.Bags {
	// 	_ = bag
	// }

	{
		i := game.NewIter(c.g.E)
		i.Require(game.PosKey)
		i.Require(game.SpriteKey)
		for i.Next() {
			p := *i.Pos()
			rot := i.Rot()
			rotation := float32(0)
			if rot != nil {
				rotation = *rot
			}
			c.gr.Sprite(spritemap[*i.Sprite()], p[0], p[1], rotation)
		}
	}

	c.gr.Flush()

	c.scheduleFrame()
}

var spritemap = map[game.Sprite]*Sprite{
	game.SpriteShip: Spaceship,
}

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

package game

type Sprite uint16

const (
	SpriteUnset = Sprite(iota)
	SpriteShip
	SpirteMissile
)

type Game struct {
	E           *Entities
	initialized bool
}

func NewGame() *Game {
	g := &Game{
		E: newEntities(),
	}

	return g
}

type Keystate struct {
	Press   bool
	Hold    bool
	Release bool
}

type Input struct {
	Up    Keystate
	Down  Keystate
	Left  Keystate
	Right Keystate
	Fire  Keystate
	Dt    float32
}

func (g *Game) Step(input *Input) {
	if !g.initialized {
		i := NewIter(g.E)
		i.Require(PosKey)
		i.Require(RotKey)
		i.Require(SpriteKey)
		i.New()

		pos := i.Pos()
		(*pos)[0] = 0.3
		(*pos)[1] = 0.7
		*i.Sprite() = SpriteShip
		*i.Rot() = 1

		g.initialized = true
	}
}

func (g *Game) FrameEnd() {
	{
		i := NewIter(g.E)
		i.Require(FrameEndDeleteKey)
		for i.Next() {
			i.Remove()
		}
	}
}

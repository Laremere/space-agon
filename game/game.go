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

import (
	"math"
)

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

func (k *Keystate) FrameEndReset() {
	k.Press = false
	k.Release = false
}

func (k *Keystate) Down() {
	if !k.Hold {
		k.Press = true
		k.Hold = true
	}
}

func (k *Keystate) Up() {
	if k.Hold {
		k.Release = true
		k.Hold = false
	}
}

type Input struct {
	Up    Keystate
	Down  Keystate
	Left  Keystate
	Right Keystate
	Fire  Keystate
	Dt    float32
}

func (inp *Input) FrameEndReset() {
	inp.Up.FrameEndReset()
	inp.Down.FrameEndReset()
	inp.Left.FrameEndReset()
	inp.Right.FrameEndReset()
	inp.Fire.FrameEndReset()
}

func (g *Game) Step(input *Input) {
	if !g.initialized {
		i := g.E.NewIter()
		i.Require(PosKey)
		i.Require(RotKey)
		i.Require(SpriteKey)
		i.Require(PlayerControlledShipKey)
		i.Require(TimedDestroyKey)
		i.Require(KeepInCameraKey)
		i.New()

		pos := i.Pos()
		(*pos)[0] = 0
		(*pos)[1] = 0
		*i.Sprite() = SpriteShip
		*i.Rot() = 0
		*i.TimedDestroy() = 50

		// g.initialized = true
	}

	{
		i := g.E.NewIter()
		i.Require(TimedDestroyKey)
		for i.Next() {
			*i.TimedDestroy() -= input.Dt
			if *i.TimedDestroy() <= 0 {
				i.Remove()
			}
		}
	}

	{
		i := g.E.NewIter()
		i.Require(PosKey)
		i.Require(RotKey)
		i.Require(PlayerControlledShipKey)
		for i.Next() {
			const rotationSpeed = 1
			const forwardSpeed = 1

			if input.Left.Hold {
				*i.Rot() += rotationSpeed * input.Dt
			}
			if input.Right.Hold {
				*i.Rot() -= rotationSpeed * input.Dt
			}

			if input.Up.Hold {
				dx := float32(math.Cos(float64(*i.Rot()))) * forwardSpeed * input.Dt
				dy := float32(math.Sin(float64(*i.Rot()))) * forwardSpeed * input.Dt

				(*i.Pos())[0] += dx
				(*i.Pos())[1] += dy
			}
		}
	}

}

func (g *Game) FrameEnd() {
	{
		i := g.E.NewIter()
		i.Require(FrameEndDeleteKey)
		for i.Next() {
			i.Remove()
		}
	}
}

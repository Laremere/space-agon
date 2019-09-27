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
	"math/rand"
)

type Game struct {
	E              *Entities
	initialized    bool
	ControlledShip *Lookup
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
	Up         Keystate
	Down       Keystate
	Left       Keystate
	Right      Keystate
	Fire       Keystate
	Dt         float32
	IsRendered bool
}

func (inp *Input) FrameEndReset() {
	inp.Up.FrameEndReset()
	inp.Down.FrameEndReset()
	inp.Left.FrameEndReset()
	inp.Right.FrameEndReset()
	inp.Fire.FrameEndReset()
}

func (g *Game) Step(input *Input) {
	if g.ControlledShip.Alive() {
		i := g.E.NewIter()
		i.Get(g.ControlledShip)

		shipControl := i.ShipControl()
		shipControl.Up = input.Up.Hold
		shipControl.Down = input.Down.Hold
		shipControl.Left = input.Left.Hold
		shipControl.Right = input.Right.Hold
		shipControl.Fire = input.Fire.Hold
	}

	if !g.initialized {
		{ // Spawn Spaceship
			i := g.E.NewIter()
			i.Require(PosKey)
			i.Require(RotKey)
			i.Require(SpriteKey)
			i.Require(KeepInCameraKey)
			i.Require(SpinKey)
			i.Require(MomentumKey)
			i.Require(ShipControlKey)
			i.Require(LookupKey)
			i.New()

			pos := i.Pos()
			(*pos)[0] = 5
			(*pos)[1] = 0
			*i.Sprite() = SpriteShip
			*i.Rot() = 0

			g.ControlledShip = i.Lookup()
		}

		{ // spawn stars
			i := g.E.NewIter()
			i.Require(PosKey)
			i.Require(SpriteKey)
			i.New()
			// Big star
			*i.Sprite() = SpriteStar

			// spawn small stars
			for j := 0; j < 100; j++ {
				i.New()
				*i.Sprite() = SpriteStarBit
				const starBoxRadius = 50
				*i.Pos() = Vec2{
					rand.Float32()*starBoxRadius*2 - starBoxRadius,
					rand.Float32()*starBoxRadius*2 - starBoxRadius,
				}
			}
		}

		g.initialized = true
	}

	if input.IsRendered { // Spawn sun particles
		i := g.E.NewIter()
		i.Require(PosKey)
		i.Require(SpriteKey)
		i.Require(MomentumKey)
		i.Require(TimedDestroyKey)
		i.New()

		*i.Sprite() = SpriteStarBit

		rad := rand.Float32() * 2 * math.Pi
		*i.Pos() = Vec2FromRadians(rad)
		rad += rand.Float32()*2 - 1
		*i.Momentum() = Vec2FromRadians(rad).Scale(rand.Float32()*5 + 1)
		*i.TimedDestroy() = rand.Float32()*2 + 1
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
		i.Require(ShipControlKey)
		i.Require(SpinKey)
		i.Require(MomentumKey)
		for i.Next() {
			const rotationForSpeed = 5
			const rotationAgainstSpeed = 10
			const forwardSpeed = 2

			spinDesire := float32(0)
			if i.ShipControl().Left {
				spinDesire++
			}
			if i.ShipControl().Right {
				spinDesire--
			}
			if !i.ShipControl().Left && !i.ShipControl().Right {
				if *i.Spin() < 0 {
					spinDesire += 0.1
				} else {
					spinDesire -= 0.1
				}
			}

			// Game feel: Stopping spin is easier than starting it.
			if (spinDesire < 0) == (*i.Spin() < 0) {
				spinDesire *= rotationForSpeed
			} else {
				spinDesire *= rotationAgainstSpeed
			}

			*i.Spin() += spinDesire * input.Dt

			if i.ShipControl().Up {
				dx := float32(math.Cos(float64(*i.Rot()))) * forwardSpeed * input.Dt
				dy := float32(math.Sin(float64(*i.Rot()))) * forwardSpeed * input.Dt

				(*i.Momentum())[0] += dx
				(*i.Momentum())[1] += dy
			}
		}
	}

	{
		i := g.E.NewIter()
		i.Require(RotKey)
		i.Require(SpinKey)

		for i.Next() {
			*i.Rot() += *i.Spin() * input.Dt
		}
	}

	{
		i := g.E.NewIter()
		i.Require(PosKey)
		i.Require(MomentumKey)

		for i.Next() {
			i.Pos().AddEqual(i.Momentum().Scale(input.Dt))
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

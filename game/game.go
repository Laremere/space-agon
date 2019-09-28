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
	"log"
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
			i.Require(AffectedByGravityKey)
			i.New()

			pos := i.Pos()
			(*pos)[0] = 20
			(*pos)[1] = 0
			(*i.Momentum())[1] = 5
			*i.Sprite() = SpriteShip
			*i.Rot() = 0

			g.ControlledShip = i.Lookup()
		}

		if input.IsRendered { // spawn stars
			{ // Big star
				i := g.E.NewIter()
				i.Require(PosKey)
				i.Require(SpriteKey)
				i.New()

				*i.Sprite() = SpriteStar
			}

			{
				i := g.E.NewIter()
				i.Require(PosKey)
				i.Require(PointRenderKey)

				// spawn small stars
				const density = 0.05
				const starBoxRadius = 200
				for j := 0; j < int(density*starBoxRadius*starBoxRadius); j++ {
					i.New()
					// *i.Sprite() = SpriteStarBit
					*i.Pos() = Vec2{
						rand.Float32()*starBoxRadius*2 - starBoxRadius,
						rand.Float32()*starBoxRadius*2 - starBoxRadius,
					}
				}
			}
		}

		g.initialized = true
	}

	if input.IsRendered { // Spawn sun particles
		i := g.E.NewIter()
		i.Require(PosKey)
		// i.Require(SpriteKey)
		i.Require(PointRenderKey)
		i.Require(MomentumKey)
		i.Require(TimedDestroyKey)

		for j := 0; j < 10; j++ {
			i.New()
			rad := rand.Float32() * 2 * math.Pi
			*i.Pos() = Vec2FromRadians(rad)
			rad += rand.Float32()*2 - 1
			*i.Momentum() = Vec2FromRadians(rad).Scale(rand.Float32()*5 + 1)
			*i.TimedDestroy() = rand.Float32()*2 + 1
		}
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
			///////////////////////////
			// Ship Movement Controls
			///////////////////////////
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

			///////////////////////////
			// Ship Weapons
			///////////////////////////
			i.ShipControl().FireCoolDown -= input.Dt

			if i.ShipControl().FireCoolDown <= 0 && i.ShipControl().Fire {
				i.ShipControl().FireCoolDown = 0.5
				// i.ShipControl().FireCoolDown = 5

				im := g.E.NewIter()
				im.Require(PosKey)
				im.Require(RotKey)
				im.Require(SpinKey)
				im.Require(MomentumKey)
				im.Require(SpriteKey)
				im.Require(AffectedByGravityKey)
				im.Require(TimedDestroyKey)
				im.New()

				*im.Sprite() = SpriteMissile
				*im.TimedDestroy() = 10
				*im.Pos() = *i.Pos()
				*im.Rot() = *i.Rot()
				*im.Spin() = *i.Spin()
				const MissileSpeed = 10
				log.Println(*i.Rot() / math.Pi * 180)
				*im.Momentum() = *i.Momentum()
				im.Momentum().AddEqual(Vec2FromRadians(*i.Rot()).Scale(MissileSpeed))
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
		// Force of gravity = gravconst * mass1 * mass2 / (distance)^2

		// Update value = Dt * const * normalized direction vector / (distance)^2

		// = Dt * const * (-1 * Pos / Pos.Lenght) / (Pos.Length) ^ 2
		// = Dt * const * -1 * Pos / Pos.Length ^ 3
		// = Pos.Scale(Dt * const * -1 / Pos.Length ^ 3)

		// Pos.Length = (x*x + y*y) ^ 1/2
		// sqrt then cube will probably be faster than taking to the power of 1.5?

		i := g.E.NewIter()
		i.Require(PosKey)
		i.Require(AffectedByGravityKey)
		i.Require(MomentumKey)

		const gravityStrength = 200

		for i.Next() {
			length := i.Pos().Length()
			lengthCubed := length * length * length
			i.Momentum().AddEqual(i.Pos().Scale(-1 * gravityStrength * input.Dt / lengthCubed))
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

	{
		i := g.E.NewIter()
		i.Require(FrameEndDeleteKey)
		for i.Next() {
			i.Remove()
		}
	}
}

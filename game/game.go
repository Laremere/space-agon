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
	E               *Entities
	initialized     bool
	ControlledShip  *Lookup
	NextNetworkId   NetworkId
	NewClientUpdate *NetworkUpdate // TODO: Actually send to server from clients on connect
}

func NewGame() *Game {
	g := &Game{
		E: newEntities(),
		// Oh man, this is such a bad hack.
		NextNetworkId:   NetworkId(rand.Int63()),
		NewClientUpdate: NewNetworkUpdate(),
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
	// Whether entities which only exist for render should be created.
	IsRendered bool
	// Whether code which runs only if the input is going to control
	// a player ship should run.
	IsPlayer bool
	// Whether the this instance is the host, if not it is a client.
	IsHost bool
	Conns  map[int]*NetworkConnection
}

func NewInput() *Input {
	return &Input{
		Conns: make(map[int]*NetworkConnection),
	}
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

	connUpdatesOutputs := make(map[int]*NetworkUpdate)
	connUpdatesOutputs[-1] = NewNetworkUpdate()

	networkTracks := make(map[NetworkId]*NetworkTrack)
	destroyEvents := make(map[NetworkId]struct{})

	{
		for id := range input.Conns {
			connUpdatesOutputs[id] = NewNetworkUpdate()
		}
		for id, conn := range input.Conns {
			select {
			default:
			case u := <-conn.Recieving:
				for nid := range u.DestroyEvents {
					for oid := range connUpdatesOutputs {
						if oid != id {
							connUpdatesOutputs[oid].DestroyEvents[nid] = struct{}{}
						}
					}
					// log.Println("Got destroy for", nid)
					destroyEvents[nid] = struct{}{}
				}
				//////////////////////
				// spawn events
				//////////////////////
				for nid, spawnType := range u.SpawnEvents {

					for oid := range connUpdatesOutputs {
						if oid != id {
							connUpdatesOutputs[oid].SpawnEvents[nid] = spawnType
						}
					}

					switch spawnType {
					case SpawnShip:
						i := g.E.NewIter()
						i.Require(NetworkRecieveKey)
						// i.Require(NetworkPosRecieveKey)
						// i.Require(NetworkRotRecieveKey)
						// i.Require(NetworkMomentumRecieveKey)
						// i.Require(NetworkSpinRecieveKey)
						// i.Require(NetworkShipControlRecieveKey)
						spawnSpaceship(i)
						*i.NetworkId() = nid

					case SpawnMissile:
						i := g.E.NewIter()
						i.Require(NetworkRecieveKey)
						// i.Require(NetworkPosRecieveKey)
						// i.Require(NetworkRotRecieveKey)
						// i.Require(NetworkMomentumRecieveKey)
						// i.Require(NetworkSpinRecieveKey)
						spawnMissile(i)
						*i.NetworkId() = nid

					default:
						panic("Spawn what now?")
					}
				}

				//////////////////////
				// Tracks
				//////////////////////

				for nid, t := range u.Tracks {
					networkTracks[nid] = t
					for oid := range connUpdatesOutputs {
						if oid != id {
							connUpdatesOutputs[oid].Tracks[nid] = t
						}
					}
				}
			}
		}
	}

	{ // Despawn destroyed
		i := g.E.NewIter()
		// i.Require(PosKey)
		// i.Require(NetworkRecieveKey)
		// i.Require(NetworkPosRecieveKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			if _, ok := destroyEvents[*i.NetworkId()]; ok {
				i.Remove()
			}
		}
	}
	{ // Track Pos
		i := g.E.NewIter()
		i.Require(PosKey)
		i.Require(NetworkRecieveKey)
		// i.Require(NetworkPosRecieveKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			track, ok := networkTracks[*i.NetworkId()]
			if ok {
				*i.Pos() = track.Pos
			}
		}
	}
	{ // Track Rot
		i := g.E.NewIter()
		i.Require(RotKey)
		i.Require(NetworkRecieveKey)
		// i.Require(NetworkRotRecieveKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			track, ok := networkTracks[*i.NetworkId()]
			if ok {
				*i.Rot() = track.Rot
			}
		}
	}
	{ // Track Momentum
		i := g.E.NewIter()
		i.Require(MomentumKey)
		i.Require(NetworkRecieveKey)
		// i.Require(NetworkMomentumRecieveKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			track, ok := networkTracks[*i.NetworkId()]
			if ok {
				*i.Momentum() = track.Momentum
			}
		}
	}
	{ // Track Spin
		i := g.E.NewIter()
		i.Require(SpinKey)
		i.Require(NetworkRecieveKey)
		// i.Require(NetworkSpinRecieveKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			track, ok := networkTracks[*i.NetworkId()]
			if ok {
				*i.Spin() = track.Spin
			}
		}
	}
	{ // Track ShipControl
		i := g.E.NewIter()
		i.Require(ShipControlKey)
		i.Require(NetworkRecieveKey)
		// i.Require(NetworkShipControlRecieveKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			track, ok := networkTracks[*i.NetworkId()]
			if ok {
				*i.ShipControl() = track.ShipControl
			}
		}
	}

	if !g.initialized {
		if input.IsPlayer { // Spawn Spaceship
			i := g.E.NewIter()
			i.Require(NetworkTransmitKey)
			// i.Require(NetworkPosTransmitKey)
			// i.Require(NetworkRotTransmitKey)
			// i.Require(NetworkMomentumTransmitKey)
			// i.Require(NetworkSpinTransmitKey)
			// i.Require(NetworkShipControlTransmitKey)
			spawnSpaceship(i)

			pos := i.Pos()
			(*pos)[0] = 7
			(*pos)[1] = 0
			(*i.Momentum())[1] = 5
			*i.Rot() = 0
			*i.NetworkId() = g.NextNetworkId
			g.NextNetworkId++

			g.ControlledShip = i.Lookup()

			for _, u := range connUpdatesOutputs {
				u.SpawnEvents[*i.NetworkId()] = SpawnShip
			}

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
				if i.NetworkId() != nil {
					for _, u := range connUpdatesOutputs {
						// log.Println("Sent destroy for ", *i.NetworkId())
						u.DestroyEvents[*i.NetworkId()] = struct{}{}
					}
				}
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
				im.Require(NetworkTransmitKey)
				// im.Require(NetworkPosTransmitKey)
				// im.Require(NetworkRotTransmitKey)
				// im.Require(NetworkMomentumTransmitKey)
				// im.Require(NetworkSpinTransmitKey)
				im.Require(TimedDestroyKey)
				spawnMissile(im)
				*im.TimedDestroy() = 2
				*im.Pos() = *i.Pos()
				*im.Rot() = *i.Rot()
				*im.Spin() = *i.Spin()
				const MissileSpeed = 10
				*im.Momentum() = *i.Momentum()
				im.Momentum().AddEqual(Vec2FromRadians(*i.Rot()).Scale(MissileSpeed))

				*im.NetworkId() = g.NextNetworkId
				g.NextNetworkId++

				for _, u := range connUpdatesOutputs {
					u.SpawnEvents[*im.NetworkId()] = SpawnMissile
				}
			}
		}
	}

	{
		i := g.E.NewIter()
		i.Require(RotKey)
		i.Require(MomentumKey)
		i.Require(MissileKey)

		for i.Next() {
			const pushFactor = 10
			i.Momentum().AddEqual(Vec2FromRadians(*i.Rot()).Scale(pushFactor * input.Dt))
		}
	}

	if input.IsRendered { // Spawn missile trail particles
		i := g.E.NewIter()
		i.Require(RotKey)
		i.Require(MissileKey)
		i.Require(PosKey)

		ip := g.E.NewIter()
		ip.Require(PosKey)
		ip.Require(PointRenderKey)
		ip.Require(MomentumKey)
		ip.Require(TimedDestroyKey)

		for i.Next() {
			const pushFactor = 5

			for j := 0; j < 4; j++ {
				ip.New()
				*ip.Pos() = *i.Pos()

				angleOut := *i.Rot() + math.Pi + (rand.Float32()-0.5)/2
				*ip.Momentum() = i.Momentum().Add(Vec2FromRadians(angleOut).Scale(pushFactor))
				ip.Pos().AddEqual(ip.Momentum().Scale(float32(j) * input.Dt / 4))
				*ip.TimedDestroy() = rand.Float32()*2 + 1
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

	{ // Transmit Pos
		i := g.E.NewIter()
		i.Require(PosKey)
		i.Require(NetworkTransmitKey)
		// i.Require(NetworkPosTransmitKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			for _, u := range connUpdatesOutputs {
				t, ok := u.Tracks[*i.NetworkId()]
				if !ok {
					t = &NetworkTrack{}
					u.Tracks[*i.NetworkId()] = t
				}
				t.Pos = *i.Pos()
			}
		}
	}
	{ // Transmit Rot
		i := g.E.NewIter()
		i.Require(RotKey)
		i.Require(NetworkTransmitKey)
		// i.Require(NetworkRotTransmitKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			for _, u := range connUpdatesOutputs {
				t, ok := u.Tracks[*i.NetworkId()]
				if !ok {
					t = &NetworkTrack{}
					u.Tracks[*i.NetworkId()] = t
				}
				t.Rot = *i.Rot()
			}
		}
	}
	{ // Transmit Momentum
		i := g.E.NewIter()
		i.Require(MomentumKey)
		i.Require(NetworkTransmitKey)
		// i.Require(NetworkMomentumTransmitKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			for _, u := range connUpdatesOutputs {
				t, ok := u.Tracks[*i.NetworkId()]
				if !ok {
					t = &NetworkTrack{}
					u.Tracks[*i.NetworkId()] = t
				}
				t.Momentum = *i.Momentum()
			}
		}
	}
	{ // Transmit Spin
		i := g.E.NewIter()
		i.Require(SpinKey)
		i.Require(NetworkTransmitKey)
		// i.Require(NetworkSpinTransmitKey)
		i.Require(NetworkIdKey)
		for i.Next() {
			for _, u := range connUpdatesOutputs {
				t, ok := u.Tracks[*i.NetworkId()]
				if !ok {
					t = &NetworkTrack{}
					u.Tracks[*i.NetworkId()] = t
				}
				t.Spin = *i.Spin()
			}
		}
	}

	for id := range input.Conns {
		NetworkUpdateCombineAndPass(input.Conns[id].Sending, connUpdatesOutputs[id])
	}

	g.NewClientUpdate.AndThen(connUpdatesOutputs[-1])
}

type NetworkConnection struct {
	Sending   chan *NetworkUpdate
	Recieving chan *NetworkUpdate
}

func NewNetworkConnection() *NetworkConnection {
	n := &NetworkConnection{
		Sending:   make(chan *NetworkUpdate, 1),
		Recieving: make(chan *NetworkUpdate, 1),
	}
	return n
}

func NetworkUpdateCombineAndPass(c chan *NetworkUpdate, u *NetworkUpdate) {
	select {
	case c <- u:
	case uPrevious := <-c:
		toSend := NewNetworkUpdate()
		toSend.AndThen(uPrevious)
		toSend.AndThen(u)
		c <- toSend
	}
}

type NetworkId uint64

type NetworkUpdate struct {
	SpawnEvents   map[NetworkId]SpawnType
	DestroyEvents map[NetworkId]struct{}
	Tracks        map[NetworkId]*NetworkTrack
}

func NewNetworkUpdate() *NetworkUpdate {
	return &NetworkUpdate{
		SpawnEvents:   make(map[NetworkId]SpawnType),
		DestroyEvents: make(map[NetworkId]struct{}),
		Tracks:        make(map[NetworkId]*NetworkTrack),
	}
}

func (uPrevious *NetworkUpdate) AndThen(uNext *NetworkUpdate) {
	for k, v := range uNext.Tracks {
		uPrevious.Tracks[k] = v
	}
	for k, v := range uNext.SpawnEvents {
		uPrevious.SpawnEvents[k] = v
	}
	for k, v := range uNext.DestroyEvents {
		if _, ok := uPrevious.SpawnEvents[k]; ok {
			delete(uPrevious.SpawnEvents, k)
		} else {
			uPrevious.DestroyEvents[k] = v
		}
	}
}

type SpawnType uint

const (
	SpawnShip = SpawnType(iota)
	SpawnMissile
)

type NetworkTrack struct {
	Pos         Vec2
	Momentum    Vec2
	Rot         float32
	Spin        float32
	ShipControl ShipControl
}

func spawnSpaceship(i *Iter) {
	i.Require(PosKey)
	i.Require(RotKey)
	i.Require(SpriteKey)
	i.Require(KeepInCameraKey)
	i.Require(SpinKey)
	i.Require(MomentumKey)
	i.Require(ShipControlKey)
	i.Require(LookupKey)
	i.Require(AffectedByGravityKey)
	i.Require(NetworkIdKey)
	i.New()

	*i.Sprite() = SpriteShip
}

func spawnMissile(i *Iter) {
	i.Require(PosKey)
	i.Require(RotKey)
	i.Require(SpinKey)
	i.Require(MomentumKey)
	i.Require(SpriteKey)
	i.Require(AffectedByGravityKey)
	i.Require(NetworkIdKey)
	i.Require(MissileKey)
	i.New()

	*i.Sprite() = SpriteMissile
}

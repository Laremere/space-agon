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

import "math"

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Component types
////////////////////////////////////////////////////////////////////////////////

type Sprite uint16

const (
	SpriteUnset = Sprite(iota)
	SpriteShip
	SpriteMissile
	SpriteStar
	SpriteStarBit
)

type Vec2 [2]float32

func Vec2FromRadians(rad float32) Vec2 {
	sin, cos := math.Sincos(float64(rad))
	return Vec2{float32(cos), float32(sin)}
}

func (v Vec2) Scale(s float32) Vec2 {
	return Vec2{v[0] * s, v[1] * s}
}

func (v Vec2) Add(o Vec2) Vec2 {
	return Vec2{v[0] + o[0], v[1] + o[1]}
}

func (v *Vec2) AddEqual(o Vec2) {
	(*v)[0] += o[0]
	(*v)[1] += o[1]
}

func (v *Vec2) Length() float32 {
	x := (*v)[0]
	y := (*v)[1]
	return float32(math.Sqrt(float64(x*x + y*y)))
}

type Lookup [2]int

func (l *Lookup) Alive() bool {
	return l != nil && (*l)[0] >= 0
}

type ShipControl struct {
	Up           bool
	Down         bool
	Left         bool
	Right        bool
	Fire         bool
	FireCoolDown float32
}

// TODO: Use?
type PlayerConnectedEvent struct {
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Comp definitions, add for each new type of component.
////////////////////////////////////////////////////////////////////////////////

type Vec2Comp []Vec2

func (c *Vec2Comp) Swap(j1, j2 int) {
	(*c)[j1], (*c)[j2] = (*c)[j2], (*c)[j1]
}

func (c *Vec2Comp) Extend(i int) {
	*c = append(*c, Vec2{})
}

func (c *Vec2Comp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

type SpriteComp []Sprite

func (c *SpriteComp) Swap(j1, j2 int) {
	(*c)[j1], (*c)[j2] = (*c)[j2], (*c)[j1]
}

func (c *SpriteComp) Extend(i int) {
	*c = append(*c, SpriteUnset)
}

func (c *SpriteComp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

type FloatComp []float32

func (c *FloatComp) Swap(j1, j2 int) {
	(*c)[j1], (*c)[j2] = (*c)[j2], (*c)[j1]
}

func (c *FloatComp) Extend(i int) {
	*c = append(*c, 0)
}

func (c *FloatComp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

type LookupComp []*Lookup

func (c *LookupComp) Swap(j1, j2 int) {
	(*c)[j1], (*c)[j2] = (*c)[j2], (*c)[j1]
	(*c)[j1][1] = j1
	(*c)[j2][1] = j2
}

func (c *LookupComp) Extend(i int) {
	j := len(*c)
	*c = append(*c, &Lookup{i, j})
}

func (c *LookupComp) RemoveLast() {
	j := len(*c) - 1
	(*c)[j][0] = -2
	(*c)[j][1] = -3
	*c = (*c)[:j]
}

type ShipControlComp []ShipControl

func (c *ShipControlComp) Swap(j1, j2 int) {
	(*c)[j1], (*c)[j2] = (*c)[j2], (*c)[j1]
}

func (c *ShipControlComp) Extend(i int) {
	*c = append(*c, ShipControl{})
}

func (c *ShipControlComp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

type SpawnTypeComp []SpawnType

func (c *SpawnTypeComp) Swap(j1, j2 int) {
	(*c)[j1], (*c)[j2] = (*c)[j2], (*c)[j1]
}

func (c *SpawnTypeComp) Extend(i int) {
	*c = append(*c, 0)
}

func (c *SpawnTypeComp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

type NetworkIdComp []NetworkId

func (c *NetworkIdComp) Swap(j1, j2 int) {
	(*c)[j1], (*c)[j2] = (*c)[j2], (*c)[j1]
}

func (c *NetworkIdComp) Extend(i int) {
	*c = append(*c, 0)
}

func (c *NetworkIdComp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Pieces that need to be updated for each new component.
////////////////////////////////////////////////////////////////////////////////

const (
	// Section for keys associated with a component.
	PosKey = CompKey(iota)
	SpriteKey
	RotKey
	TimedDestroyKey
	MomentumKey
	SpinKey
	LookupKey
	ShipControlKey
	SpawnEventKey
	NetworkIdKey

	// Section for keys which are only used as tags.
	FrameEndDeleteKey
	KeepInCameraKey
	AffectedByGravityKey
	PointRenderKey

	NetworkPosTransmitKey
	NetworkRotTransmitKey
	NetworkMomentumTransmitKey
	NetworkSpinTransmitKey
	NetworkShipControlTransmitKey

	NetworkPosRecieveKey
	NetworkRotRecieveKey
	NetworkMomentumRecieveKey
	NetworkSpinRecieveKey
	NetworkShipControlRecieveKey

	doNotMoveOrUseLastKeyForNumberOfKeys
)

type EntityBag struct {
	count    int
	comps    []Comp
	compsKey compsKey

	Pos          *Vec2Comp
	Sprite       *SpriteComp
	Rot          *FloatComp
	TimedDestroy *FloatComp
	Momentum     *Vec2Comp
	Spin         *FloatComp
	Lookup       *LookupComp
	ShipControl  *ShipControlComp
	SpawnEvent   *SpawnTypeComp
	NetworkId    *NetworkIdComp
}

func newEntityBag(compsKey *compsKey) *EntityBag {
	bag := &EntityBag{
		count:    0,
		comps:    nil,
		compsKey: *compsKey,
	}

	if inRequirement(compsKey, PosKey) {
		bag.Pos = &Vec2Comp{}
		bag.comps = append(bag.comps, bag.Pos)
	}

	if inRequirement(compsKey, SpriteKey) {
		bag.Sprite = &SpriteComp{}
		bag.comps = append(bag.comps, bag.Sprite)
	}

	if inRequirement(compsKey, RotKey) {
		bag.Rot = &FloatComp{}
		bag.comps = append(bag.comps, bag.Rot)
	}

	if inRequirement(compsKey, TimedDestroyKey) {
		bag.TimedDestroy = &FloatComp{}
		bag.comps = append(bag.comps, bag.TimedDestroy)
	}

	if inRequirement(compsKey, MomentumKey) {
		bag.Momentum = &Vec2Comp{}
		bag.comps = append(bag.comps, bag.Momentum)
	}

	if inRequirement(compsKey, SpinKey) {
		bag.Spin = &FloatComp{}
		bag.comps = append(bag.comps, bag.Spin)
	}

	if inRequirement(compsKey, LookupKey) {
		bag.Lookup = &LookupComp{}
		bag.comps = append(bag.comps, bag.Lookup)
	}

	if inRequirement(compsKey, ShipControlKey) {
		bag.ShipControl = &ShipControlComp{}
		bag.comps = append(bag.comps, bag.ShipControl)
	}

	if inRequirement(compsKey, SpawnEventKey) {
		bag.SpawnEvent = &SpawnTypeComp{}
		bag.comps = append(bag.comps, bag.SpawnEvent)
	}

	if inRequirement(compsKey, NetworkIdKey) {
		bag.NetworkId = &NetworkIdComp{}
		bag.comps = append(bag.comps, bag.NetworkId)
	}

	return bag
}

func (iter *Iter) Pos() *Vec2 {
	comp := iter.e.bags[iter.i].Pos
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) Sprite() *Sprite {
	comp := iter.e.bags[iter.i].Sprite
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) Rot() *float32 {
	comp := iter.e.bags[iter.i].Rot
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) TimedDestroy() *float32 {
	comp := iter.e.bags[iter.i].TimedDestroy
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) Momentum() *Vec2 {
	comp := iter.e.bags[iter.i].Momentum
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) Spin() *float32 {
	comp := iter.e.bags[iter.i].Spin
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) Lookup() *Lookup {
	comp := iter.e.bags[iter.i].Lookup
	if comp == nil {
		return nil
	}
	return (*comp)[iter.j]
}

func (iter *Iter) ShipControl() *ShipControl {
	comp := iter.e.bags[iter.i].ShipControl
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) SpawnEvent() *SpawnType {
	comp := iter.e.bags[iter.i].SpawnEvent
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

func (iter *Iter) NetworkId() *NetworkId {
	comp := iter.e.bags[iter.i].NetworkId
	if comp == nil {
		return nil
	}
	return &(*comp)[iter.j]
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Pieces that shouldn't change due to new components.
////////////////////////////////////////////////////////////////////////////////

func inRequirement(compsKey *compsKey, compKey CompKey) bool {
	return 0 < (*compsKey)[compKey/compsKeyUnitSize]&(1<<(compKey%compsKeyUnitSize))
}

func (e *EntityBag) Add(i int) int {
	j := e.count
	e.count++
	for _, c := range e.comps {
		c.Extend(i)
	}
	return j
}

func (e *EntityBag) Remove(i int) {
	e.count--
	for _, c := range e.comps {
		c.Swap(e.count, i)
	}
}

type Iter struct {
	e            *Entities
	i            int
	j            int
	requirements compsKey
}

func (iter *Iter) Require(k CompKey) {
	iter.requirements[k/compsKeyUnitSize] |= 1 << (k % compsKeyUnitSize)
}

func (iter *Iter) Next() bool {
	iter.j++
	for iter.i == -1 || iter.j >= iter.e.bags[iter.i].count {
		for {
			iter.i++
			if iter.i >= len(iter.e.bags) {
				return false
			}
			if iter.meetsRequirements(iter.e.bags[iter.i]) {
				break
			}
		}
		iter.j = 0
	}
	return true
}

func (iter *Iter) meetsRequirements(bag *EntityBag) bool {
	for i := 0; i < len(iter.requirements); i++ {
		if iter.requirements[i] != (iter.requirements[i] & bag.compsKey[i]) {
			return false
		}
	}
	return true
}

func (iter *Iter) New() {
	var ok bool
	iter.i, ok = iter.e.bagsByKey[iter.requirements]
	if !ok {
		iter.e.bagsByKey[iter.requirements] = len(iter.e.bags)
		iter.i = len(iter.e.bags)
		iter.e.bags = append(iter.e.bags, newEntityBag(&iter.requirements))
	}

	iter.j = iter.e.bags[iter.i].Add(iter.i)
}

func (iter *Iter) Get(indices *Lookup) {
	iter.i = (*indices)[0]
	iter.j = (*indices)[1]
}

func (iter *Iter) Remove() {
	iter.e.bags[iter.i].Remove(iter.j)
	// So that a call to next will arrive at this index, which now contains  a
	// different entity.
	iter.j--
}

type CompKey uint16
type compsKey [doNotMoveOrUseLastKeyForNumberOfKeys/compsKeyUnitSize + 1]uint8

const compsKeyUnitSize = 8

type Entities struct {
	bags      []*EntityBag
	bagsByKey map[compsKey]int
}

func newEntities() *Entities {
	return &Entities{
		bagsByKey: make(map[compsKey]int),
	}
}

func (e *Entities) NewIter() *Iter {
	return &Iter{
		e: e,
		i: -1,
		j: -1,
	}
}

type Comp interface {
	Swap(j1, j2 int)
	Extend(i int)
	RemoveLast()
}

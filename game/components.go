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

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Comp definitions, add for each new type of component.
////////////////////////////////////////////////////////////////////////////////

type Vec2Comp [][2]float32

func (c *Vec2Comp) Swap(i, j int) {
	(*c)[i], (*c)[j] = (*c)[j], (*c)[i]
}

func (c *Vec2Comp) Extend() {
	*c = append(*c, [2]float32{})
}

func (c *Vec2Comp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

type SpriteComp []Sprite

func (c *SpriteComp) Swap(i, j int) {
	(*c)[i], (*c)[j] = (*c)[j], (*c)[i]
}

func (c *SpriteComp) Extend() {
	*c = append(*c, SpriteUnset)
}

func (c *SpriteComp) RemoveLast() {
	*c = (*c)[:len(*c)-1]
}

type FloatComp []float32

func (c *FloatComp) Swap(i, j int) {
	(*c)[i], (*c)[j] = (*c)[j], (*c)[i]
}

func (c *FloatComp) Extend() {
	*c = append(*c, 0)
}

func (c *FloatComp) RemoveLast() {
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

	// Section for keys which are only used as tags.
	FrameEndDeleteKey
	PlayerControlledShipKey
	KeepInCameraKey

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

	return bag
}

func (iter *Iter) Pos() *[2]float32 {
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

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Pieces that shouldn't change due to new components.
////////////////////////////////////////////////////////////////////////////////

func inRequirement(compsKey *compsKey, compKey CompKey) bool {
	return 0 < (*compsKey)[compKey/compsKeyUnitSize]&(1<<(compKey%compsKeyUnitSize))
}

func (e *EntityBag) Add() int {
	i := e.count
	e.count++
	for _, c := range e.comps {
		c.Extend()
	}
	return i
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

	iter.j = iter.e.bags[iter.i].Add()
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
	Swap(i, j int)
	Extend()
	RemoveLast()
}

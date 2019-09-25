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

type Pos struct {
	x float32
	y float32
}

type PosComp []Pos

func (c *PosComp) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c *PosComp) Append() {
	c = append(c, Pos{})
}

func (c *PosComp) Reduce() {
	c = c[:len(c)-1]
}

// type EntityBag struct {
// 	count int
// 	comps []Comp
// }

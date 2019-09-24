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

package webgl

import (
	"errors"
	"fmt"
	"syscall/js"
)

type BufferBit int
type BufferTarget js.Value
type BufferUsage js.Value
type ShaderType js.Value
type ProgramParameterBool js.Value
type ShaderParameterBool js.Value
type DrawMode js.Value
type ArrayTypes js.Value

type Buffer js.Value
type Program js.Value
type Shader js.Value
type UniformLocation js.Value
type AttribLocation js.Value

type WebGL struct {
	canvas, gl js.Value

	COLOR_BUFFER_BIT BufferBit
	DEPTH_BUFFER_BIT BufferBit

	VERTEX_SHADER   ShaderType
	FRAGMENT_SHADER ShaderType

	ARRAY_BUFFER BufferTarget
	STATIC_DRAW  BufferUsage
	DYNAMIC_DRAW BufferUsage

	LINK_STATUS    ProgramParameterBool
	COMPILE_STATUS ShaderParameterBool

	TRIANGLES DrawMode

	FLOAT ArrayTypes
	SHORT ArrayTypes
}

func InitWebgl(canvas js.Value) (*WebGL, error) {
	w := WebGL{}

	w.canvas = canvas
	w.gl = canvas.Call("getContext", "webgl")

	if w.gl == js.Null() {
		return nil, errors.New("Creating a webgl context is not supported.  This won't work.")
	}

	w.COLOR_BUFFER_BIT = BufferBit(w.gl.Get("COLOR_BUFFER_BIT").Int())
	w.DEPTH_BUFFER_BIT = BufferBit(w.gl.Get("DEPTH_BUFFER_BIT").Int())

	w.ARRAY_BUFFER = BufferTarget(w.gl.Get("ARRAY_BUFFER"))

	w.STATIC_DRAW = BufferUsage(w.gl.Get("STATIC_DRAW"))
	w.DYNAMIC_DRAW = BufferUsage(w.gl.Get("DYNAMIC_DRAW"))

	w.LINK_STATUS = ProgramParameterBool(w.gl.Get("LINK_STATUS"))
	w.COMPILE_STATUS = ShaderParameterBool(w.gl.Get("COMPILE_STATUS"))

	w.VERTEX_SHADER = ShaderType(w.gl.Get("VERTEX_SHADER"))
	w.FRAGMENT_SHADER = ShaderType(w.gl.Get("FRAGMENT_SHADER"))

	w.TRIANGLES = DrawMode(w.gl.Get("TRIANGLES"))

	w.FLOAT = ArrayTypes(w.gl.Get("FLOAT"))
	w.SHORT = ArrayTypes(w.gl.Get("SHORT"))

	return &w, nil
}

func (w *WebGL) ClearColor(r, g, b, a float64) {
	w.gl.Call("clearColor", r, g, b, a)
}

func (w *WebGL) Clear(colorBits BufferBit) {
	w.gl.Call("clear", int(colorBits))
}

/////////////////////////////////////////////
// Buffer
/////////////////////////////////////////////

func (w *WebGL) CreateBuffer() *Buffer {
	b := Buffer(w.gl.Call("createBuffer"))
	return &b
}

func (w *WebGL) BindBuffer(t BufferTarget, b *Buffer) {
	w.gl.Call("bindBuffer", js.Value(t), js.Value(*b))
}

func (w *WebGL) BufferDataF32(t BufferTarget, a []float32, u BufferUsage) {
	w.gl.Call("bufferData", js.Value(t), copyFloat32SliceToJS(a), js.Value(u))
}

func (w *WebGL) BufferDataSize(t BufferTarget, size int, u BufferUsage) {
	w.gl.Call("bufferData", js.Value(t), size, js.Value(u))
}

func (w *WebGL) BufferSubDataI16(t BufferTarget, offset int, a []int16) {
	w.gl.Call("bufferSubData", js.Value(t), offset, copyInt16SliceToJS(a))
}

/////////////////////////////////////////////
// Program
/////////////////////////////////////////////

func (w *WebGL) CreateProgram() *Program {
	p := Program(w.gl.Call("createProgram"))
	return &p
}

func (w *WebGL) AttachShader(p *Program, s *Shader) {
	w.gl.Call("attachShader", js.Value(*p), js.Value(*s))
}

func (w *WebGL) LinkProgram(p *Program) {
	w.gl.Call("linkProgram", js.Value(*p))
}

func (w *WebGL) GetProgramParameterBool(p *Program, param ProgramParameterBool) bool {
	return w.gl.Call("getProgramParameter", js.Value(*p), js.Value(param)).Bool()
}

func (w *WebGL) GetProgramInfoLog(p *Program) string {
	return w.gl.Call("getProgramInfoLog", js.Value(*p)).String()
}

func (w *WebGL) UseProgram(p *Program) {
	w.gl.Call("useProgram", js.Value(*p))
}

func (w *WebGL) GetUniformLocation(p *Program, name string) UniformLocation {
	return UniformLocation(w.gl.Call("getUniformLocation", js.Value(*p), name))
}

func (w *WebGL) GetAttribLocation(p *Program, name string) AttribLocation {
	return AttribLocation(w.gl.Call("getAttribLocation", js.Value(*p), name))
}

/////////////////////////////////////////////
// Shader
/////////////////////////////////////////////

func (w *WebGL) CreateShader(t ShaderType) *Shader {
	s := Shader(w.gl.Call("createShader", js.Value(t)))
	return &s
}

func (w *WebGL) ShaderSource(s *Shader, code string) {
	w.gl.Call("shaderSource", js.Value(*s), code)
}

func (w *WebGL) CompileShader(s *Shader) {
	w.gl.Call("compileShader", js.Value(*s))
}

func (w *WebGL) GetShaderParameterBool(s *Shader, param ShaderParameterBool) bool {
	return w.gl.Call("getShaderParameter", js.Value(*s), js.Value(param)).Bool()
}

func (w *WebGL) GetShaderInfoLog(s *Shader) string {
	return w.gl.Call("getShaderInfoLog", js.Value(*s)).String()
}

/////////////////////////////////////////////
// Shader parameters and draw calls
/////////////////////////////////////////////

func (w *WebGL) Uniform2fv(u UniformLocation, v [2]float64) {
	w.gl.Call("uniform2fv", js.Value(u), copyFloat64SliceToJS(v[:]))
}

func (w *WebGL) Uniform4fv(u UniformLocation, v [4]float64) {
	w.gl.Call("uniform4fv", js.Value(u), copyFloat64SliceToJS(v[:]))
}

func (w *WebGL) EnableVertexAttribArray(a AttribLocation) {
	w.gl.Call("enableVertexAttribArray", js.Value(a))
}

func (w *WebGL) VertexAttribPointer(
	a AttribLocation, size int, arrayTypes ArrayTypes, normalized bool, stride, offset int64) {

	w.gl.Call("vertexAttribPointer", js.Value(a), size, js.Value(arrayTypes), normalized, stride, offset)
}

func (w *WebGL) DrawArrays(mode DrawMode, first, size int) {
	w.gl.Call("drawArrays", js.Value(mode), first, size)
}

/////////////////////////////////////////////
// Utility functions (anything that wouldn't be on an OpenGL spec)
/////////////////////////////////////////////

func CreateProgram(w *WebGL, vertexShaderCode, fragmentShaderCode string) (*Program, error) {
	vertexShader, err := CompileShader(w, vertexShaderCode, w.VERTEX_SHADER)
	if err != nil {
		return nil, fmt.Errorf("Error compiling vertex shader: %s", err)
	}

	fragmentShader, err := CompileShader(w, fragmentShaderCode, w.FRAGMENT_SHADER)
	if err != nil {
		return nil, fmt.Errorf("Error compiling fragment shader: %s", err)
	}

	p := w.CreateProgram()
	w.AttachShader(p, vertexShader)
	w.AttachShader(p, fragmentShader)
	w.LinkProgram(p)

	if !w.GetProgramParameterBool(p, w.LINK_STATUS) {
		return nil, fmt.Errorf("Error linking shader: %s", err)
	}

	return p, nil
}

func CompileShader(w *WebGL, code string, t ShaderType) (*Shader, error) {
	s := w.CreateShader(t)

	w.ShaderSource(s, code)
	w.CompileShader(s)

	if !w.GetShaderParameterBool(s, w.COMPILE_STATUS) {
		return nil, fmt.Errorf("Error Compiling Shader: %s", w.GetShaderInfoLog(s))
	}
	return s, nil
}

// TODO: Javascript types arrays are backed by byte buffers.  It may be faster
// to convert the golang slice into a byte buffer, and use js.CopyBytesToJS.
// However there are possible endian issues.  I believe this method must be used
// to properly trasmit 64 bit numbers, and the conversion to and from floats is
// lossy.
func copyFloat64SliceToJS(a []float64) js.Value {
	const bytesPerValue = 64 / 8
	global := js.Global()
	r := global.New("Float64Array", global.New("ArrayBuffer", js.ValueOf(len(a)*bytesPerValue)))
	for i, v := range a {
		r.SetIndex(i, js.ValueOf(v))
	}
	return r
}

func copyFloat32SliceToJS(a []float32) js.Value {
	const bytesPerValue = 32 / 8
	global := js.Global()
	r := global.New("Float32Array", global.New("ArrayBuffer", js.ValueOf(len(a)*bytesPerValue)))
	for i, v := range a {
		r.SetIndex(i, js.ValueOf(v))
	}
	return r
}

func copyInt16SliceToJS(a []int16) js.Value {
	const bytesPerValue = 16 / 8
	global := js.Global()
	r := global.New("Int16Array", global.New("ArrayBuffer", js.ValueOf(len(a)*bytesPerValue)))
	for i, v := range a {
		r.SetIndex(i, js.ValueOf(v))
	}
	return r
}

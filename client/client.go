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
	"errors"
	"io"
	"log"
	"sync"
	"syscall/js"

	"github.com/googleforgames/space-agon/game"
	"github.com/googleforgames/space-agon/game/pb"
	"github.com/googleforgames/space-agon/game/protostream"
	"google.golang.org/grpc/status"
	ompb "open-match.dev/open-match/pkg/pb"
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
	inp := game.NewInput()
	inp.IsRendered = true
	inp.IsPlayer = true
	js.Global().Get("document").Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// log.Println("keydown", args[0].Get("code").String())
		switch args[0].Get("code").String() {
		case "ArrowUp":
			inp.Up.Down()
		case "ArrowLeft":
			inp.Left.Down()
		case "ArrowRight":
			inp.Right.Down()
		case "ArrowDown":
			inp.Down.Down()
		case "Space":
			inp.Fire.Down()
		}
		return nil
	}))
	js.Global().Get("document").Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// log.Println("keyup", args[0].Get("code").String())
		switch args[0].Get("code").String() {
		case "ArrowUp":
			inp.Up.Up()
		case "ArrowLeft":
			inp.Left.Up()
		case "ArrowRight":
			inp.Right.Up()
		case "ArrowDown":
			inp.Down.Up()
		case "Space":
			inp.Fire.Up()
		}
		return nil
	}))

	log.Println("Initiating Graphics.")
	gr, err := NewGraphics()
	if err != nil {
		return nil, err
	}

	c := &client{
		gr:            gr,
		g:             game.NewGame(),
		inp:           inp,
		lastTimestamp: js.Global().Get("performance").Call("now").Float(),
	}

	js.Global().Set("connectThis", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		addr := js.Global().Get("window").Get("location").Get("host").String()
		c.connect(addr)
		return nil
	}))

	js.Global().Set("matchmake", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.matchmake()
		return nil
	}))

	js.Global().Set("connect", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		addr := args[0].String()
		c.connect(addr)
		return nil
	}))

	return c, nil
}

type client struct {
	gr            *graphics
	g             *game.Game
	inp           *game.Input
	lastTimestamp float64
	lock          sync.Mutex
	sending       chan []*pb.Memo
	recieving     chan []*pb.Memo
}

func (c *client) connect(addr string) {
	wws, err := NewWrappedWebSocket("ws://" + addr + "/connect/")
	if err != nil {
		log.Fatal(err)
	}
	stream := protostream.NewProtoStream(wws)

	go func() {
		{
			clientInitialize := &pb.ClientInitialize{}
			err := stream.Recv(clientInitialize)
			if err != nil {
				log.Println("Failed to initialize client:", err)
			}

			c.lock.Lock()
			defer c.lock.Unlock()

			c.inp.IsConnected = true
			c.inp.Cid = clientInitialize.Cid
			c.g = game.NewGame()

			if c.sending != nil {
				close(c.sending)
			}
			c.sending = make(chan []*pb.Memo, 1)
			c.recieving = make(chan []*pb.Memo, 1)

		}
		go func() {
			var err error
			for err == nil {
				toSend := <-c.sending
				err = stream.Send(&pb.Memos{Memos: toSend})
			}
			log.Println("Error sending memos:", err)
			for range c.sending {
			}
		}()

		go func() {
			for {
				memos := &pb.Memos{}
				err := stream.Recv(memos)
				if err != nil {
					log.Println("Error recieving from stream: ", err.Error())
					return
				}
				combineToSend(c.recieving, memos.Memos)
			}
		}()
	}()
}

func (c *client) matchmake() {
	addr := js.Global().Get("window").Get("location").Get("host").String()

	wws, err := NewWrappedWebSocket("ws://" + addr + "/matchmake/")
	if err != nil {
		log.Fatal(err)
	}
	stream := protostream.NewProtoStream(wws)
	go func() {
		defer wws.Close()
		for {
			a := &ompb.Assignment{}
			err := stream.Recv(a)
			if err != nil {
				log.Println("Error receiving assignment:", err)
			}

			if a.Error != nil {
				err := status.FromProto(a.Error).Err()
				if err != nil {
					// TODO: Display error
					log.Println("Error on assignment:", err)
					return
				}
			}

			if a.Connection != "" {
				go c.connect(a.Connection)
				return
			}
		}
	}()
}

func (c *client) scheduleFrame() {
	js.Global().Get("window").Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.lock.Lock()
		defer c.lock.Unlock()
		now := args[0].Float()
		c.inp.Dt = float32((now - c.lastTimestamp) / 1000)
		c.lastTimestamp = now

		select {
		case c.inp.Memos = <-c.recieving:
		default:
			c.inp.Memos = nil
		}

		c.frame()

		if c.sending != nil {
			combineToSend(c.sending, c.inp.MemosOut)
		}
		c.inp.MemosOut = nil

		return nil
	}))
}

var rotation = float32(0)

func (c *client) frame() {
	const maximumStep = float32(1) / 20
	for c.inp.Dt > maximumStep {
		actualDt := c.inp.Dt
		c.inp.Dt = maximumStep
		c.g.Step(c.inp)
		c.inp.Dt = actualDt - maximumStep
	}
	c.g.Step(c.inp)

	c.gr.Clear()
	{
		i := c.g.E.NewIter()
		i.Require(game.PosKey)
		i.Require(game.KeepInCameraKey)

		xMin := float32(-10)
		yMin := float32(-10)
		xMax := float32(10)
		yMax := float32(10)

		for i.Next() {
			x := (*i.Pos())[0]
			y := (*i.Pos())[1]
			boundary := float32(0)

			sprite := i.Sprite()
			if sprite != nil {
				boundary = spritemap[*sprite].size * 2
			}

			if x-boundary < xMin {
				xMin = x - boundary
			}
			if x+boundary > xMax {
				xMax = x + boundary
			}
			if y-boundary < yMin {
				yMin = y - boundary
			}
			if y+boundary > yMax {
				yMax = y + boundary
			}
		}

		c.gr.SetCamera(xMin, yMin, xMax, yMax)
	}

	{
		i := c.g.E.NewIter()
		i.Require(game.PosKey)
		i.Require(game.PointRenderKey)
		for i.Next() {
			p := *i.Pos()
			c.gr.Point(p[0], p[1])
		}
	}

	{
		i := c.g.E.NewIter()
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

	c.inp.FrameEndReset()

	c.scheduleFrame()
}

//////////////////////////////////////////////////////
//////////////////////////////////////////////////////
//////////////////////////////////////////////////////

type WrappedWebSocket struct {
	r         *io.PipeReader
	w         *io.PipeWriter
	ws        js.Value
	open      chan struct{}
	closeLock sync.Mutex
	closed    chan struct{}
}

var errWritingOnClosed = errors.New("Writing on closed WrappedWebSocket")

func NewWrappedWebSocket(addr string) (*WrappedWebSocket, error) {
	ws := js.Global().Get("WebSocket").New(addr)

	r, w := io.Pipe()
	wws := &WrappedWebSocket{
		r:      r,
		w:      w,
		ws:     ws,
		open:   make(chan struct{}),
		closed: make(chan struct{}),
	}

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		close(wws.open)
		return nil
	}))

	blobCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		data := js.Global().Get("Uint8Array").New(args[0])
		length := data.Length()
		b := make([]byte, length)

		js.CopyBytesToGo(b, data)

		_, err := wws.w.Write(b)
		if err != nil {
			log.Println("Error in onmessage on ", addr, ":", err)
		}
		return nil
	})

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		data := args[0].Get("data")
		data.Call("arrayBuffer").Call("then", blobCallback)

		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Websocket error: addr=", addr, " error=", args[0].Call("toString"))
		wws.Close()
		select {
		case <-wws.open:
		default:
			close(wws.open)
		}
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Websocket closed: ", addr)
		wws.Close()
		return nil
	}))

	// Consider: waiting for open? However then this caller would need to ensure
	// that it doesn't block JS's event loop.

	return wws, nil
}

func (wws *WrappedWebSocket) Read(b []byte) (n int, err error) {
	return wws.r.Read(b)
}

func (wws *WrappedWebSocket) Write(b []byte) (n int, err error) {
	defer func() {
		r := recover()
		if r != nil {
			panic("Recovered in write!")
		}
	}()

	select {
	case <-wws.open:
	}

	select {
	case <-wws.closed:
		return 0, errWritingOnClosed
	default:
	}

	data := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(data, b)

	wws.ws.Call("send", data)
	return len(b), nil
}

func (wws *WrappedWebSocket) Close() error {
	wws.closeLock.Lock()
	defer wws.closeLock.Unlock()

	select {
	case <-wws.closed:
		return nil
	default:
	}

	wws.w.Close()
	close(wws.closed)
	wws.ws.Call("close")
	return nil
}

func combineToSend(c chan []*pb.Memo, memos []*pb.Memo) {
	select {
	case previousMemos := <-c:
		previousMemos = append(previousMemos, memos...)
		c <- previousMemos
	case c <- memos:
	}
}

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
	"log"
	"sync"
	"syscall/js"

	"github.com/golang/protobuf/proto"
	"github.com/googleforgames/space-agon/game"
	"github.com/googleforgames/space-agon/game/pb"
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
	initialized   bool
}

func (c *client) connect(addr string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.initialized = false

	if c.sending != nil {
		close(c.sending)
	}

	c.sending = make(chan []*pb.Memo, 1)
	c.recieving = make(chan []*pb.Memo, 1)

	// for id, conn := range c.inp.Conns {
	// 	close(conn.Sending)
	// 	delete(c.inp.Conns, id)
	// }

	// serverConn := game.NewNetworkConnection()
	// c.inp.Conns[0] = serverConn
	ws := js.Global().Get("WebSocket").New("ws://" + addr + "/connect/")

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Websocket onopen!", args[0].Get("toString"))
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// log.Println("Websocket onmessage!", args[0].Get("data").String())
		if !c.initialized {
			clientInitialize := &pb.ClientInitialize{}
			buf := proto.NewBuffer([]byte(args[0].Get("data").String()))
			err := buf.DecodeMessage(clientInitialize)
			if err != nil {
				log.Printf("Recieving had Unmarshal error %v", err)
				// TOOD: Disconnect, display error, etc.
				return nil
			}

			c.initialize(ws, clientInitialize)
			return nil
		}

		memos := &pb.Memos{}

		buf := proto.NewBuffer([]byte(args[0].Get("data").String()))
		err := buf.DecodeMessage(memos)
		if err != nil {
			log.Printf("Recieving had Unmarshal error %v", err)
			// TOOD: Disconnect, display error, etc.
			return nil
		}

		combineToSend(c.recieving, memos.Memos)

		// v := game.NewNetworkUpdate()
		// err := json.Unmarshal([]byte(args[0].Get("data").String()), v)
		// if err != nil {
		// 	log.Printf("Recieving had Unmarshal error %v", err)
		// 	return nil
		// }
		// game.NetworkUpdateCombineAndPass(serverConn.Recieving, v)
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Websocket error!", args[0].Call("toString"))
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Websocket closed!", args[0].Call("toString"))
		return nil
	}))

	c.inp.IsConnected = true
}

func (c *client) initialize(ws js.Value, clientInitialize *pb.ClientInitialize) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.initialized = true
	c.inp.Cid = clientInitialize.Cid

	c.g = game.NewGame()

	go func() {
		buf := proto.NewBuffer(nil)

		for toSend := range c.sending {
			err := buf.EncodeMessage(&pb.Memos{Memos: toSend})
			if err != nil {
				log.Printf("Sending had Marshal error %v", err)
				// TODO: Disconnect, display error, etc.
				return
			}
			ws.Call("send", string(buf.Bytes()))

			buf.Reset()
		}

		// for toSend := range serverConn.Sending {
		// 	b, err := json.Marshal(toSend)
		// 	if err != nil {
		// 		log.Printf("Sending had Marshal error %v", err)
		// 		return
		// 	}
		// 	ws.Call("send", string(b))
		// }
	}()

}

func (c *client) matchmake() {
	addr := js.Global().Get("window").Get("location").Get("host").String()
	ws := js.Global().Get("WebSocket").New("ws://" + addr + "/matchmake/")

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Matchmaking Websocket onopen!", args[0].Get("toString"))
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Matchmaking Websocket onmessage!")

		a := &ompb.Assignment{}
		err := proto.Unmarshal([]byte(args[0].Get("data").String()), a)
		if err != nil {
			log.Println("Error Unmarshaling assignment:", err)
			return nil
		}

		if a.Error != nil {
			err := status.FromProto(a.Error).Err()
			if err != nil {
				// TODO: Display error
				log.Println("Error on assignment:", err)
				return nil
			}
		}

		if a.Connection != "" {
			ws.Call("close")
			c.connect(a.Connection)
		}
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Matchmaking Websocket error!", args[0].Call("toString"))
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("Matchmaking Websocket closed!", args[0].Call("toString"))
		return nil
	}))
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

	// count := 0
	{
		i := c.g.E.NewIter()
		i.Require(game.PosKey)
		i.Require(game.SpriteKey)
		for i.Next() {
			// count++
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

type WrappedWebSocket struct {
	open     chan struct{}
	closed   chan struct{}
	hasError chan struct{}
	messages chan []byte
	err      error
}

// func NewWrappedWebSocket(addr string) (*WrappedWebSocket, error) {
// 	w := &WrappedWebSocket{
// 		open:   make(chan struct{}),
// 		closed: make(chan struct{}),
// 		hasErr: make(chan struct{}),
// 	}

// 	ws := js.Global().Get("WebSocket").New(addr)

// 	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
// 		close(w.open)
// 		// go func() {
// 		// 	log.Println("Websocket onopen!", args[0].Get("toString"))

// 		// 	for toSend := range serverConn.Sending {
// 		// 		b, err := json.Marshal(toSend)
// 		// 		if err != nil {
// 		// 			log.Printf("Sending had Marshal error %v", err)
// 		// 			return
// 		// 		}
// 		// 		ws.Call("send", string(b))
// 		// 	}
// 		// }()
// 		return nil
// 	}))

// 	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
// 		// // log.Println("Websocket onmessage!", args[0].Get("data").String())

// 		// v := game.NewNetworkUpdate()
// 		// err := json.Unmarshal([]byte(args[0].Get("data").String()), v)
// 		// if err != nil {
// 		// 	log.Printf("Recieving had Unmarshal error %v", err)
// 		// 	return nil
// 		// }
// 		// game.NetworkUpdateCombineAndPass(serverConn.Recieving, v)
// 		return nil
// 	}))

// 	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
// 		// log.Println("Websocket error!", args[0].Call("toString"))
// 		return nil
// 	}))

// 	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
// 		// log.Println("Websocket closed!", args[0].Call("toString"))
// 		return nil
// 	}))

// 	switch {
// 	case <-w.open:
// 		return w, nil
// 	case <-w.hasErr:
// 		return nil, w.err
// 	}
// }

// func (w *WrappedWebSocket) Send(b []byte) error {

// }

// func (w *WrappedWebSocket) Recv() ([]byte, error) {

// }

func combineToSend(c chan []*pb.Memo, memos []*pb.Memo) {
	select {
	case previousMemos := <-c:
		previousMemos = append(previousMemos, memos...)
		c <- previousMemos
	case c <- memos:
	}
}

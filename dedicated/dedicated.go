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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"sync"
	"time"

	agonesSdk "agones.dev/agones/pkg/sdk"
	agones "agones.dev/agones/sdks/go"
	"github.com/googleforgames/space-agon/game"
	"golang.org/x/net/websocket"
)

func main() {
	a, err := agones.NewSDK()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		time.Sleep(3)
		a.Ready()
		for range time.Tick(time.Second) {
			a.Health()
		}
	}()

	http.Handle("/connect/", Start(a))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Println("Starting dedicated server")
	log.Fatal(http.ListenAndServe(":2156", nil))
}

func Start(a *agones.SDK) websocket.Handler {
	d := &dedicated{
		g:      game.NewGame(),
		nextId: make(chan int, 1),
		inp:    game.NewInput(),
		a:      a,
	}
	d.inp.IsRendered = false
	d.inp.IsPlayer = false
	d.inp.IsHost = false

	d.nextId <- 0

	go func() {
		time.Sleep(15 * time.Second)
		last := time.Now()
		for t := range time.Tick(time.Second / 60) {
			d.lock.Lock()
			d.inp.Dt = float32(t.Sub(last).Seconds())
			d.g.Step(d.inp)
			d.lock.Unlock()
		}
	}()

	a.WatchGameServer(d.watchGameServer)

	return d.Handler
}

type dedicated struct {
	g *game.Game

	lock sync.Mutex
	inp  *game.Input

	sendChannel map[int]chan byte

	nextId chan int

	a                 *agones.SDK
	shutdown          sync.Once
	waitForDisconnect sync.WaitGroup
}

func (d *dedicated) Handler(c *websocket.Conn) {
	d.waitForDisconnect.Add(1)
	defer d.waitForDisconnect.Done()

	ctx, cancel := context.WithCancel(context.Background())
	n := game.NewNetworkConnection()

	id := <-d.nextId
	d.nextId <- id + 1

	firstTransmit := game.NewNetworkUpdate()
	d.lock.Lock()
	d.inp.Conns[id] = n
	firstTransmit.AndThen(d.g.NewClientUpdate)
	n.Sending <- firstTransmit
	d.lock.Unlock()

	go func() {
		defer cancel()
		e := json.NewEncoder(c)
		for {
			select {
			case u := <-n.Sending:
				err := e.Encode(u)
				if err != nil {
					log.Printf("Client %d had write error %v", id, err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer cancel()
		d := json.NewDecoder(c)
		for {
			u := &game.NetworkUpdate{}
			err := d.Decode(u)
			if err != nil {
				if err != nil {
					log.Printf("Client %d had read/decode error %v", id, err)
					return
				}
			}
			game.NetworkUpdateCombineAndPass(n.Recieving, u)
		}
	}()

	<-ctx.Done()
}

func (d *dedicated) watchGameServer(gs *agonesSdk.GameServer) {
	if gs.GetStatus().GetState() == "Allocated" {
		d.shutdown.Do(func() {
			log.Println("Detected the server is allocated.")
			time.Sleep(time.Second * 15)
			log.Println("Waiting for players to disconnect then shutting down.")
			d.waitForDisconnect.Wait()
			d.a.Shutdown()
		})
	}
}

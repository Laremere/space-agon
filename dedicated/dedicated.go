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
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	agonesSdk "agones.dev/agones/pkg/sdk"
	agones "agones.dev/agones/sdks/go"
	"github.com/golang/protobuf/proto"
	"github.com/googleforgames/space-agon/game"
	"github.com/googleforgames/space-agon/game/pb"
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
		g:       game.NewGame(),
		nextCid: make(chan int64, 1),
		inp:     game.NewInput(),
		mr:      newMemoRouter(),
		a:       a,
	}
	d.inp.IsRendered = false
	d.inp.IsPlayer = false
	d.inp.IsHost = false

	d.nextCid <- 1

	go func() {
		time.Sleep(15 * time.Second)
		last := time.Now()
		for t := range time.Tick(time.Second / 60) {
			d.inp.Dt = float32(t.Sub(last).Seconds())
			d.g.Step(d.inp)
		}
	}()

	a.WatchGameServer(d.watchGameServer)

	return d.Handler
}

type dedicated struct {
	g *game.Game

	inp *game.Input

	// sendChannel map[int]chan byte

	nextCid chan int64

	mr *memoRouter

	a                 *agones.SDK
	shutdown          sync.Once
	waitForDisconnect sync.WaitGroup
}

func (d *dedicated) Handler(c *websocket.Conn) {
	d.waitForDisconnect.Add(1)
	defer d.waitForDisconnect.Done()

	ctx, cancel := context.WithCancel(context.Background())
	// n := game.NewNetworkConnection()

	cid := <-d.nextCid
	d.nextCid <- cid + 1

	toSend, recieve := d.mr.connect(cid)
	defer d.mr.disconnect(cid)

	// firstTransmit := game.NewNetworkUpdate()
	// d.lock.Lock()
	// d.inp.Conns[id] = n
	// firstTransmit.AndThen(d.g.NewClientUpdate)
	// n.Sending <- firstTransmit
	// d.lock.Unlock()

	go func() {
		defer cancel()
		// e := json.NewEncoder(c)

		buf := proto.NewBuffer(nil)

		err := buf.EncodeMessage(&pb.ClientInitialize{Cid: cid})
		if err != nil {
			log.Printf("Client %d had marshal error %v", cid, err)
			return
		}
		_, err = c.Write(buf.Bytes())
		if err != nil {
			log.Printf("Client %d had write error %v", cid, err)
			return
		}
		buf.Reset()

		for {
			select {
			case memos := <-toSend:
				err := buf.EncodeMessage(&pb.Memos{Memos: memos})
				if err != nil {
					log.Printf("Client %d had marshal error %v", cid, err)
					return
				}
				_, err = c.Write(buf.Bytes())
				if err != nil {
					log.Printf("Client %d had write error %v", cid, err)
					return
				}
				buf.Reset()
			// case u := <-n.Sending:
			// err := e.Encode(u)
			// if err != nil {
			// 	log.Printf("Client %d had write error %v", cid, err)
			// 	return
			// }
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer cancel()
		// d := json.NewDecoder(c)
		r := newProtoReader(c)
		for {
			memos := &pb.Memos{}
			err := r.Unmarshal(memos)
			if err != nil {
				if err != nil {
					log.Printf("Client %d had read/decode error %v", cid, err)
					return
				}
			}
			recieve(memos.Memos)

			// u := &game.NetworkUpdate{}
			// err := d.Decode(u)
			// if err != nil {
			// 	if err != nil {
			// 		log.Printf("Client %d had read/decode error %v", cid, err)
			// 		return
			// 	}
			// }
			// game.NetworkUpdateCombineAndPass(n.Recieving, u)
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

func combineToSend(c chan []*pb.Memo, memos []*pb.Memo) {
	select {
	case previousMemos := <-c:
		previousMemos = append(previousMemos, memos...)
		c <- previousMemos
	case c <- memos:
	}
}

type memoRouter struct {
	incoming     chan []*pb.Memo
	outgoing     map[int64]chan []*pb.Memo
	outgoingLock sync.Mutex
}

func newMemoRouter() *memoRouter {
	mr := &memoRouter{
		incoming: make(chan []*pb.Memo, 1),
		outgoing: make(map[int64]chan []*pb.Memo),
	}

	go func() {
		for memos := range mr.incoming {
			mr.outgoingLock.Lock()
			pending := make(map[int64][]*pb.Memo)
			for _, memo := range memos {
				for cid := range mr.outgoing {
					if isMemoRecipient(cid, memo) {
						pending[cid] = append(pending[cid], memo)
					}
				}
			}

			for cid, c := range mr.outgoing {
				combineToSend(c, pending[cid])
			}
			mr.outgoingLock.Unlock()
		}
	}()

	return mr
}

// TODO: Being lazy with client and server message passing: the clients, when
// sending a message to themselves (including broadcasts) should directly send
// themselves the message.  So then the server here should take care to not
// send it back to that client (so it doesn't get the same message twice).
// Though also the server currently sends messages to itself through this router.
func (mr *memoRouter) connect(cid int64) (toSend chan []*pb.Memo, recieve func([]*pb.Memo)) {
	mr.outgoingLock.Lock()
	defer mr.outgoingLock.Unlock()

	if _, ok := mr.outgoing[cid]; ok {
		panic("Cid connected twice?")
	}

	toSend = make(chan []*pb.Memo, 1)
	mr.outgoing[cid] = toSend

	recieve = func(memos []*pb.Memo) {
		combineToSend(mr.incoming, memos)
	}

	return toSend, recieve
}

func (mr *memoRouter) disconnect(cid int64) {
	// TODO: send disconnect memo
	mr.outgoingLock.Lock()
	defer mr.outgoingLock.Unlock()

	delete(mr.outgoing, cid)
}

func isMemoRecipient(cid int64, memo *pb.Memo) bool {
	switch r := memo.Recipient.(type) {
	case *pb.Memo_To:
		return cid == r.To
	case *pb.Memo_EveryoneBut:
		return cid != r.EveryoneBut
	case *pb.Memo_Everyone:
		return true
	}
	panic("Unknown recipient type")
}

type protoReader struct {
	r     io.Reader
	b     []byte
	start int
	end   int
}

func newProtoReader(r io.Reader) *protoReader {
	return &protoReader{
		r:     r,
		start: 0,
		end:   0,
	}
}

func (p *protoReader) Unmarshal(m proto.Message) error {
	messageSize := 0
	for {
		varIntVal, varintSize := proto.DecodeVarint(p.b[p.start:p.end])
		if varintSize > 0 {
			p.start += varintSize
			messageSize = int(varIntVal)
			break
		}
		err := p.fill(10) // Max size of a varint
		if err != nil {
			return err
		}
	}

	for p.end-p.start < messageSize {
		err := p.fill(messageSize)
		if err != nil {
			return err
		}
	}

	err := proto.Unmarshal(p.b[p.start:p.start+messageSize], m)

	p.start += messageSize
	if p.start == p.end {
		p.start = 0
		p.end = 0
	}
	return err
}

// Will read so that (p.end - p.start) <= target.  However it tries to read
// so that (p.end - p.start) == target.  It will only not do so if a given read
// does not return enough bytes.
func (p *protoReader) fill(target int) error {
	if p.end >= p.start+target {
		return nil
	}
	if p.start+target > len(p.b) {
		if len(p.b) < target {
			old := p.b[p.start:p.end]
			p.b = make([]byte, target)
			copy(p.b, old)
		} else {
			copy(p.b, p.b[p.start:p.end])
		}
	}

	n, err := p.r.Read(p.b[p.end : p.start+target])
	if err != nil {
		return err
	}
	p.end += n
	return nil
}

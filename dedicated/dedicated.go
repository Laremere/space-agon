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
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	agonesSdk "agones.dev/agones/pkg/sdk"
	agones "agones.dev/agones/sdks/go"
	"github.com/googleforgames/space-agon/game"
	"github.com/googleforgames/space-agon/game/pb"
	"github.com/googleforgames/space-agon/game/protostream"
	"golang.org/x/net/websocket"
)

func main() {
	log.Println("Initializing dedicated server")

	waitForEmpty := startAgones()

	http.Handle("/connect/", newDedicated(waitForEmpty))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Println("Starting dedicated server")
	log.Fatal(http.ListenAndServe(":2156", nil))
}

type dedicated struct {
	g *game.Game

	inp *game.Input

	nextCid chan int64

	mr *memoRouter

	waitForEmpty *sync.WaitGroup
}

func newDedicated(waitForEmpty *sync.WaitGroup) websocket.Handler {
	d := &dedicated{
		g:            game.NewGame(),
		nextCid:      make(chan int64, 1),
		inp:          game.NewInput(),
		mr:           newMemoRouter(),
		waitForEmpty: waitForEmpty,
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

	return d.Handler
}

func (d *dedicated) Handler(c *websocket.Conn) {
	c.PayloadType = 2 // Sets sent payloads to binary

	d.waitForEmpty.Add(1)
	defer d.waitForEmpty.Done()

	ctx, cancel := context.WithCancel(context.Background())
	// n := game.NewNetworkConnection()

	cid := <-d.nextCid
	d.nextCid <- cid + 1

	toSend, recieve := d.mr.connect(cid)
	defer d.mr.disconnect(cid)

	stream := protostream.NewProtoStream(c)

	go func() {
		defer cancel()
		err := stream.Send(&pb.ClientInitialize{Cid: cid})
		if err != nil {
			log.Printf("Client %d had send clientInitialize error %v", cid, err)
			return
		}

		for {
			select {
			case memos := <-toSend:
				err := stream.Send(&pb.Memos{Memos: memos})
				if err != nil {
					log.Printf("Client %d had send memos error %v", cid, err)
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer cancel()
		for {
			memos := &pb.Memos{}
			err := stream.Recv(memos)
			if err != nil {
				log.Printf("Client %d had read/decode error %v", cid, err)
				return
			}
			recieve(memos.Memos)
		}
	}()

	<-ctx.Done()
}

///////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////

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
	createMemos  map[uint64]*pb.Memo
}

func newMemoRouter() *memoRouter {
	mr := &memoRouter{
		incoming: make(chan []*pb.Memo, 1),
		outgoing: make(map[int64]chan []*pb.Memo),

		createMemos: make(map[uint64]*pb.Memo),
	}

	go func() {
		for memos := range mr.incoming {
			mr.outgoingLock.Lock()

			pending := make(map[int64][]*pb.Memo)
			for _, memo := range memos {

				switch a := memo.Actual.(type) {
				case *pb.Memo_SpawnEvent:
					actual := a.SpawnEvent
					mr.createMemos[actual.Nid] = memo
				case *pb.Memo_DestroyEvent:
					actual := a.DestroyEvent
					delete(mr.createMemos, actual.Nid)
				}

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
// sending a message to themselves (including broadcasts) should directly senda
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

	memos := []*pb.Memo{}
	for _, memo := range mr.createMemos {
		memos = append(memos, memo)
	}
	toSend <- memos

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

// ///////////////////////////////////////////////////////////////////////
// ///////////////////////////////////////////////////////////////////////
// ///////////////////////////////////////////////////////////////////////

// type protoReader struct {
// 	r     io.Reader
// 	b     []byte
// 	start int
// 	end   int
// }

// func newProtoReader(r io.Reader) *protoReader {
// 	return &protoReader{
// 		r:     r,
// 		start: 0,
// 		end:   0,
// 	}
// }

// func (p *protoReader) Unmarshal(m proto.Message) error {
// 	messageSize := 0
// 	for {
// 		varIntVal, varintSize := proto.DecodeVarint(p.b[p.start:p.end])
// 		log.Println("varIntVal", varIntVal, "varintSize", varintSize, "bytes", p.b[p.start:p.end])
// 		if varintSize > 0 {
// 			p.start += varintSize
// 			messageSize = int(varIntVal)
// 			break
// 		}
// 		err := p.fill(10) // Max size of a varint
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	log.Println("declared message size:", messageSize)

// 	for p.end-p.start < messageSize {
// 		err := p.fill(messageSize)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	err := proto.Unmarshal(p.b[p.start:p.start+messageSize], m)
// 	log.Println("actual message size:", proto.Size(m))

// 	p.start += messageSize
// 	if p.start == p.end {
// 		p.start = 0
// 		p.end = 0
// 	}
// 	return err
// }

// // Will read so that (p.end - p.start) <= target.  However it tries to read
// // so that (p.end - p.start) == target.  It will only not do so if a given read
// // does not return enough bytes.
// func (p *protoReader) fill(target int) error {
// 	// log.Println("Filling with target", target)
// 	if p.end >= p.start+target {
// 		return nil
// 	}
// 	if p.start+target > len(p.b) {
// 		if len(p.b) < target {
// 			old := p.b[p.start:p.end]
// 			p.b = make([]byte, target)
// 			copy(p.b, old)
// 		} else {
// 			copy(p.b, p.b[p.start:p.end])
// 		}
// 		p.end -= p.start
// 		p.start = 0
// 	}

// 	n, err := p.r.Read(p.b[p.end : p.start+target])
// 	if err != nil {
// 		return err
// 	}
// 	p.end += n
// 	return nil
// }

///////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////

func startAgones() *sync.WaitGroup {
	waitForEmpty := &sync.WaitGroup{}

	{
		disabled, ok := os.LookupEnv("DISABLE_AGONES")
		if ok {
			if disabled == "true" {
				log.Println("Agones disabled")
				return waitForEmpty
			}
			log.Fatal("Unknown DISABLE_AGONES value:", disabled)
		}
	}

	log.Println("Starting Agones")
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

	var shutdown sync.Once
	a.WatchGameServer(func(gs *agonesSdk.GameServer) {
		if gs.GetStatus().GetState() == "Allocated" {
			shutdown.Do(func() {
				log.Println("Detected the server is allocated.")
				time.Sleep(time.Second * 15)
				log.Println("Waiting for players to disconnect then shutting down.")
				waitForEmpty.Wait()
				a.Shutdown()
			})
		}
	})

	return waitForEmpty
}

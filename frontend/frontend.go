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
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/pkg/pb"
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/app/static/"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			log.Println("Serving index page.")
			http.ServeFile(w, r, "/app/static/index.html")
		} else {
			log.Println("404 on", r.URL.Path)
			http.NotFound(w, r)
		}
	})

	http.Handle("/matchmake/", websocket.Handler(matchmake))

	log.Println("Starting server")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func matchmake(ws *websocket.Conn) {
	ctx := ws.Request().Context()

	conn, err := grpc.Dial("om-frontend.open-match.svc.cluster.local:50504", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fe := pb.NewFrontendClient(conn)

	var ticketId string
	{
		req := &pb.CreateTicketRequest{
			Ticket: &pb.Ticket{},
		}

		resp, err := fe.CreateTicket(ctx, req)
		if err != nil {
			panic(err)
		}
		ticketId = resp.Ticket.Id
	}

	defer func() {
		_, err := fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: ticketId})
		if err != nil {
			log.Println("Error deleting ticket", ticketId, ":", err)
		}
	}()

	{
		req := &pb.GetAssignmentsRequest{
			TicketId: ticketId,
		}

		stream, err := fe.GetAssignments(ctx, req)
		if err != nil {
			logAndSendError(ws, err)
			return
		}
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				logAndSendError(ws, err)
				return
			}

			err = sendAssignment(ws, resp.Assignment)
			if err != nil {
				log.Println("Error sending updated assignment:", err)
				return
			}
		}
	}
}

func logAndSendError(ws *websocket.Conn, err error) {
	log.Println("Error streaming assignment:", err)
	err = sendAssignment(ws, &pb.Assignment{Error: status.Convert(err).Proto()})
	if err != nil {
		log.Println("Error sending error:", err)
	}
}

func sendAssignment(ws *websocket.Conn, a *pb.Assignment) error {
	b, err := proto.Marshal(a)
	if err != nil {
		return err
	}

	n, err := ws.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return errors.New("Wrong number of bytes sent to websocket.")
	}
	return nil
}

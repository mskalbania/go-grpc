package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log"
	"os"
	"proto/gen/todo"
	"time"
)

func main() {
	addr := os.Getenv("GRPC_SERVER_ADDR")
	if addr == "" {
		log.Fatalf("GRPC_SERVER_ADDR not set")
	}

	//1. Create a connection to the server
	connOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), //this will use TCP without TLS
	}
	conn, err := grpc.NewClient(addr, connOpts...)
	if err != nil {
		log.Fatalf("error creating client: %v", err)
	}

	defer conn.Close()

	//2. Crate client for specific service
	todoClient := todo.NewTodoServiceClient(conn)

	//3a. Call server - unary API example
	for i := 1; i < 10; i++ {
		rs, err := todoClient.AddTask(context.Background(), &todo.AddTaskRequest{
			Description: fmt.Sprintf("do smth %v", i),
			DueDate:     timestamppb.New(time.Now().Add(time.Hour * 24)),
		})
		if err != nil {
			log.Fatalf("error adding task: %v", err)
		}
		log.Printf("added task with id: %d", rs.Id)
	}

	//3. Call sever - server streaming API example
	streamingClient, err := todoClient.ListTasks(context.Background(), &todo.ListTasksRequest{})
	if err != nil {
		log.Fatalf("error getting tasks: %v", err)
	}
	for { //inf loop until send trailer received from server
		rs, err := streamingClient.Recv()
		if err == io.EOF {
			log.Printf("server done")
			break
		}
		if err != nil {
			log.Fatalf("error getting task: %v", err)
		}
		log.Printf("got task: %v", rs)
	}
}

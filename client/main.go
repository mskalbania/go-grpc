package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log"
	"os"
	"proto/gen/todo"
	"sync"
	"time"
)

func main() {
	addr := os.Getenv("GRPC_SERVER_ADDR")
	if addr == "" {
		log.Fatalf("GRPC_SERVER_ADDR not set")
	}

	//1. Create a connection to the server
	//enabling TLS - adding CA so client can verify server certificate
	//also server override is required here since certificate is for: Subject Alternative Names: *.test.example.com
	cr, err := credentials.NewClientTLSFromFile("ca_cert.pem", "x.test.example.com")
	if err != nil {
		log.Fatalf("error creating credentials: %v", err)
	}

	connOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(cr),                //this will use TLS now
		grpc.WithUnaryInterceptor(clientSideInterceptor), //register client side interceptor
		//grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)), //use gzip compression for all calls
	}
	conn, err := grpc.NewClient(addr, connOpts...)
	if err != nil {
		log.Fatalf("error creating client: %v", err)
	}

	defer conn.Close()

	//2. Crate client for specific service
	todoClient := todo.NewTodoServiceClient(conn)

	//3a. Call server - unary API example
	//google advises to use timeouts for every grpc call
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	//headers in grpc, manually setting in RPC call
	ctx = metadata.AppendToOutgoingContext(ctx, "x-api-key", "XD")
	defer cancel()
	for i := 1; i < 4; i++ {
		rs, err := todoClient.AddTask(ctx, &todo.AddTaskRequest{
			Description: fmt.Sprintf("do smth %v", i),
			DueDate:     timestamppb.New(time.Now().Add(-time.Hour * 24)),
		}, grpc.UseCompressor(gzip.Name)) //use gzip compression for this call
		if err != nil {
			if s, ok := status.FromError(err); ok { //and here we can convert error back to status
				switch s.Code() {
				case codes.InvalidArgument:
					log.Fatalf("invalid argument: %v", s.Message())
				}
			}

			log.Fatalf("error adding task: %v", err)
		}
		log.Printf("added task with id: %d", rs.Id)
	}

	//3b. Call sever - server streaming API example
	var ids []uint64
	mask, err := fieldmaskpb.New(&todo.Task{}, "id") //this check if paths exists in Task proto
	if err != nil {
		log.Fatalf("error creating mask: %v", err)
	}
	serverStreaming, err := todoClient.ListTasks(context.Background(), &todo.ListTasksRequest{Mask: mask})
	if err != nil {
		log.Fatalf("error getting tasks: %v", err)
	}
	for { //inf loop until send trailer received from server
		rs, err := serverStreaming.Recv()
		if err == io.EOF {
			log.Printf("server done")
			break
		}
		if err != nil {
			log.Fatalf("error getting task: %v", err)
		}
		ids = append(ids, rs.Task.Id)
		log.Printf("got task with id: %d, description: %s, dueDate: %s, overdue: %v", rs.Task.Id, rs.Task.Description, rs.Task.DueDate.AsTime().Format(time.StampMilli), rs.Overdue)
	}

	//3c. Call server - client streaming API example
	clientStreaming, err := todoClient.UpdateTask(context.Background())
	if err != nil {
		log.Fatalf("error updating tasks: %v", err)
	}
	for _, id := range ids[1:] {
		err := clientStreaming.Send(&todo.UpdateTaskRequest{
			Id:          id,
			Description: "updated!!",
			Done:        true,
			DueDate:     timestamppb.New(time.Now()),
		})
		if err != nil {
			log.Fatalf("error updating task: %v", err)
		}
	}
	if _, err := clientStreaming.CloseAndRecv(); err != nil { //tells the server that streaming is done and awaits rs
		log.Fatalf("error closing send: %v", err)
	}

	//3d. Call server - bi-directional streaming API example
	biDirStreaming, err := todoClient.DeleteTask(context.Background())
	if err != nil {
		log.Fatalf("error deleting tasks: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { //feedback loop part, listens for server confirmations
		for {
			_, err := biDirStreaming.Recv()
			if err == io.EOF { //server done
				wg.Done()
				break
			}
			if err != nil {
				log.Fatalf("error deleting task %v", err)
			}
			log.Printf("deleted task confirmation")
		}
	}()
	for _, id := range ids {
		if err := biDirStreaming.Send(&todo.DeleteTaskRequest{Id: id}); err != nil {
			log.Fatalf("error deleting task: %v", err)
		}
	}
	if err := biDirStreaming.CloseSend(); err != nil { //send half close
		log.Fatalf("error closing send: %v", err)
	}
	wg.Wait()
}

func clientSideInterceptor(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "tk-tk")
	return invoker(ctx, method, req, reply, cc, opts...)
}

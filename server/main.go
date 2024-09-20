package main

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"proto/gen/todo"
	"server/api"
	"syscall"
)

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		log.Fatalf("LISTEN_ADDR not set")
	}
	//1. Create a listener on the specified address
	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("error listenting: %v", err)
	}
	defer listen.Close()

	//2. Create a new grpc server
	var opts []grpc.ServerOption
	//some no-op interceptor, could be access token validation here
	opts = append(opts, grpc.UnaryInterceptor(someInterceptor))
	server := grpc.NewServer(opts...)
	defer server.Stop()

	//3. server.RegisterService() register implemented services here
	todoAPI := api.NewTodoAPI()
	todo.RegisterTodoServer(server, todoAPI)

	//4. Start the server (different goroutine to register shutdown hook below)
	go func() {
		if err := server.Serve(listen); err != nil {
			log.Fatalf("error serving: %v", err)
		}
	}()

	//5. Graceful shutdown
	shutDown := make(chan os.Signal, 1)
	signal.Notify(shutDown, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-shutDown:
		log.Println("shutting down server")
		server.GracefulStop()
	}
}

func someInterceptor(ctx context.Context, rq any, i *grpc.UnaryServerInfo, h grpc.UnaryHandler) (resp any, err error) {
	return h(ctx, rq)
}

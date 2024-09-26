package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"log"
	"net"
	"os"
	"os/signal"
	"proto/gen/todo"
	"server/api"
	"server/db"
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

	//enabling TLS
	//left - public certificate presented during handshake, right - private key associated with cert public key
	cr, err := credentials.NewServerTLSFromFile("server_cert.pem", "server_key.pem")
	if err != nil {
		log.Fatalf("failed to create credentials: %v", err)
	}
	opts = append(opts, grpc.Creds(cr))
	server := grpc.NewServer(opts...)
	defer server.Stop()

	//3. server.RegisterService() register implemented services here
	todoAPI := api.NewTodoAPI(db.NewTodoDB())
	todo.RegisterTodoServiceServer(server, todoAPI)

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

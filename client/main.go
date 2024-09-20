package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
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
}

package main

import (
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"log"
	"net"
	"os"
	"os/signal"
	"proto/gen/todo"
	"server/api"
	"server/db"
	"strings"
	"syscall"
	"time"
)

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		log.Fatalf("LISTEN_ADDR not set")
	}

	grpcServer, listen := configureGrpcServer(listenAddr)

	go func() {
		if err := grpcServer.Serve(listen); err != nil {
			log.Fatalf("error serving: %v", err)
		}
	}()

	//5. Graceful shutdown
	shutDown := make(chan os.Signal, 1)
	signal.Notify(shutDown, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-shutDown:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		done := make(chan interface{})

		go func() {
			grpcServer.GracefulStop()
			log.Println("grpc server stopped")
			done <- true
		}()
		select { //either stopped or timout
		case <-ctx.Done():
			log.Println("timeout waiting for servers to stop")
		case <-done:
		}
	}
}

func someInterceptor(ctx context.Context, rq any, i *grpc.UnaryServerInfo, h grpc.UnaryHandler) (resp any, err error) {
	return h(ctx, rq)
}

func configureGrpcServer(listenAddr string) (*grpc.Server, net.Listener) {
	//1. Create a listener on the specified address
	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("error listenting: %v", err)
	}

	//2. Create a new grpc server
	var opts []grpc.ServerOption
	//some no-op interceptor, could be access token validation here
	//opts = append(opts, grpc.UnaryInterceptor(someInterceptor)) //can't set multiple interceptors

	//auth interceptor using grpc-middleware package
	opts = append(opts, grpc.ChainUnaryInterceptor(
		logging.UnaryServerInterceptor(l(), logging.WithLogOnEvents(logging.FinishCall)),
		auth.UnaryServerInterceptor(authInterceptor),
	))

	//enabling TLS
	//left - public certificate presented during handshake, right - private key associated with cert public key
	cr, err := credentials.NewServerTLSFromFile("server_cert.pem", "server_key.pem")
	if err != nil {
		log.Fatalf("failed to create credentials: %v", err)
	}
	opts = append(opts, grpc.Creds(cr))

	server := grpc.NewServer(opts...)

	//3. server.RegisterService() register implemented services here
	todoAPI := api.NewTodoAPI(db.NewTodoDB())
	todo.RegisterTodoServiceServer(server, todoAPI)
	return server, listen
}

func authInterceptor(ctx context.Context) (context.Context, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	token := md.Get("authorization")[0]
	////auth logic here
	ctx = context.WithValue(ctx, "token", token)
	return ctx, nil
}

func l() logging.LoggerFunc {
	return func(ctx context.Context, level logging.Level, msg string, fields ...any) {
		f := make(map[string]string, len(fields)/2)
		i := logging.Fields(fields).Iterator()
		for i.Next() {
			k, v := i.At()
			f[k] = v.(string)
		}
		log.Printf("%s/%s | %s | %s | Message: %s", f["grpc.service"], f["grpc.method"], f["grpc.code"], f["grpc.time_ms"], strings.SplitAfter(f["grpc.error"], "desc = ")[1])
	}
}

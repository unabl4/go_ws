package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
)

func StartMyMicroservice(ctx context.Context, listenAddr string, aclData string) error {
	lis, err := net.Listen("tcp", listenAddr) // tcp socket
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
	)

	// grpc server start logic
	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			panic("grpc server start failed")
		}
	}()

	// grpc server stop logic
	go func() {
		// ?
		grpcServer.Stop()
	}()

	return nil
}

// ---

func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Println("TEST", info)
	return handler(ctx, req)
}

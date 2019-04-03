package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"context"
	"encoding/json"
	_ "fmt"
	"google.golang.org/grpc"
	"log"
	"net"
)

type BizSrv struct {
}

// ---

func (s BizSrv) Add(ctx context.Context, _ *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s BizSrv) Check(ctx context.Context, _ *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s BizSrv) Test(ctx context.Context, _ *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

// ---
// streaming services

type AdmSrv struct {
}

func (s AdmSrv) Logging(_ *Nothing, srv Admin_LoggingServer) error {
	return nil // ?
}

func (s AdmSrv) Statistics(interval *StatInterval, srv Admin_StatisticsServer) error {
	return nil
}

type ACL map[string][]string

// ---
// union type
type Srv struct {
	ctx context.Context // cancellation
	acl ACL

	BizSrv
	AdmSrv
}

// ===

func StartMyMicroservice(ctx context.Context, listenAddr string, aclData string) error {
	var acl ACL
	if err := json.Unmarshal([]byte(aclData), &acl); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", listenAddr) // tcp socket
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
	)

	srv := &Srv{ctx: ctx, acl: acl}

	// register the services
	RegisterBizServer(grpcServer, srv)
	RegisterAdminServer(grpcServer, srv) // streaming?

	// ---

	// grpc server start logic (non-blocking)
	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			log.Fatal("grpc server start failed", err)
		}
	}()

	// grpc server stop logic
	go func() {
		<-ctx.Done() // await until everything is done and the stop signal is acquired
		grpcServer.Stop()
	}()

	return nil
}

// ---

func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

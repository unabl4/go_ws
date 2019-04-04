package main

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

import (
	"context"
	"encoding/json"
	_ "fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

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
	ctx context.Context // cancellation, streaming control

	// channel-specific logic
	logChan        chan *Event      // new event
	logSubChan     chan chan *Event // new subscriber
	logSubscribers []chan *Event    // list of subscribers
}

func (s *AdmSrv) Logging(_ *Nothing, srv Admin_LoggingServer) error {
	// fmt.Println("ADMSRV CALLED")

	ch := make(chan *Event, 0)
	s.logSubChan <- ch

	for {
		select {
		case event := <-ch:
			srv.Send(event)
		case <-s.ctx.Done():
			return nil
		}
	}

	return nil
}

func (s AdmSrv) Statistics(interval *StatInterval, srv Admin_StatisticsServer) error {
	return nil
}

type ACL map[string][]string // 'consumer' -> list of endpoints

// ---

// union type
type Srv struct {
	acl ACL

	BizSrv
	AdmSrv
}

// ===

func StartMyMicroservice(ctx context.Context, listenAddr string, aclData string) error {
	// parse ACL
	var acl ACL
	if err := json.Unmarshal([]byte(aclData), &acl); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", listenAddr) // tcp socket
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := &Srv{AdmSrv: AdmSrv{ctx: ctx}, acl: acl}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(srv.unaryInterceptor),
		grpc.StreamInterceptor(srv.streamInterceptor),
	)

	// register the services
	RegisterBizServer(grpcServer, srv)
	RegisterAdminServer(grpcServer, srv) // streaming?

	// ---
	// prepare the channels, attach the handler
	srv.logChan = make(chan *Event, 0)         // broadcast events
	srv.logSubChan = make(chan chan *Event, 0) // broadcast new subscribers

	go func() {
		for {
			select {
			case event := <-srv.logChan: // new event (broadcast)
				// fmt.Println("EVENT!", event)
				// fmt.Println("SUBS:", srv.logSubscribers)

				// deliver the 'event' to all the subscribers
				for _, subChan := range srv.logSubscribers {
					subChan <- event // notify the subscriber
				}
			case newSub := <-srv.logSubChan:
				// add new 'subscriber' to the list of subscribers
				// fmt.Println("NEW SUB:", newSub)
				srv.logSubscribers = append(srv.logSubscribers, newSub)
			case <-ctx.Done():
				return
			}
		}
	}()

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

func (s *Srv) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// fmt.Println("UNARY INTERCEPTOR", req)

	// get meta
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "can't get metadata")
	}

	consumer, ok := meta["consumer"]
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "can't get metadata")
	}

	// log the unary event
	s.logChan <- &Event{
		Consumer: consumer[0],
		Method:   info.FullMethod,
		Host:     "127.0.0.1:8083",
	}

	return handler(ctx, req)
}

func (s *Srv) streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// fmt.Println("STREAM INTERCEPTOR", srv)

	// get meta
	meta, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "can't get metadata")
	}

	consumer, ok := meta["consumer"]
	if !ok {
		return grpc.Errorf(codes.Unauthenticated, "can't get metadata")
	}

	// log the stream event
	s.logChan <- &Event{
		Consumer: consumer[0],
		Method:   info.FullMethod,
		Host:     "127.0.0.1:8083",
	}

	return handler(srv, ss)
}

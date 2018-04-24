package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	pb "istio-stackdriver/helloworld"
)

var (
	port = flag.String("port", "50051", "grpc port")
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

func (s *server) visitSvcC() (string, error) {
	conn, err := net.Dial("tcp", "svc-c:23333")
	if err != nil {
		return "", fmt.Errorf("unable to build connection with svcC: %v", err)
	}
	fmt.Fprintf(conn, "Greating from svcB!\n")
	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("run into error when reading messages from svcC: %v", err)
	}
	return message, nil
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		log.Printf("request metadata is: %v", md)
	} else {
		log.Printf("cannot get metdadata: %v", ok)
	}
	m := "Hello " + in.Name + "\n"
	if cm, err := s.visitSvcC(); err != nil {
		return nil, fmt.Errorf("failed to get response from svcC: %v", err)
	} else {
		m += "Message from svcC: " + cm + "\n"
	}
	return &pb.HelloReply{Message: m}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

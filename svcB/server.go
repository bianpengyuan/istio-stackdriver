package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
	pb "istio-stackdriver/helloworld"
)

var (
	port = flag.String("port", "50051", "grpc port")
)

func extractHeaders(md metadata.MD) map[string]string {
	headers := []string{
		"x-request-id",
		"x-b3-traceid",
		"x-b3-spanid",
		"x-b3-parentspanid",
		"x-b3-sampled",
		"x-b3-flags",
		"x-ot-span-context",
	}

	ret := map[string]string{}
	for _, key := range headers {
		val := md.Get(key)
		if len(val) != 0 {
			ret[key] = val[0]
		}
	}
	return ret
}

// server is used to implement helloworld.GreeterServer.
type server struct{}

func (s *server) visitHttpbin(md metadata.MD) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://httpbin.org:80", nil)
	eh := extractHeaders(md)
	for k, v := range eh {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return fmt.Sprintf("Get response status code %v from httpbin", resp.StatusCode), nil
}

func (s *server) visitSvcC() (string, error) {
	conn, err := net.Dial("tcp", "svc-c:23333")
	defer conn.Close()
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
	m := "SvcB\n"
	r := rand.Intn(10)
	if r < 5 {
		if cm, err := s.visitSvcC(); err != nil {
			return nil, fmt.Errorf("failed to get response from svcC: %v", err)
		} else {
			m += cm
		}
	} else {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, fmt.Errorf("failed to get grpc context")
		}
		if gm, err := s.visitHttpbin(md); err != nil {
			return nil, fmt.Errorf("failed to visit httpbin.com")
		} else {
			m += gm + "\n"
		}
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

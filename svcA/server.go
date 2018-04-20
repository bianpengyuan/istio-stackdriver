package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "istio-stackdriver/helloworld"
	"log"
	"net/http"
)

var (
	port = flag.String("port", "8080", "http port")
)

func svcBGreeting() (string, error) {
	conn, err := grpc.Dial("svc-b:50051", grpc.WithInsecure())
	if err != nil {
		return "", fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := "svcA"
	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("could not greet: %v", err)
	}
	return r.Message, nil
}

func EchoHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte("Hello this is svcA\n"))
	if m, err := svcBGreeting(); err == nil {
		writer.Write([]byte("Greeting from svcB: " + m + "\n"))
	} else {
		log.Printf("%v", err)
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/", EchoHandler)
	http.ListenAndServe(":"+*port, nil)
}

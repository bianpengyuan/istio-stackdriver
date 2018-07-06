package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	pb "istio-stackdriver/helloworld"
	"log"
	"math/rand"
	http "net/http"
)

var (
	port = flag.String("port", "8080", "http port")
)

func extractHeaders(r *http.Request) map[string]string {
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
		val := r.Header.Get(key)
		if val != "" {
			ret[key] = val
		}
	}
	return ret
}

func svcBGreeting(req *http.Request) (string, error) {
	conn, err := grpc.Dial("svc-b:50051", grpc.WithInsecure())
	if err != nil {
		return "", fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)
	eh := extractHeaders(req)
	log.Printf("tracing header: %v", eh)

	// Contact the server and print out its response.
	name := "svcA"

	ctx := context.Background()
	for key, val := range eh {
		ctx = metadata.AppendToOutgoingContext(ctx, key, val)
	}

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("could not greet: %v", err)
	}

	// execWorkflow(req)
	return r.Message, nil
}

func EchoHandler(writer http.ResponseWriter, request *http.Request) {
	r := rand.Intn(10)
	if r < 2 {
		http.Error(writer, "error", http.StatusForbidden)
		return
	} else if r >= 2 && r < 4 {
		http.Error(writer, "error", http.StatusNotFound)
		return
	}
	writer.Write([]byte("svcA\n"))
	if m, err := svcBGreeting(request); err == nil {
		writer.Write([]byte(m + "\n"))
	} else {
		log.Printf("%v", err)
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/", EchoHandler)

	http.ListenAndServe(":"+*port, nil)
}

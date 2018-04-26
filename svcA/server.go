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
	"time"

	openzipkin "github.com/openzipkin/zipkin-go"
	zrh "github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/exporter/zipkin"
	b3 "go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace"
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

func execWorkflow(req *http.Request) {
	f := &b3.HTTPFormat{}
	p, ok := f.SpanContextFromRequest(req)
	if !ok {
		fmt.Println("Cannot parse http request with b3 format")
		return
	}

	ctx, span := trace.StartSpanWithRemoteParent(context.Background(), "svc-a-foo", p)
	_, span1 := trace.StartSpan(ctx, "svc-a-bar")
	time.Sleep(50 * time.Millisecond)
	span1.End()
	_, span2 := trace.StartSpan(ctx, "svc-a-baz")
	time.Sleep(50 * time.Millisecond)
	span2.End()
	span.End()

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

	execWorkflow(req)
	return r.Message, nil
}

func EchoHandler(writer http.ResponseWriter, request *http.Request) {
	r := rand.Intn(10)
	if r < 3 {
		http.Error(writer, "error", http.StatusForbidden)
		return
	} else if r >= 3 && r < 7 {
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

	// Initialize open census zipkin exporter
	endpoint, err := openzipkin.NewEndpoint("svc-a-workload", "")
	if err != nil {
		log.Println(err)
	}

	// The Zipkin reporter takes collected spans from the app and reports them to the backend
	// http://localhost:9411/api/v2/spans is the default for the Zipkin Span v2
	reporter := zrh.NewReporter("http://zipkin.istio-system:9411/api/v2/spans")
	defer reporter.Close()

	// The OpenCensus exporter wraps the Zipkin reporter
	exporter := zipkin.NewExporter(reporter, endpoint)
	trace.RegisterExporter(exporter)

	// For example purposes, sample every trace.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	http.ListenAndServe(":"+*port, nil)
}

package main

import (
	"bufio"
	// "encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	// "time"

	"golang.org/x/net/context"
	// "google.golang.org/api/option"
	"google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
	pb "istio-stackdriver/helloworld"
	//	"go.opencensus.io/exporter/stackdriver"
	//	"go.opencensus.io/trace"
)

var (
	port = flag.String("port", "50051", "grpc port")
)

//func buildTraceID(s string) ([16]byte, error) {
//	tid := [16]byte{}
//
//	l := hex.DecodedLen(len(s))
//	decoded, err := hex.DecodeString(s)
//
//	if err != nil {
//		return tid, err
//	}
//	for i := 0; i < 16; i++ {
//		if i < 16-l {
//			tid[i] = 0
//		} else {
//			tid[i] = decoded[l+i-16]
//		}
//	}
//	return tid, err
//}

// func buildSpanID(s string) ([8]byte, error) {
//	sid := [8]byte{}
//
//	l := hex.DecodedLen(len(s))
//	decoded, err := hex.DecodeString(s)
//
//	if err != nil {
//		return sid, err
//	}
//	for i := 0; i < 8; i++ {
//		if i < 8-l {
//			sid[i] = 0
//		} else {
//			sid[i] = decoded[l+i-8]
//		}
//	}
//	return sid, nil
// }

//func execWorkflow(md metadata.MD) {
//	vals := md.Get("x-b3-traceid")
//	if len(vals) == 0 {
//		fmt.Println("Cannot find trace id")
//	}
//	tid, err := buildTraceID(vals[0])
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	vals = md.Get("x-b3-spanid")
//	if len(vals) == 0 {
//		fmt.Println("Cannot find span id")
//	}
//	sid, err := buildSpanID(vals[0])
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	p := trace.SpanContext{
//		TraceID:      tid,
//		SpanID:       sid,
//		TraceOptions: 0x1,
//	}
//
//	ctx, span := trace.StartSpanWithRemoteParent(context.Background(), "svc-b-foo", p)
//	_, span1 := trace.StartSpan(ctx, "svc-b-bar")
//	time.Sleep(50 * time.Millisecond)
//	span1.End()
//	_, span2 := trace.StartSpan(ctx, "svc-b-baz")
//	time.Sleep(50 * time.Millisecond)
//	span2.End()
//	span.End()
//}

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

func (s *server) visitGoogle(md metadata.MD) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://www.google.com:443", nil)
	eh := extractHeaders(md)
	for k, v := range eh {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Get response status code %v from google", resp.StatusCode), nil
}

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
	m := "SvcB\n"
	if cm, err := s.visitSvcC(); err != nil {
		return nil, fmt.Errorf("failed to get response from svcC: %v", err)
	} else {
		m += cm + "\n"
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get grpc context")
	}
	if gm, err := s.visitGoogle(md); err != nil {
		return nil, fmt.Errorf("failed to visit google.com")
	} else {
		m += gm + "\n"
	}
	// execWorkflow(md)
	return &pb.HelloReply{Message: m}, nil
}

func main() {
	//	exporter, err := stackdriver.NewExporter(stackdriver.Options{
	//		BundleDelayThreshold: time.Second / 10,
	//		BundleCountThreshold: 5,
	//	})
	//	if err != nil {
	//		log.Println(err)
	//	}
	//	trace.RegisterExporter(exporter)
	//	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

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

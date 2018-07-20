package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	trafficRate = flag.Int("traffic-rate", 100, "traffic rate")
)

func getIngressIP() string {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	rate := time.Second
	throttle := time.Tick(rate)
	for {
		<-throttle
		selectors := labels.Set{"istio": "ingressgateway"}.AsSelectorPreValidated()
		listOptions := metav1.ListOptions{
			LabelSelector: selectors.String(),
		}
		svcs, err := clientset.CoreV1().Services("istio-system").List(listOptions)
		if err != nil {
			panic(err.Error())
		}
		if svcs == nil || len(svcs.Items) == 0 {
			log.Printf("Cannot find ingress resource.")
			continue
		}
		ingress := svcs.Items[0].Status.LoadBalancer.Ingress
		if len(ingress) == 0 {
			log.Printf("Cannot find ingress field.")
			continue
		}
		return ingress[0].IP
	}
}

func main() {
	rate := time.Second / time.Duration(*trafficRate)
	throttle := time.Tick(rate)
	refreshIPCounter := 10000
	ip := "127.0.0.1"
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {
		if refreshIPCounter == 10000 {
			refreshIPCounter = 0
			ip = getIngressIP()
			log.Printf("the ip address of gateway is %v", ip)
		}
		refreshIPCounter++
		<-throttle
		go func() {
			resp, err := http.Get("http://" + ip + "/")
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			if err != nil {
				log.Printf("Error http request %v", err)
			}

			nginxResp, err := http.Get("http://" + ip + "/nginx")
			if nginxResp != nil && nginxResp.Body != nil {
				nginxResp.Body.Close()
			}
			if err != nil {
				log.Printf("Error http request %v", err)
			}

			nginxHTTPSResp, err := http.Get("https://" + ip + ":443/index.html")
			if nginxHTTPSResp != nil && nginxHTTPSResp.Body != nil {
				nginxHTTPSResp.Body.Close()
			}
			if err != nil {
				log.Printf("Error http request %v", err)
			}
		}()
	}
}

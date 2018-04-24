package main

import (
	"log"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
		selectors := labels.Set{"istio": "ingress"}.AsSelectorPreValidated()
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
	ip := getIngressIP()
	log.Printf("the ip address of gateway is %v", ip)
	rate := time.Second * 10
	throttle := time.Tick(rate)
	for {
		<-throttle
		go func() {
			resp, err := http.Get("http://" + ip + "/")
			defer resp.Body.Close()
			if err != nil {
				log.Printf("Error http request %v", err)
			}
		}()
	}
}

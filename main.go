package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"

	"github.com/miekg/dns"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var records = map[string]string{}

func parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			log.Printf("Query for %s\n", q.Name)
			ip := records[q.Name]
			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	w.WriteMsg(m)
}

func watchIngresses(clientset *kubernetes.Clientset, records map[string]string) {
	wi, err := clientset.NetworkingV1().Ingresses("").Watch(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for event := range wi.ResultChan() {
		switch event.Type {
		case watch.Added:
			for _, rule := range event.Object.(*networkingv1.Ingress).Spec.Rules {
				fmt.Println("Added: ", rule.Host)
				records[rule.Host+"."] = "127.0.0.1"
			}
		case watch.Deleted:
			for _, rule := range event.Object.(*networkingv1.Ingress).Spec.Rules {
				fmt.Println("Removed: ", rule.Host)
				delete(records, rule.Host+".")
			}
		}

	}
}

func main() {

	kubeconfig := flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	go watchIngresses(clientset, records)

	dns.HandleFunc(".", handleDnsRequest)

	// start server
	port := 5353
	server := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}
	log.Printf("Starting at %d\n", port)
	err = server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}

}

package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

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

	wi, err := clientset.NetworkingV1().Ingresses("").Watch(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for event := range wi.ResultChan() {
		switch event.Type {
		case watch.Added:
			for _, rule := range event.Object.(*networkingv1.Ingress).Spec.Rules {
				fmt.Println("Added: ", rule.Host)
			}
		case watch.Deleted:
			for _, rule := range event.Object.(*networkingv1.Ingress).Spec.Rules {
				fmt.Println("Removed: ", rule.Host)
			}
		}

	}

}

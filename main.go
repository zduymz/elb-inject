package main

import (
	"flag"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/zduymz/elb-inject/pkg/apis/elb-inject"
	"github.com/zduymz/elb-inject/pkg/controller"
	//clientset "k8s.io/sample-controller/pkg/generated/clientset/versioned"
	//informers "k8s.io/sample-controller/pkg/generated/informers/externalversions"
	"github.com/zduymz/elb-inject/pkg/signals"
)

var config elb_inject.Config

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	// (client kubernetes.Interface, defaultResync time.Duration)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)

	controller, err := controller.NewController(kubeInformerFactory.Core().V1().Pods(), kubeClient, &config)
	if err != nil {
		klog.Fatalf("Error building kubernetes controller: %s", err.Error())
	}

	kubeInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&config.KubeConfig, "kubeconfig", "./minikube.config", "Hello")
	flag.StringVar(&config.Master, "master", "", "Hello")
	flag.StringVar(&config.AWSRegion, "aws.region", "us-west-2", "Hello")
	flag.StringVar(&config.AWSVPCId, "aws.vpcid", "vpc-9931a0fc", "Hello")
	flag.IntVar(&config.APIRetries, "aws.retries", 3, "Hello")
	flag.StringVar(&config.AWSAssumeRole, "aws.sts", "", "Hello")
	flag.StringVar(&config.AWSCredsFile, "aws.creds", "/Users/dmai/.aws/credentials", "Hello")
}

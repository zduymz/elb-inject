package controller

import (
	"fmt"
	"github.com/zduymz/elb-inject/pkg/apis/elb-inject"
	"github.com/zduymz/elb-inject/pkg/provider"
	"k8s.io/client-go/kubernetes"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "what the fucking name"

const (
	// add to pod when injection is done
	annotationStatus = "devops.apixio.com/elb-inject-status"

	// inject a pod ip to this target group
	annotationInject = "devops.apixio.com/elb-inject-target-group-name"
)

var (
	// i don't want to mess up with namespace system and public
	kubeSystemNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
)

type Controller struct {
	podLister     corelisters.PodLister
	kubeclientset kubernetes.Interface
	hasSynced     cache.InformerSynced
	workqueue     workqueue.RateLimitingInterface
	provider      *provider.AWSProvider
}

func NewController(podInformer coreinformers.PodInformer, kubeclientset kubernetes.Interface, config *elb_inject.Config) (*Controller, error) {
	klog.Info("Setting up AWS")

	p, err := provider.NewAWSProvider(provider.AWSConfig{
		Region:     config.AWSRegion,
		AssumeRole: config.AWSAssumeRole,
		AWSCredsFile:   config.AWSCredsFile,
		APIRetries: 3,
		DryRun:     false,
	})
	if err != nil {
		klog.Errorf("Error: %s", err.Error())
		return nil, err
	}

	controller := &Controller{
		podLister:     podInformer.Lister(),
		hasSynced:     podInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ELB Register"),
		provider:      p,
		kubeclientset: kubeclientset,
	}

	klog.Info("Setting up event handlers")

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleAddObject,
		UpdateFunc: func(old, new interface{}) {
			newOne := new.(*corev1.Pod)
			oldOne := old.(*corev1.Pod)
			if newOne.ResourceVersion == oldOne.ResourceVersion {
				return
			}
			controller.handleAddObject(new)
		},
		DeleteFunc: controller.handleDeleteObject,
	})

	return controller, nil
}

// Run will set event handler for pod, syncing informer caches and starting workers.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Starting controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.hasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// obj in form of namespace/name
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			klog.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		klog.V(4).Infof("syncHanlder key %s", key)
		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, Requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the pod with this namespace/name
	po, err := c.podLister.Pods(namespace).Get(name)
	if err != nil {
		// If no longer exist, in which case we stop processing
		if errors.IsNotFound(err) {
			klog.Warningf("object '%s' in work queue no longer exists", key)
			return nil
		}

		return err
	}

	// make sure pod is running
	if !c.isPodRunning(po) {
		klog.Errorf("Pod %s : %s ", po.GetName(), po.Status.Phase)
		return fmt.Errorf("Pod not running")
	}

	//TODO: need to check is ready to serve traffic
	// Not sure this is good solution but it worked for now
	if !c.isPodReady(po) {
		klog.Infof("Pod [%s] is not ready, can not inject", po.GetName())
		return fmt.Errorf("Pod is not ready")
	}

	// double check
	if should := c.shouldInject(po); !should {
		return nil
	}

	targetGroup := po.Annotations[annotationInject]
	klog.Infof("Starting register [%s] to Target: [%s]", po.Status.PodIP, targetGroup)
	if err := c.provider.RegisterIPToTargetGroup(&targetGroup, &po.Status.PodIP); err != nil {
		return err
	}

	klog.V(4).Info("Adding `injected` annotation to pod template")
	if err := c.updatePodAnnotation(po); err != nil {
		return err
	}

	return nil
}

func (c *Controller) updatePodAnnotation(po *corev1.Pod) error {
	poCopy := po.DeepCopy()
	poCopy.Annotations[annotationStatus] = "true"
	_, err := c.kubeclientset.CoreV1().Pods(poCopy.GetNamespace()).Update(poCopy)
	return err
}

func (c *Controller) enqueuePod(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// TODO: this step is quite redundant, what tombstone is?
func (c *Controller) handleAddObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("error decoding object, invalid type")
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			klog.Errorf("error decoding object tombstone, invalid type")
			return
		}
		klog.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	klog.V(4).Infof("Processing object: %s", object.GetName())

	// TODO: should we check object KIND before converting?
	po, err := c.podLister.Pods(object.GetNamespace()).Get(object.GetName())
	if err != nil {
		klog.Infof("Ignoring orphaned object [%s] ", object.GetName())
		return
	}

	if should := c.shouldInject(po); should {
		klog.Infof("Injecting object: %s", po.GetName())
		c.enqueuePod(po)
		return
	}
}

func (c *Controller) handleDeleteObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("error decoding object, invalid type")
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			klog.Errorf("error decoding object tombstone, invalid type")
			return
		}
		klog.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	klog.V(4).Infof("Processing object: %s", object.GetName())

	po := obj.(*corev1.Pod)
	// pod should have been injected
	if po.Annotations[annotationStatus] == "" {
		klog.Errorf("Not found %s in %s", annotationStatus, po.GetName())
		return
	}

	// pod should contain annotationInject
	targetGroup := po.Annotations[annotationInject]
	if targetGroup == "" {
		klog.Errorf("Not found %s in %s", annotationInject, po.GetName())
		return
	}

	if err := c.provider.DeregisterIPToTargetGroup(&targetGroup, &po.Status.PodIP); err != nil {
		klog.Errorf("Can not deregister pod %s - %s", po.GetName(), po.Status.PodIP)
	}
	klog.Infof("Deregister [%s] from [%s] sucessfully", po.Status.PodIP, targetGroup)
}

func (c *Controller) shouldInject(pod *corev1.Pod) bool {

	// Don't inject in the Kubernetes system namespaces
	for _, ns := range kubeSystemNamespaces {
		if pod.GetNamespace() == ns {
			return false
		}
	}

	// If we already injected then don't do inject again
	if pod.Annotations[annotationStatus] != "" {
		return false
	}

	// Only work with annotation defined
	if pod.Annotations[annotationInject] == "" {
		return false
	}

	return true
}

func (c *Controller) isPodReady(pod *corev1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if ! containerStatus.Ready {
			return false
		}
	}
	return true
}

func (c *Controller) isPodRunning(pod *corev1.Pod) bool {
	if podStatus := pod.Status.Phase; podStatus != corev1.PodRunning {
		return false
	}
	return true
}

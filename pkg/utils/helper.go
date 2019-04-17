package utils

import "k8s.io/klog"

// All dummy function should stay here

func DumpObject(obj interface{})  {
	klog.Info(obj)
}

func Log(s string) {
	klog.Infof("%s \n", s)
}

// check the list contain string
func IsInSlice(x string, list []*interface{}) bool {
	for _,v := range list {
		if *v == x {
			return true
		}
	}
	return false
}
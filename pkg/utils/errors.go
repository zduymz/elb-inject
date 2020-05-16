package utils

import "fmt"

type PodNotRun struct {}

func (p PodNotRun) Error() string {
	return fmt.Sprintf("Pod not running")
}

type AWSDeregisterError struct {
	TargetGroupARN string
	Err error
}

func (a AWSDeregisterError) Error() string {
	return a.Err.Error()
}
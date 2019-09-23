package elb_inject

import "time"

type Config struct {
	Master         string
	RequestTimeout time.Duration
	AWSAssumeRole  string
	AWSRegion      string
	AWSVPCId       string
	APIRetries     int
	SlackWebHook   string

	// Just use for testing purpse
	AWSCredsFile   string
	KubeConfig     string
}


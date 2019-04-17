package provider

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/linki/instrumented_http"
	"k8s.io/klog"
	"strings"
)

type TargetGroupAPI interface {
	DescribeTargetGroups(input *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error)
	RegisterTargets(input *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error)
	DeregisterTargets(input *elbv2.DeregisterTargetsInput) (*elbv2.DeregisterTargetsOutput, error)
}
type AWSProvider struct {
	client TargetGroupAPI
	dryRun bool
}

// AWSConfig contains configuration to create a new AWS provider.
type AWSConfig struct {
	Region     string
	VpcId      string
	AWSCreds   *credentials.Credentials
	AssumeRole string
	APIRetries int
	DryRun     bool
}

// NewAWSProvider initializes a new AWS Route53 based Provider.
func NewAWSProvider(awsConfig AWSConfig) (*AWSProvider, error) {
	config := aws.NewConfig().WithMaxRetries(awsConfig.APIRetries).WithRegion(awsConfig.Region)

	// Only use for testing
	if awsConfig.AWSCreds != nil {
		klog.Warning("Not use aws credentials when running on production")
		config.WithCredentials(awsConfig.AWSCreds)
	}

	config.WithHTTPClient(
		instrumented_http.NewClient(config.HTTPClient, &instrumented_http.Callbacks{
			PathProcessor: func(path string) string {
				parts := strings.Split(path, "/")
				return parts[len(parts)-1]
			},
		}),
	)

	awsSession, err := session.NewSessionWithOptions(session.Options{
		Config:            *config,
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		return nil, err
	}

	if awsConfig.AssumeRole != "" {
		klog.Infof("Assuming role: %s", awsConfig.AssumeRole)
		awsSession.Config.WithCredentials(stscreds.NewCredentials(awsSession, awsConfig.AssumeRole))
	}

	provider := &AWSProvider{
		client: elbv2.New(awsSession),
		dryRun: awsConfig.DryRun,
	}

	return provider, nil
}

// Return targetGroup in map[Name: ARN]
// I only care the targetGroup with TargetType is IP
func (p *AWSProvider) getTargetGroups() (map[string]*string, error) {
	targetGroups := make(map[string]*string)
	describeTargetGroupsInput := &elbv2.DescribeTargetGroupsInput{
		PageSize: aws.Int64(400),
	}

	for {
		describeTargetGroupsOutput, err := p.client.DescribeTargetGroups(describeTargetGroupsInput)
		if err != nil {
			klog.Errorf("Can not describe TargetGroup: %s", err.Error())
			return nil, err
		}

		describeTargetGroupsInput.Marker = describeTargetGroupsOutput.NextMarker

		for _, targetGroup := range describeTargetGroupsOutput.TargetGroups {
			// only support target type IP
			if *targetGroup.TargetType == elbv2.TargetTypeEnumIp {
				targetGroups[*targetGroup.TargetGroupName] = targetGroup.TargetGroupArn
			}
		}

		if describeTargetGroupsOutput.NextMarker == nil {
			break
		}
	}
	klog.V(4).Infof("TargetGroups available: %v", targetGroups)
	return targetGroups, nil
}

func (p *AWSProvider) RegisterIPToTargetGroup(targetGroupName *string, IPAddress *string) error {
	klog.V(4).Info("Getting list of current TargetGroups")
	targetGroups, err := p.getTargetGroups()
	if err != nil {
		return err
	}

	if exist := targetGroups[*targetGroupName]; exist == nil {
		klog.Errorf("TargetGroupName: %s is not found", *targetGroupName)
		// TODO: should put back to the queue or throw away?
		return fmt.Errorf("TargetGroup not found")
	}

	target := &elbv2.TargetDescription{
		Id: IPAddress,
	}

	params := &elbv2.RegisterTargetsInput{
		TargetGroupArn: targetGroups[*targetGroupName],
		Targets:        []*elbv2.TargetDescription{target},
	}

	if _, err := p.client.RegisterTargets(params); err != nil {
		klog.Errorf("Can not register %s to targetGroup %s. Reason: %s", *IPAddress, *targetGroupName, err.Error())
		return err
	}

	return nil
}

func (p *AWSProvider) DeregisterIPToTargetGroup(targetGroupName *string, IPAddress *string) error {
	targetGroups, err := p.getTargetGroups()
	if err != nil {
		return err
	}

	if exist := targetGroups[*targetGroupName]; exist == nil {
		klog.Errorf("TargetGroupName: %s is not found", *targetGroupName)
		// TODO: should put back to the queue or throw away?
		// When this case happen?
		return fmt.Errorf("TargetGroup not found")
	}

	target := &elbv2.TargetDescription{
		Id: IPAddress,
	}

	params := &elbv2.DeregisterTargetsInput{
		TargetGroupArn: targetGroups[*targetGroupName],
		Targets:        []*elbv2.TargetDescription{target},
	}

	if _, err := p.client.DeregisterTargets(params); err != nil {
		klog.Errorf("Can not register %s to targetGroup %s. Reason: %s", *IPAddress, *targetGroupName, err.Error())
		return err
	}

	return nil
}

package provider

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/zduymz/elb-inject/pkg/utils"

	"math/rand"
	"testing"
	"time"
)

//TODO: write more test cases
type mockSession struct{}

func (s *mockSession) DescribeTargetGroups(_ *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error) {
	// should return different result depend on input

	output := &elbv2.DescribeTargetGroupsOutput{
		TargetGroups: []*elbv2.TargetGroup{
			{
				HealthCheckEnabled:         aws.Bool(true),
				HealthCheckIntervalSeconds: aws.Int64(30),
				HealthCheckPath:            aws.String("/"),
				HealthCheckPort:            aws.String("traffic-port"),
				HealthCheckProtocol:        aws.String("HTTPS"),
				HealthCheckTimeoutSeconds:  aws.Int64(5),
				HealthyThresholdCount:      aws.Int64(5),
				LoadBalancerArns:           []*string{aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/app/devops-graylog/8505c4e92fffa17e")},
				Matcher: &elbv2.Matcher{
					HttpCode: aws.String("200"),
				},
				Port:                    aws.Int64(443),
				Protocol:                aws.String("HTTPS"),
				TargetGroupArn:          aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/devops-graylog/efb439fb328cf038"),
				TargetGroupName:         aws.String("dmai-test-0"),
				TargetType:              aws.String("ip"),
				UnhealthyThresholdCount: aws.Int64(3),
				VpcId:                   aws.String("vpc-c78fffa0"),
			},
			//---------
			{
				HealthCheckEnabled:         aws.Bool(true),
				HealthCheckIntervalSeconds: aws.Int64(30),
				HealthCheckPath:            aws.String("/"),
				HealthCheckPort:            aws.String("traffic-port"),
				HealthCheckProtocol:        aws.String("HTTPS"),
				HealthCheckTimeoutSeconds:  aws.Int64(5),
				HealthyThresholdCount:      aws.Int64(5),
				LoadBalancerArns:           []*string{aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/net/devops-graylog-tcp/acaec130065e8111")},
				Matcher: &elbv2.Matcher{
					HttpCode: aws.String("200"),
				},
				Port:                    aws.Int64(443),
				Protocol:                aws.String("HTTPS"),
				TargetGroupArn:          aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/devops-graylog-fluent/167d61f098a726ce"),
				TargetGroupName:         aws.String("dmai-test-1"),
				TargetType:              aws.String("ip"),
				UnhealthyThresholdCount: aws.Int64(2),
				VpcId:                   aws.String("vpc-c78fffa0"),
			},
			//---------
			{
				HealthCheckEnabled:         aws.Bool(true),
				HealthCheckIntervalSeconds: aws.Int64(30),
				HealthCheckPath:            aws.String("/"),
				HealthCheckPort:            aws.String("traffic-port"),
				HealthCheckProtocol:        aws.String("HTTPS"),
				HealthCheckTimeoutSeconds:  aws.Int64(5),
				HealthyThresholdCount:      aws.Int64(5),
				Matcher: &elbv2.Matcher{
					HttpCode: aws.String("200"),
				},
				Port:                    aws.Int64(80),
				Protocol:                aws.String("HTTP"),
				TargetGroupArn:          aws.String("please-return-error"),
				TargetGroupName:         aws.String("dmai-test-2"),
				TargetType:              aws.String("ip"),
				UnhealthyThresholdCount: aws.Int64(3),
				VpcId:                   aws.String("vpc-9931a0fc"),
			},
			//---------
			{
				HealthCheckEnabled:         aws.Bool(true),
				HealthCheckIntervalSeconds: aws.Int64(30),
				HealthCheckPath:            aws.String("/"),
				HealthCheckPort:            aws.String("traffic-port"),
				HealthCheckProtocol:        aws.String("HTTPS"),
				HealthCheckTimeoutSeconds:  aws.Int64(5),
				HealthyThresholdCount:      aws.Int64(5),
				Matcher: &elbv2.Matcher{
					HttpCode: aws.String("200"),
				},
				Port:                    aws.Int64(80),
				Protocol:                aws.String("HTTP"),
				TargetGroupArn:          aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/dmai-test-3/ac0e6820c8cbd875"),
				TargetGroupName:         aws.String("dmai-test-3"),
				TargetType:              aws.String("ip"),
				UnhealthyThresholdCount: aws.Int64(3),
				VpcId:                   aws.String("vpc-9931a0fc"),
			},
		},
	}
	return output, nil
}
func (s *mockSession) RegisterTargets(input *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error) {
	randomErrors := []error{
		fmt.Errorf(elbv2.ErrCodeTargetGroupNotFoundException),
		fmt.Errorf(elbv2.ErrCodeTooManyTargetsException),
		fmt.Errorf(elbv2.ErrCodeInvalidTargetException),
		fmt.Errorf(elbv2.ErrCodeTooManyRegistrationsForTargetIdException),
	}
	rand.Seed(time.Now().Unix())
	if *input.TargetGroupArn == "please-return-error" {
		n := rand.Int() % len(randomErrors)
		return nil, randomErrors[n]
	}

	return nil, nil
}
func (s *mockSession) DeregisterTargets(input *elbv2.DeregisterTargetsInput) (*elbv2.DeregisterTargetsOutput, error) {
	randomErrors := []error{
		fmt.Errorf(elbv2.ErrCodeTargetGroupNotFoundException),
		fmt.Errorf(elbv2.ErrCodeInvalidTargetException),
		utils.AWSDeregisterError{
			TargetGroupARN: "fake-target-group",
			Err:            fmt.Errorf(elbv2.ErrCodeInvalidTargetException),
		},
	}
	rand.Seed(time.Now().Unix())
	if *input.TargetGroupArn == "please-return-error" {
		n := rand.Int() % len(randomErrors)
		return nil, randomErrors[n]
	}

	return nil, nil
}

func NewMockAWSProvider() *AWSProvider {
	provider := &AWSProvider{
		client: &mockSession{},
		dryRun: false,
		cachePool: cache.New(1*time.Minute, 1*time.Minute),
	}

	return provider
}

func TestGetTargetGroups(t *testing.T) {
	provider := NewMockAWSProvider()
	targetGroups, _ := provider.getTargetGroups()
	expectedTargetGroups := map[string]*string{
		"dmai-test-0": aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/devops-graylog/efb439fb328cf038"),
		"dmai-test-1":  aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/devops-graylog-fluent/167d61f098a726ce"),
		"dmai-test-2": aws.String("please-return-error"),
		"dmai-test-3": aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/dmai-test-3/ac0e6820c8cbd875"),
	}

	assert.Equal(t, targetGroups, expectedTargetGroups)

}

func TestRegister(t *testing.T) {
	provider := NewMockAWSProvider()
	err := provider.RegisterIPToTargetGroup(aws.String("dmai-test-2"), aws.String("1.1.1.1"))
	assert.NotEqual(t, err, nil)

	err = provider.RegisterIPToTargetGroup(aws.String("dmai-test-0"), aws.String("1.1.1.1"))
	assert.Equal(t, err, nil)
}

func TestDeregister(t *testing.T) {
	provider := NewMockAWSProvider()
	err := provider.DeregisterIPFromTargetGroup(aws.String("dmai-test-1"), aws.String("1.1.1.1"))
	assert.Equal(t, err, nil)

	err = provider.DeregisterIPFromTargetGroup(aws.String("dmai-test-2"), aws.String("1.1.1.2"))
	assert.NotEqual(t, err, nil)
}
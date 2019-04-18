# elb-inject

This project is one of many projects when i'm in a journey building k8s architecture for migrating to k8s from aws

## Purpose

An idea is simple. I have a lot of AWS ELBv2 with many target groups. They are running with many EC2 instances on AWS. To migrate from AWS to K8s seemlessly, i have to run AWS and K8S in parallel.  
  
I would like a new created POD automatically self-register to the specific target group when it had a tag.
Simply put annotation into manifest and magic happen:
`devops.apixio.com/elb-inject-target-group-name: targetGroup`

## Testing on local
Edit `run` in `Makefile` to use correct configuration
```bash
# start minikube
minikube start

# build binary, because i used MacBook
make macos

# run elb-inject
make run
```

## Run on Kubernetes
Worker Node must have at least the following IAM permissions
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "elasticloadbalancing:RegisterTargets",
                "elasticloadbalancing:DescribeTargetGroups",
                "elasticloadbalancing:DeregisterTargets"
            ],
            "Resource": "*"
        }
    ]
}
```
### Without RBAC
```bash
kubectl create -f manifest.yml
```

### With RBAC
```bash
kubectl create -f manifest-rbac.yml
```
# elb-inject

This project is one of many projects when i'm in a journey building k8s architecture for migrating to k8s from aws

**Note:** This project is not completed yet. Still on working on it


## Purpose

An idea is simple. I have a lot of AWS ELBv2 with many target groups. They are running with many EC2 instances on AWS. To migrate from AWS to K8s seemlessly, i have to run AWS and K8S in parallel.  
  
I would like a new created POD automatically self-register to the specific target group when it had a tag.
Simply put annotation into manifest and magic happen:
`devops.apixio.com/elb-inject-target-group-name: targetGroup`

## Build
```bash
make macos
```

## Testing
```bash
make run
```
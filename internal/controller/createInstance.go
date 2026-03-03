package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	computev1 "github.com/mrudhulraj/kube-controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// context and ec2Instance meta
func createEC2Instance(ec2Instance *computev1.Ec2Instance) (createdInstanceInfo *computev1.CreatedInstanceInfo, err error) {
	l := log.Log.WithName("createEc2Instance")

	// l.Info("Loading AWS config")
	// cfg, err := awsconfig.LoadDefaultConfig(ctx)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to load AWS config: %w", err)
	// }

	ec2Client := awsClient(ec2Instance.Spec.Region)

	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AmiID),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		KeyName:      aws.String(ec2Instance.Spec.KeyPair),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	l.Info(" ==== CALLING AWS RunINstances API ====")
	result, err := ec2Client.RunInstances(context.TODO(), runInput)
	if err != nil {
		l.Error(err, "Failed to create EC2 instance")
		return nil, fmt.Errorf("failed to create EC2 instance : %w", err)
	}

	if len(result.Instances) == 0 {
		l.Error(nil, "No instances returned in RunInstancesOutput")
		return nil, nil
	}

	inst := result.Instances[0]
	l.Info("=== EC2 INSTANCE CREATED SUCCESSFULLY === ", "instanceID", *inst.InstanceId)

	runWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 3 * time.Minute

	err = runWaiter.Wait(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}, maxWaitTime)

	if err != nil {
		l.Error(err, "Failed to describe EC2 instance")
		return nil, fmt.Errorf("failed to describe EC2 instance:%w", err)
	}

	l.Info("=== CALLING AWS DescribeInstances API TO GET INSTANCE DETAILS ===")
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{*inst.InstanceId},
	}

	describeResult, err := ec2Client.DescribeInstances(context.TODO(), describeInput)
	if err != nil {
		l.Error(err, "Failed to describe instance")
		return nil, fmt.Errorf("failed to describe EC2 instance: %w", err)
	}

	fmt.Println("Describe Result ", " Public IP ", *&describeResult.Reservations[0].Instances[0].PublicDnsName, "state", describeResult.Reservations[0].Instances[0].State.Name)

	fmt.Printf("Private IP of the instance: %v", derefString(inst.PrivateIpAddress))
	fmt.Printf("State of the instance: %v", describeResult.Reservations[0].Instances[0].State.Name)
	fmt.Printf("Private DNS of the instance: %v", derefString(inst.PrivateDnsName))
	fmt.Printf("Instance ID of the instance: %v", derefString(inst.InstanceId))
	fmt.Println("Instance Type of the instance: ", inst.InstanceType)
	fmt.Printf("Image ID of the instance: %v", derefString(inst.ImageId))
	fmt.Printf("Key Name of the instance: %v", derefString(inst.KeyName))

	instance := describeResult.Reservations[0].Instances[0]
	createdInstanceInfo = &computev1.CreatedInstanceInfo{
		InstanceID: *instance.InstanceId,
		PublicIP:   derefString(instance.PublicIpAddress),
		State:      string(instance.State.Name),
	}
	return createdInstanceInfo, nil
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return "<nil>"
}

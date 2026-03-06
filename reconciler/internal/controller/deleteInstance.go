package controller

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	computev1 "github.com/mrudhulraj/kube-controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func deleteEC2Instance(ctx context.Context, ec2Instance *computev1.Ec2Instance) (bool, error) {
	l := log.FromContext(ctx)

	l.Info("Deleting  EC2 Instance", "instanceID", ec2Instance.Status.InstanceID)

	ec2Client := awsClient(ec2Instance.Spec.Region)

	_, err := ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	})

	if err != nil {
		l.Error(err, "Failed to terminate ec2 Instance")
		return false, err
	}

	l.Info("Instance termination initiated")

	waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)

	maxwaitTime := 5 * time.Minute
	waitParams := &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2Instance.Status.InstanceID},
	}

	l.Info("Waiting for instance to be terminated")

	err = waiter.Wait(ctx, waitParams, maxwaitTime)
	if err != nil {
		l.Error(err, "Faield while waiting for instance termination")
		return false, err
	}

	return true, nil
}

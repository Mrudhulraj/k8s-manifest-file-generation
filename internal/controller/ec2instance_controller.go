/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	computev1 "github.com/mrudhulraj/kube-controller/api/v1"
)

// Ec2InstanceReconciler reconciles a Ec2Instance object
type Ec2InstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=compute.controller.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.controller.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.controller.com,resources=ec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ec2Instance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *Ec2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// TODO(user): your logic here
	// Holds the current status regardless of the presence/absence/modified state of the ec2 instance
	// r is reconciler webserver
	ec2instance := &computev1.Ec2Instance{}
	if err := r.Get(ctx, req.NamespacedName, ec2instance); err != nil {
		return ctrl.Result{}, err
	}

	// Check If delete is already called
	if !ec2instance.DeletionTimestamp.IsZero() {
		l.Info("Has Deletion Timestamp, Instance is being deleted")

		_, err := deleteEC2Instance(ctx, ec2instance)

		if err != nil {
			l.Error(err, "Failed to delete the EC2 Instance")
			return ctrl.Result{Requeue: true}, err // Requeue request if it failed
		}
		return ctrl.Result{}, nil
	}

	// Check if instance is there
	if ec2instance.Status.InstanceID != "" {
		l.Info("Requesed object already exists in K8s.")
		return ctrl.Result{}, nil
	}
	l.Info("Creating new instance")
	// Add Finalizer
	ec2instance.Finalizers = append(ec2instance.Finalizers, "ec2instance.compute.cloud.com")
	if err := r.Update(ctx, ec2instance); err != nil {
		l.Error(err, "Failed to add finalizer")
		return ctrl.Result{Requeue: true}, err
	}

	// Create a new Instance
	l.Info("=== CONTINUING WITH EC2 INSTANCE CREATION === ")

	createdInstanceInfo, err := createEC2Instance(ec2instance)
	if err != nil {
		l.Error(err, "Failed to create EC2 instance")
		return ctrl.Result{}, err
	}

	l.Info("=== ABOUT TO UPDATE STATUS - This will trigger reconcile loop again ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State)

	ec2instance.Status.InstanceID = createdInstanceInfo.InstanceID
	ec2instance.Status.State = createdInstanceInfo.State
	ec2instance.Status.PublicIP = createdInstanceInfo.PublicIP
	ec2instance.Status.PrivateIP = createdInstanceInfo.PrivateIP
	ec2instance.Status.PublicDNS = createdInstanceInfo.PublicDNS
	ec2instance.Status.PrivateDNS = createdInstanceInfo.PrivateDNS

	err = r.Status().Update(ctx, ec2instance)
	if err != nil {
		l.Error(err, "Failed to update status")
		// Kubernetes will retry with backoff
		return ctrl.Result{}, err
	}

	l.Info("=== STATUS UPDATED - Reconcile loop will be triggered again ===")
	return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Ec2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.Ec2Instance{}).
		Named("ec2instance").
		Complete(r)
}

/*
Copyright 2021. @mcbenjemaa

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

package controllers

import (
	"context"
	"fmt"

	"reflect"

	daemonv1alpha1 "github.com/mcbenjemaa/daemonjob-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = daemonv1alpha1.GroupVersion.String()
	kind        = reflect.TypeOf(daemonv1alpha1.DaemonJob{}).Name()
	annotation  = "daemon.justk8s.com/node-name"
)

// DaemonJobReconciler reconciles a DaemonJob object
type DaemonJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=daemon.justk8s.com,resources=daemonjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=daemon.justk8s.com,resources=daemonjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=daemon.justk8s.com,resources=daemonjobs/finalizers,verbs=update

//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch,resources=jobs/finalizers,verbs=update

//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=nodes/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DaemonJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := clog.FromContext(ctx)

	log.Info("reconciling DaemonJob")

	// Retreive DaemonJob object
	var daemonJob daemonv1alpha1.DaemonJob
	if err := r.Get(ctx, req.NamespacedName, &daemonJob); err != nil {
		log.Error(err, "unable to fetch DaemonJob")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Retreive childJobs
	var childJobs batchv1.JobList
	if err := r.List(ctx, &childJobs,
		client.InNamespace(req.Namespace),
		client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	// Get List of all nodes
	nodeList, err := r.listNodes(ctx)
	if err != nil {
		log.Error(err, "unable to list nodes")
		return ctrl.Result{}, err
	}

	// update status
	status, err := r.daemonJobStatus(&daemonJob, &childJobs, nodeList)
	if !reflect.DeepEqual(status, daemonJob.Status) {
		log.Info("Updating daemon job status")
		daemonJob.Status = *status.DeepCopy()
		err = r.Status().Update(ctx, &daemonJob)
		if err != nil {
			return ctrl.Result{}, err
		}
		//if err := r.Status().Update(ctx, &daemonJob); err != nil {
		//	log.Error(err, "unable to update DaemonJob status")
		//	return ctrl.Result{}, err
		//}
	}

	// desiredJobs
	desiredJobs, err := r.desiredJobsForDaemonJob(req.Namespace, &daemonJob, nodeList)
	if err != nil {
		log.Error(err, "unable to construct required Jobs")
		return ctrl.Result{}, err
	}

	// create desired Jobs
	err = r.createDesiredJobsForDaemonJob(ctx, &daemonJob, desiredJobs)
	if err != nil {
		log.Error(err, "error creating desired jobs")
		return ctrl.Result{}, err
	}
	// your logic here

	return ctrl.Result{}, nil
}

func (r *DaemonJobReconciler) listNodes(ctx context.Context) (*v1.NodeList, error) {

	var nodeList v1.NodeList

	if err := r.List(ctx, &nodeList, client.MatchingLabelsSelector{Selector: labels.Everything()}); err != nil {
		return nil, err
	}

	return &nodeList, nil
}

func (dr *DaemonJobReconciler) daemonJobStatus(dj *daemonv1alpha1.DaemonJob, childJobs *batchv1.JobList, nodeList *v1.NodeList) (*daemonv1alpha1.DaemonJobStatus, error) {
	var desiredNumberScheduled, numberAvailable, completedJobs, failedJobs int32

	//desiredNumberScheduled = len(nodeList.Items)
	for _, node := range nodeList.Items {
		shouldRun, _ := dr.nodeShouldRunDaemonJob(&node, dj)

		if shouldRun {
			desiredNumberScheduled++
		}
	}

	for _, job := range childJobs.Items {
		finishedType := jobStatus(job)
		switch finishedType {
		case "": // ongoing
			numberAvailable++
		case batchv1.JobFailed:
			failedJobs++
			numberAvailable++
		case batchv1.JobComplete:
			completedJobs++
			numberAvailable++
		}
	}

	status := &daemonv1alpha1.DaemonJobStatus{
		DesiredNumberScheduled: desiredNumberScheduled,
		NumberAvailable:        &numberAvailable,
		FailedJobs:             &failedJobs,
		CompletedJobs:          &completedJobs,
	}

	return status, nil
}

// nodeShouldRunDaemonJob checks a set of preconditions against a (node,daemonjob) and returns a
// summary. Returned booleans are:
// TODO
// * shouldRun:
//     Returns true when a daemonset should run on the node if a daemonset pod is not already
//     running on that node.
// * shouldContinueRunning:
//     Returns true when a daemonset should continue running on a node if a daemonset pod is already
//     running on that node.
func (dsc *DaemonJobReconciler) nodeShouldRunDaemonJob(node *v1.Node, dj *daemonv1alpha1.DaemonJob) (bool, bool) {
	//pod := daemonctrl.NewPod(dj, node.Name)

	// If the daemon job specifies a node name, check that it matches with node.Name.
	if !(dj.Spec.JobTemplate.Spec.Template.Spec.NodeName == "" || dj.Spec.JobTemplate.Spec.Template.Spec.NodeName == node.Name) {
		return false, false
	}

	return true, true
}

// Create required Jobs that should be running
func (r *DaemonJobReconciler) desiredJobsForDaemonJob(namespace string, daemonJob *daemonv1alpha1.DaemonJob, nodeList *v1.NodeList) ([]*batchv1.Job, error) {
	jobTemplate := &daemonJob.Spec.JobTemplate

	var jobs []*batchv1.Job

	for _, node := range nodeList.Items {
		jobName := fmt.Sprintf("%s-%s", daemonJob.Name, node.Name)

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        jobName,
				Namespace:   namespace,
			},
			Spec: *jobTemplate.Spec.DeepCopy(),
		}
		for k, v := range jobTemplate.Annotations {
			job.Annotations[k] = v
		}
		// Add nodeName annotation
		job.Annotations[annotation] = node.Name

		for k, v := range jobTemplate.Labels {
			job.Labels[k] = v
		}
		if len(job.Spec.Template.Spec.NodeSelector) == 0 {
			job.Spec.Template.Spec.NodeSelector = make(map[string]string)
		}
		job.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = node.Name

		jobs = append(jobs, job)
	}
	return jobs, nil
}

// Create Desired Jobs
func (r *DaemonJobReconciler) createDesiredJobsForDaemonJob(ctx context.Context, daemonJob *daemonv1alpha1.DaemonJob, desiredJobs []*batchv1.Job) error {
	log := clog.FromContext(ctx)

	for _, job := range desiredJobs {
		if err := ctrl.SetControllerReference(daemonJob, job, r.Scheme); err != nil {
			return err
		}

		if err := r.Create(ctx, job); err != nil && errors.IsAlreadyExists(err) {
			log.Info("desired Job already exists")
		} else if err != nil {
			return err
		}
	}

	return nil
}

func (r *DaemonJobReconciler) mapToDaemonJob(_ client.Object) []ctrl.Request {
	ctx := context.Background()
	log := clog.FromContext(ctx)
	daemonJobList := &daemonv1alpha1.DaemonJobList{}
	err := r.Client.List(ctx, daemonJobList)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "error getting list of DaemonJob")
		}
		return nil
	}

	var results []ctrl.Request
	for _, daemonJob := range daemonJobList.Items {
		results = append(results, ctrl.Request{
			NamespacedName: client.ObjectKey{
				Namespace: daemonJob.GetNamespace(),
				Name:      daemonJob.GetName(),
			},
		})
	}
	return results
}

func (r *DaemonJobReconciler) indexJobOwnerField(rawObj client.Object) []string {
	job := rawObj.(*batchv1.Job)
	owner := metav1.GetControllerOf(job)
	if owner == nil {
		return nil
	}
	if owner.APIVersion != apiGVStr || owner.Kind != kind {
		return nil
	}
	return []string{owner.Name}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonJobReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &batchv1.Job{}, jobOwnerKey, r.indexJobOwnerField); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&daemonv1alpha1.DaemonJob{}).
		Watches(&source.Kind{Type: &v1.Node{}}, handler.EnqueueRequestsFromMapFunc(r.mapToDaemonJob)).
		Owns(&batchv1.Job{}).
		Complete(r)
}

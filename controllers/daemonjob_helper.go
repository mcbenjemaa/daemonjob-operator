package controllers

import (
	daemonv1alpha1 "github.com/mcbenjemaa/daemonjob-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
)

// NewPod creates a new pod
func NewPod(dj *daemonv1alpha1.DaemonJob, nodeName string) *v1.Pod {
	newPod := &v1.Pod{Spec: dj.Spec.JobTemplate.Spec.Template.Spec, ObjectMeta: dj.Spec.JobTemplate.ObjectMeta}
	newPod.Namespace = dj.Namespace
	newPod.Spec.NodeName = nodeName

	// Added default tolerations for DaemonSet pods.
	//daemonutil.AddOrUpdateDaemonPodTolerations(&newPod.Spec)

	return newPod
}

func jobStatus(job batchv1.Job) batchv1.JobConditionType {

	isJobFinished := func(job *batchv1.Job) (bool, batchv1.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == v1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}

	_, finishedType := isJobFinished(&job)

	return finishedType
}

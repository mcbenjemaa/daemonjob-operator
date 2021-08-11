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

package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobTemplateSpec defines the Template of DaemonJobSpec
type JobTemplateSpec struct {
	// Standard object's metadata of the jobs created from this template.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the job.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec batchv1.JobSpec `json:"spec,omitempty"`
}

// DaemonJobSpec defines the desired state of DaemonJob
type DaemonJobSpec struct {

	// TODO: Add selector mutually exclusive with ignoreSelector // e.g MatchLabels

	// TODO: Add ignoreSelector mutually exclusive with selector // e.g MatchLabels

	// Specifies the job that will be created when executing a DaemonJob.
	JobTemplate JobTemplateSpec `json:"jobTemplate"`
}

// DaemonJobStatus defines the observed state of DaemonJob
type DaemonJobStatus struct {

	// The total number of nodes that should be running the daemon
	// job (including nodes correctly running the daemon job).
	DesiredNumberScheduled int32 `json:"desiredNumberScheduled"`

	// The number of nodes that should be running the
	// daemon job and have one or more of the pod running and
	// available (ready for at least spec.minReadySeconds)
	// +optional
	NumberAvailable *int32 `json:"numberAvailable"`

	// The number of jobs that are completed.
	// +optional
	CompletedJobs *int32 `json:"completedJobs,omitempty"`

	// The number of jobs that are failed
	// +optional
	FailedJobs *int32 `json:"failedJobs,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".status.desiredNumberScheduled",name="DESIRED",type="integer"
//+kubebuilder:printcolumn:JSONPath=".status.numberAvailable",name="AVAILABLE",type="integer"
//+kubebuilder:printcolumn:JSONPath=".status.completedJobs",name="COMPLETED",type="integer"
//+kubebuilder:printcolumn:JSONPath=".status.failedJobs",name="Failed",type="integer"

// DaemonJob is the Schema for the daemonjobs API
type DaemonJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DaemonJobSpec   `json:"spec,omitempty"`
	Status DaemonJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DaemonJobList contains a list of DaemonJob
type DaemonJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DaemonJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DaemonJob{}, &DaemonJobList{})
}

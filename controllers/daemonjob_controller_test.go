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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	daemonv1alpha1 "github.com/mcbenjemaa/daemonjob-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("DaemonJob controller", func() {
	const (
		Namespace     = "default"
		DaemonJobName = "test-daemonjob"
		JobName       = "test-daemonjob-test-0"
		NodeName      = "test-0"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating DaemonJob Status", func() {
		ctx := context.Background()

		daemonJobLookupKey := types.NamespacedName{Name: DaemonJobName, Namespace: Namespace}
		createdDaemonJob := &daemonv1alpha1.DaemonJob{}
		jobLookupKey := types.NamespacedName{Name: JobName, Namespace: Namespace}
		createdJob := &batchv1.Job{}
		gvk := daemonv1alpha1.GroupVersion.WithKind(kind)

		It("should increase DaemonJob Status.NumberAvailable when new Jobs are created", func() {
			By("creating a new DaemonJob")

			daemonJob := &daemonv1alpha1.DaemonJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "daemon.justk8s.com/v1alpha1",
					Kind:       "DaemonJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      DaemonJobName,
					Namespace: Namespace,
				},
				Spec: daemonv1alpha1.DaemonJobSpec{
					JobTemplate: daemonv1alpha1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:    "test",
											Image:   "busybox",
											Command: []string{"date"},
										},
									},
									RestartPolicy: v1.RestartPolicyOnFailure,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, daemonJob)).Should(Succeed())

			By("check DaemonJob has been created")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, daemonJobLookupKey, createdDaemonJob)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdDaemonJob.Spec.JobTemplate).ToNot(BeNil()) // TODO: check with another spec field

			By("checking the DaemonJob has zero child Jobs")
			Consistently(func() (int32, error) {
				err := k8sClient.Get(ctx, daemonJobLookupKey, createdDaemonJob)
				if err != nil {
					return -1, err
				}
				return createdDaemonJob.Status.DesiredNumberScheduled, nil
			}, duration, interval).Should(BeZero())

			By("creating a new Job")
			testJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      JobName,
					Namespace: Namespace,
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:    "test",
									Image:   "busybox",
									Command: []string{"date"},
								},
							},
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": NodeName,
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
			}

			controllerRef := metav1.NewControllerRef(createdDaemonJob, gvk)
			testJob.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, testJob)).Should(Succeed())

			By("checking that Job is created")
			Eventually(func() error {
				err := k8sClient.Get(ctx, jobLookupKey, createdJob)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).ShouldNot(HaveOccurred())
			Expect(createdJob.Spec.Template).ToNot(BeNil())

			By("checking that the DaemonJob has one child Job")
			Eventually(func() (int32, error) {
				err := k8sClient.Get(ctx, daemonJobLookupKey, createdDaemonJob)
				if err != nil {
					return -1, err
				}
				return *createdDaemonJob.Status.NumberAvailable, nil
			}, timeout, interval).Should(BeEquivalentTo(1))

		})

		It("should update DaemonJob Status.NumberAvailable when a Jobs are deleted", func() {

			By("removing Job")
			Expect(k8sClient.Delete(ctx, createdJob)).ToNot(HaveOccurred())

			By("patching Job with nil finalizers")
			patch := []byte(`[ { "op": "remove", "path": "/metadata/finalizers" } ]`)
			Expect(k8sClient.Patch(ctx, createdJob, client.RawPatch(types.JSONPatchType, patch))).NotTo(HaveOccurred())

			By("check the Job does not exists anymore")
			Consistently(func() error {
				err := k8sClient.Get(ctx, jobLookupKey, createdJob)
				if err != nil {
					return err
				}
				return nil
			}, duration, interval).Should(HaveOccurred())

			By("checking the DaemonJob has zero child Jobs")
			Eventually(func() (int, error) {
				emptyDaemonJobSet := &daemonv1alpha1.DaemonJob{}
				err := k8sClient.Get(ctx, daemonJobLookupKey, emptyDaemonJobSet)
				if err != nil {
					return -1, err
				}
				return int(*emptyDaemonJobSet.Status.NumberAvailable), nil
			}, timeout, interval).Should(BeZero())

		})

		It("should create Job when new node is added", func() {
			By("creating a new Node")
			testNode := &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: NodeName,
					Labels: map[string]string{
						"beta.kubernetes.io/os":  "linux",
						"kubernetes.io/hostname": NodeName,
					},
				},
				Spec: v1.NodeSpec{},
			}
			Expect(k8sClient.Create(ctx, testNode)).NotTo(HaveOccurred())

			By("checking that Job has been created")
			nodeCreatedJob := &batchv1.Job{}
			Eventually(func() error {
				err := k8sClient.Get(ctx, jobLookupKey, nodeCreatedJob)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).ShouldNot(HaveOccurred())
			Expect(nodeCreatedJob.Spec.Template).ToNot(BeNil())
			Expect(nodeCreatedJob.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"]).To(Equal(NodeName))
			Expect(nodeCreatedJob.ObjectMeta.Annotations[annotation]).To(Equal(NodeName))
		})

	})
})

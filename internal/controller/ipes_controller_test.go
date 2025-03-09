/*
Copyright 2025.

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
	"fmt"
	"time"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Test Creating ExternalSecret", func() {
	const (
		triggerSecretName  = "test-secret"
		externalSecretName = "test-es"
		podName            = "test-pod"
		containerName      = "test-container"
		containerImage     = "busybox:latest"
		namespaceName      = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	Context("When creating/deleting a pod", func() {
		It("should create External Secret if required", func() {
			ctx := context.Background()
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaceName,
					Name:      podName,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: triggerSecretName},
					},
					Containers: []corev1.Container{
						{Name: containerName, Image: containerImage},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			podLookupKey := types.NamespacedName{Name: podName, Namespace: namespaceName}
			createdPod := &corev1.Pod{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, createdPod)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Expect(createdPod.Spec.ImagePullSecrets[0].Name).To(Equal(triggerSecretName))

			esLookupKey := types.NamespacedName{Name: externalSecretName, Namespace: namespaceName}
			createdEs := &esv1beta1.ExternalSecret{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, esLookupKey, createdEs)).To(Succeed())
			}, timeout, interval).Should(Succeed())
		})
		It("should delete External Secret if no longer required", func() {
			ctx := context.Background()
			podLookupKey := types.NamespacedName{Name: podName, Namespace: namespaceName}
			deletedPod := &corev1.Pod{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, deletedPod)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			err := k8sClient.Delete(ctx, deletedPod)
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, podLookupKey, deletedPod)).ToNot(Succeed())
			}, timeout, interval).Should(Succeed())

			esList := &esv1beta1.ExternalSecretList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, esList)).To(Succeed())
				g.Expect(esList.Items).Should(BeEmpty())
				fmt.Printf("External Secret list: %+v\n", esList)
			}, timeout, interval).Should(Succeed())
		})
	})

})

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
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Test Creating ExternalSecret", func() {
	ctx := context.Background()

	os.Setenv("TRIGGER_SECRET_NAME", "test-secret")
	os.Setenv("DESIRED_EXTERNAL_SECRET", heredoc.Doc(`
		apiVersion: external-secrets.io/v1beta1
		kind: ExternalSecret
		metadata:
  		name: test-es
		spec:
			refreshInterval: 1h
			secretStoreRef:
				kind: ClusterSecretStore
				name: test-sm
			target:
				name: test-secret
				creationPolicy: Merge
			data:
				- secretKey: test-key
					remoteRef:
						key: test
        property: test-property
	`))

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-pod",
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "test-secret"},
			},
			Containers: []corev1.Container{
				{Name: "test-container", Image: "busybox:latest"},
			},
		},
	}

	It("creates required resource", func() {
		err := k8sClient.Create(ctx, pod)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() {
			externalSecret := &esv1beta1.ExternalSecret{}
			err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "test-es"}, externalSecret)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(externalSecret.Name).Should(Equal("test-es"))
			Expect(externalSecret.Namespace).Should(Equal("default"))
		})
	})

	It("deletes previously required resource", func() {
		err := k8sClient.Delete(ctx, pod)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() {
			externalSecretList := &esv1beta1.ExternalSecretList{}
			err = k8sClient.List(ctx, externalSecretList, &client.ListOptions{Namespace: "default"})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(externalSecretList.Items).Should(BeEmpty())
		})
	})
})

/*
Copyright 2024.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/resources"
)

var _ = Describe("Keycloak Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const resourceNs = "rhsso"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNs,
		}
		keycloak := &ssov1alpha1.Keycloak{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Keycloak")
			err := k8sClient.Get(ctx, typeNamespacedName, keycloak)
			if err != nil && errors.IsNotFound(err) {

				keycloak = &ssov1alpha1.Keycloak{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNs,
					},
					Spec: ssov1alpha1.KeycloakSpec{
						Instances: &[]int32{1}[0],
						Admin: ssov1alpha1.AdminUser{
							Username: ssov1alpha1.SecretOption{
								Value: "admin",
							},
							Password: ssov1alpha1.SecretOption{
								Value: "admin",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, keycloak)).To(Succeed())
			}
		})

		AfterEach(func() {
			keycloak = &ssov1alpha1.Keycloak{}
			err := k8sClient.Get(ctx, typeNamespacedName, keycloak)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Keycloak")
			Expect(k8sClient.Delete(ctx, keycloak)).To(Succeed())
		})

		It("should successfully reconcile the Route resource", func() {
			By("Reconciling the keycloak resource")
			controllerReconciler := &KeycloakReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking Statefulset resource has been created")
			statefulSets := &appsv1.StatefulSetList{}
			err = k8sClient.List(ctx, statefulSets, &client.ListOptions{
				LabelSelector: labels.SelectorFromSet(labels.Set{"app": "rhbk"}),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(statefulSets.Items).To(HaveLen(1))
			Expect(statefulSets.Items[0].Name).To(Equal(resourceName))
			Expect(statefulSets.Items[0].Spec.Replicas).To(Equal(keycloak.Spec.Instances))
		})

		It("should successfully reconcile service resource", func() {
			By("Reconciling the keycloak resource")
			controllerReconciler := &KeycloakReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking Service resource has been created")
			svc := &v1.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: resourceNs,
				Name:      resources.GetSvcName(keycloak),
			}, svc)
			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Name).To(Equal(resources.GetSvcName(keycloak)))
		})

		It("should successfully reconcile discovery service resource", func() {
			By("Reconciling the keycloak resource")
			controllerReconciler := &KeycloakReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking Service resource has been created")
			svc := &v1.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: resourceNs,
				Name:      resources.GetDiscoverySvcName(keycloak),
			}, svc)
			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Name).To(Equal(resources.GetDiscoverySvcName(keycloak)))
		})
	})
})

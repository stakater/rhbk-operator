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
	route "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/test/utils"
)

var _ = Describe("Keycloak Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const resourceNs = "rhbk-instance"

		ctx := context.Background()

		var keycloak *ssov1alpha1.Keycloak

		BeforeEach(func() {
			keycloak = &ssov1alpha1.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNs,
				},
			}

			By("creating the custom resource for the Kind Keycloak")
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloak), keycloak)
			if err != nil && errors.IsNotFound(err) {
				utils.GetResourceFromFile("keycloak.yaml", keycloak)
				Expect(k8sClient.Create(ctx, keycloak)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance Keycloak")
			DeleteIfExist(ctx, keycloak)
		})

		It("should sync statefulset", func() {
			key := client.ObjectKeyFromObject(keycloak)
			By("Reconciling the keycloak resource")
			ReconcileKeycloak(ctx, key)

			By("Checking Statefulset resource has been created")
			statefulSet := &appsv1.StatefulSet{}
			err := k8sClient.Get(ctx, key, statefulSet)

			Expect(err).NotTo(HaveOccurred())
			Expect(statefulSet.Name).To(Equal(resourceName))
			Expect(statefulSet.Spec.Replicas).To(Equal(keycloak.Spec.Instances))
			Expect(HasOwnerRef(keycloak, statefulSet)).To(BeTrue())
		})

		It("should setup provider download init-container", func() {
			key := client.ObjectKeyFromObject(keycloak)

			By("Reconciling the keycloak resource")
			ReconcileKeycloak(ctx, key)

			By("Checking Statefulset resource has been created")
			statefulSet := &appsv1.StatefulSet{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloak), statefulSet)

			Expect(err).NotTo(HaveOccurred())
			Expect(statefulSet.Spec.Template.Spec.InitContainers).To(HaveLen(1))
			Expect(statefulSet.Spec.Template.Spec.InitContainers[0].Args).To(Equal([]string{
				"-c",
				"mkdir -p /opt/keycloak/providers; curl -LJ --show-error --cacert conf/truststores/ca-bundle.crt -o /opt/keycloak/providers/keycloak-metrics-spi-6.0.0.jar $(KEYCLOAK_METRICS_SPI_6_0_0_JAR)",
			}))
		})

		It("should successfully reconcile resources", func() {
			key := client.ObjectKeyFromObject(keycloak)

			By("Reconciling the keycloak resource")
			ReconcileKeycloak(ctx, key)

			By("Checking Statefulset resource has been created")
			statefulSet := GetKeycloakStatefulSet(ctx, keycloak)
			Expect(statefulSet.Name).To(Equal(resourceName))
			Expect(statefulSet.Spec.Replicas).To(Equal(keycloak.Spec.Instances))
			Expect(statefulSet.Spec.Template.Spec.InitContainers).To(HaveLen(1))

			By("Checking route resource has been created")
			route := &route.Route{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloak), route)

			Expect(err).NotTo(HaveOccurred())
			Expect(route.Name).To(Equal(resourceName))

			By("Checking svc resource has been created")
			svcName := keycloak.Name + "-svc"
			svc := &v1.Service{}

			_ = k8sClient.Get(ctx, client.ObjectKey{
				Name:      svcName,
				Namespace: keycloak.Namespace,
			}, svc)

			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Name).To(Equal(svcName))

			By("Checking discovery-svc resource has been created")
			svcName = keycloak.Name + "-discovery"
			svc = &v1.Service{}

			_ = k8sClient.Get(ctx, client.ObjectKey{
				Name:      svcName,
				Namespace: keycloak.Namespace,
			}, svc)

			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Name).To(Equal(svcName))
		})

		It("should reconcile changes", func() {
			key := client.ObjectKeyFromObject(keycloak)

			By("Reconciling the keycloak resource")
			ReconcileKeycloak(ctx, key)

			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloak), keycloak)
			Expect(err).NotTo(HaveOccurred())

			keycloak.Spec.Instances = &[]int32{1}[0]
			err = k8sClient.Update(ctx, keycloak)
			Expect(err).NotTo(HaveOccurred())

			ReconcileKeycloak(ctx, key)

			By("Checking Statefulset resource has been created")
			statefulSet := &appsv1.StatefulSet{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(keycloak), statefulSet)

			Expect(err).NotTo(HaveOccurred())
			Expect(statefulSet.Name).To(Equal(resourceName))
			Expect(statefulSet.Spec.Replicas).To(Equal(keycloak.Spec.Instances))
		})
	})
})

func GetKeycloakStatefulSet(ctx context.Context, kc *ssov1alpha1.Keycloak) *appsv1.StatefulSet {
	statefulSet := &appsv1.StatefulSet{}
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kc), statefulSet)
	Expect(err).NotTo(HaveOccurred())
	return statefulSet
}

func ReconcileKeycloak(ctx context.Context, key client.ObjectKey) {
	controllerReconciler := &KeycloakReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: key,
	})

	Expect(err).NotTo(HaveOccurred())
}

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
	"fmt"
	"go/build"
	"path/filepath"
	"runtime"
	"testing"

	monitoring "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	//+kubebuilder:scaffold:imports

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	route "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join(build.Default.GOPATH, "pkg", "mod", "github.com", "openshift", "api@v0.0.0-20250305144515-529099f6d7a6", "route", "v1", "zz_generated.crd-manifests"),
		},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.29.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = ssov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = route.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = monitoring.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	err = k8sClient.Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhbk-instance",
		},
	})
	Expect(err).NotTo(HaveOccurred())

	err = k8sClient.Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhbk-import",
		},
	})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func DeleteIfExist(ctx context.Context, obj client.Object) {
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	if err != nil && errors.IsNotFound(err) {
		return
	}

	obj.SetFinalizers(nil)
	Expect(k8sClient.Update(ctx, obj)).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationBackground))).ToNot(HaveOccurred())
}

func HasOwnerRef(parent client.Object, child client.Object) bool {
	for _, reference := range child.GetOwnerReferences() {
		if reference.UID == parent.GetUID() {
			return true
		}
	}

	return false
}

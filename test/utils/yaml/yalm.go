package yaml

import (
	"fmt"

	"os"

	"github.com/imdario/mergo"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetResourceFromFile(filename string, obj client.Object) {
	override := obj.DeepCopyObject()
	yamlData, err := os.ReadFile(fmt.Sprintf("../../test/crds/%s", filename))
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to read YAML file")

	err = yaml.Unmarshal(yamlData, obj)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to parse YAML")

	err = mergo.Merge(obj, override, mergo.WithOverride)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to merge StatefulSet objects")
}

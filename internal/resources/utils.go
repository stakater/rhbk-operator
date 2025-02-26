package resources

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"

	"github.com/stakater/rhbk-operator/internal/constants"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/kstatus/status"
)

func FormatResource(obj interface{}) string {
	jsonData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v\n", obj)
	}

	return string(jsonData)
}

func GetDefaultLabels() map[string]string {
	return map[string]string{
		"app":                              "rhbk",
		constants.RHBKWatchedResourceLabel: strconv.FormatBool(true),
	}
}

func IsJobCompleted(job *v1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type != v1.JobComplete {
			continue
		}

		return condition.Status == "True"
	}

	return false
}

func AddOrReplaceCondition(c metav1.Condition, conditions []metav1.Condition) []metav1.Condition {
	for i, condition := range conditions {
		if c.Type == condition.Type {
			conditions[i] = c
			return conditions
		}
	}
	conditions = append(conditions, c)
	return conditions
}

func ConvertToUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: unstructuredObj}, nil
}

func CheckStatus(obj runtime.Object) status.Status {
	u, err := ConvertToUnstructured(obj)
	if err != nil {
		return status.UnknownStatus
	}

	compute, err := status.Compute(u)

	if err != nil {
		return status.UnknownStatus
	}

	return compute.Status
}

func MatchSet(set1 map[string]string, set2 map[string]string) bool {
	selector := labels.SelectorFromSet(set2)
	return selector.Matches(labels.Set(set1))
}

func GetHash(s string) (uint32, error) {
	hasher := fnv.New32a()
	_, err := hasher.Write([]byte(s))
	if err != nil {
		return 0, err
	}
	return hasher.Sum32(), nil
}

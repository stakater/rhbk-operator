package resources

import (
	"encoding/json"
	"fmt"
	"github.com/stakater/rhbk-operator/internal/constants"
	v1 "k8s.io/api/batch/v1"
	"strconv"
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

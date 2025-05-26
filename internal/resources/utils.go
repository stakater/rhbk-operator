package resources

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/stakater/rhbk-operator/internal/constants"
)

func FormatResource(obj interface{}) string {
	jsonData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v\n", obj)
	}

	return string(jsonData)
}

func DecorateDefaultLabels(existing map[string]string) {
	if existing == nil {
		existing = make(map[string]string)
	}

	existing["app"] = "rhbk"
}

func IsJobCompleted(job *v1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == v1.JobComplete {
			return condition.Status == v13.ConditionTrue
		}
	}

	return false
}

func IsStatefulSetReady(sts *v12.StatefulSet) bool {
	if sts == nil {
		return false
	}

	return sts.Status.ReadyReplicas == sts.Status.Replicas
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

func GetOwnerLabels(ownerName string, ownerNamespace string) map[string]string {
	return map[string]string{
		constants.RHBKImportOwnerLabel:     ownerName,
		constants.RHBKImportNamespaceLabel: ownerNamespace,
	}
}

func AddOrReplaceEnv(env v13.EnvVar, vars []v13.EnvVar) []v13.EnvVar {
	found := false
	for i, envVar := range vars {
		if envVar.Name == env.Name {
			vars[i] = env
			found = true
			break
		}
	}

	if !found {
		vars = append(vars, env)
	}

	return vars
}

// EscapeString escapes a value using json.Marshal to ensure it's properly escaped.
// For PEM and key formats, it preserves newlines and special characters.
func EscapeString(value string) (string, error) {
	// Check if the string appears to be a PEM or key format
	if strings.Contains(value, "-----BEGIN") || strings.Contains(value, "-----END") {
		return value, nil
	}

	// For other strings, use json.Marshal
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to escape value: %w", err)
	}
	// Remove the surrounding quotes that json.Marshal adds
	return string(b[1 : len(b)-1]), nil
}

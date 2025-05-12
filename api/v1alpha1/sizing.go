package v1alpha1

import (
	"math"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// RealmSizing represents the sizing configuration for a Keycloak realm
// https://www.keycloak.org/high-availability/concepts-memory-and-cpu-sizing
type RealmSizing struct {
	// Number of logins to create
	LoginsPerSecond int32 `json:"loginsPerSecond"`

	// Number of clients to create
	ClientCredentialsGrantsPerSecond int32 `json:"clientCredentialsGrantsPerSecond"`

	// Number of users to create
	RefreshTokenGrantsPerSecond int32 `json:"refreshTokenGrantsPerSecond"`

	// Number of cached sessions, defaults to 10000
	CachedSessions int32 `json:"cachedSessions"`
}

// CalculateResourceLimits calculates the required resource limits based on the sizing configuration
// and the number of instances. The total load will be distributed across all instances.
func (r *RealmSizing) CalculateResourceLimits(instances *int32) corev1.ResourceRequirements {
	inst := int32(1)
	if instances != nil && *instances > 0 {
		inst = *instances
	}

	// Base memory calculation
	baseMemory := int64(1250)   // MB - Base memory including caches
	nonHeapMemory := int64(300) // MB - Non-heap memory
	// Memory limit = (base - non-heap) / 0.7
	memoryLimitFloat := float64(baseMemory-nonHeapMemory) / 0.7

	// CPU calculation
	var cpuCores float64

	// Password logins (distributed across instances)
	if r.LoginsPerSecond > 0 {
		cpuCores += float64(r.LoginsPerSecond) / (15.0 * float64(inst))
	}

	// Client credentials (distributed across instances)
	if r.ClientCredentialsGrantsPerSecond > 0 {
		cpuCores += float64(r.ClientCredentialsGrantsPerSecond) / (120.0 * float64(inst))
	}

	// Refresh tokens (distributed across instances)
	if r.RefreshTokenGrantsPerSecond > 0 {
		cpuCores += float64(r.RefreshTokenGrantsPerSecond) / (120.0 * float64(inst))
	}

	// CPU request is the base CPU needed
	cpuRequest := int64(cpuCores * 1000)
	// CPU limit includes 150% headroom
	cpuLimit := int64(cpuCores * 2.5 * 1000)

	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(cpuRequest, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(baseMemory*1024*1024, resource.BinarySI),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(cpuLimit, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(int64(math.Ceil(memoryLimitFloat/10)*10)*1024*1024, resource.BinarySI),
		},
	}
}

// SumSizing sums all non-nil RealmSizing values and uses defaultSizing for nil values.
func SumSizing(sizings []*RealmSizing, defaultSizing *RealmSizing) RealmSizing {
	var sum RealmSizing
	for _, s := range sizings {
		if s == nil {
			if defaultSizing != nil {
				sum.LoginsPerSecond += defaultSizing.LoginsPerSecond
				sum.ClientCredentialsGrantsPerSecond += defaultSizing.ClientCredentialsGrantsPerSecond
				sum.RefreshTokenGrantsPerSecond += defaultSizing.RefreshTokenGrantsPerSecond
				sum.CachedSessions += defaultSizing.CachedSessions
			}
		} else {
			sum.LoginsPerSecond += s.LoginsPerSecond
			sum.ClientCredentialsGrantsPerSecond += s.ClientCredentialsGrantsPerSecond
			sum.RefreshTokenGrantsPerSecond += s.RefreshTokenGrantsPerSecond
			sum.CachedSessions += s.CachedSessions
		}
	}
	return sum
}

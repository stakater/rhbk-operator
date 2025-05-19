package v1alpha1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestRealmSizing_CalculateResourceLimits(t *testing.T) {
	tests := []struct {
		name      string
		sizing    RealmSizing
		instances int32
		want      corev1.ResourceRequirements
	}{
		{
			name: "official example",
			sizing: RealmSizing{
				LoginsPerSecond:                  45,  // 3 vCPUs
				ClientCredentialsGrantsPerSecond: 360, // 3 vCPUs
				RefreshTokenGrantsPerSecond:      360, // 3 vCPUs
			},
			instances: 3,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(3000, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(7500, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name: "official example 1 instance",
			sizing: RealmSizing{
				LoginsPerSecond:                  45,  // 3 vCPUs
				ClientCredentialsGrantsPerSecond: 360, // 3 vCPUs
				RefreshTokenGrantsPerSecond:      360, // 3 vCPUs
			},
			instances: 1,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(9000, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(22500, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name: "single instance with all operations",
			sizing: RealmSizing{
				LoginsPerSecond:                  30,  // 2 vCPUs
				ClientCredentialsGrantsPerSecond: 240, // 2 vCPUs
				RefreshTokenGrantsPerSecond:      120, // 1 vCPU
			},
			instances: 1,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(5000, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(12500, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name: "multiple instances with all operations",
			sizing: RealmSizing{
				LoginsPerSecond:                  30,  // 2 vCPUs total
				ClientCredentialsGrantsPerSecond: 240, // 2 vCPUs total
				RefreshTokenGrantsPerSecond:      120, // 1 vCPU total
			},
			instances: 2,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(2500, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(6250, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name: "only password logins",
			sizing: RealmSizing{
				LoginsPerSecond: 45, // 3 vCPUs
			},
			instances: 1,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(3000, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(7500, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name: "only client credentials",
			sizing: RealmSizing{
				ClientCredentialsGrantsPerSecond: 360, // 3 vCPUs
			},
			instances: 1,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(3000, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(7500, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name: "only refresh tokens",
			sizing: RealmSizing{
				RefreshTokenGrantsPerSecond: 120, // 1 vCPU
			},
			instances: 1,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(1000, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(2500, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name:      "zero instances defaults to one",
			sizing:    RealmSizing{},
			instances: 0,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(0, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(0, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
		{
			name: "disable CPU limits",
			sizing: RealmSizing{
				LoginsPerSecond:                  45,  // 3 vCPUs
				ClientCredentialsGrantsPerSecond: 360, // 3 vCPUs
				RefreshTokenGrantsPerSecond:      360, // 3 vCPUs
				DisableCPULimits:                 true,
			},
			instances: 3,
			want: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(3000, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(1250*1024*1024, resource.BinarySI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(1360*1024*1024, resource.BinarySI),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := tt.instances
			got := tt.sizing.CalculateResourceLimits(&inst)

			// Test CPU requests
			if got.Requests.Cpu().MilliValue() != tt.want.Requests.Cpu().MilliValue() {
				t.Errorf("CalculateResourceLimits() CPU Request = %v, want %v",
					got.Requests.Cpu().MilliValue(),
					tt.want.Requests.Cpu().MilliValue())
			}

			// Test CPU limits
			if tt.sizing.DisableCPULimits {
				if _, exists := got.Limits[corev1.ResourceCPU]; exists {
					t.Errorf("CalculateResourceLimits() should not have CPU limits when DisableCPULimits is true")
				}
			} else {
				if got.Limits.Cpu().MilliValue() != tt.want.Limits.Cpu().MilliValue() {
					t.Errorf("CalculateResourceLimits() CPU Limit = %v, want %v",
						got.Limits.Cpu().MilliValue(),
						tt.want.Limits.Cpu().MilliValue())
				}
			}

			// Test Memory requests
			if got.Requests.Memory().Value() != tt.want.Requests.Memory().Value() {
				t.Errorf("CalculateResourceLimits() Memory Request = %v, want %v",
					got.Requests.Memory().Value(),
					tt.want.Requests.Memory().Value())
			}

			// Test Memory limits
			if got.Limits.Memory().Value() != tt.want.Limits.Memory().Value() {
				t.Errorf("CalculateResourceLimits() Memory Limit = %v, want %v",
					got.Limits.Memory().Value(),
					tt.want.Limits.Memory().Value())
			}
		})
	}
}

func TestSumSizing(t *testing.T) {
	defaultSizing := &RealmSizing{
		LoginsPerSecond:                  10,
		ClientCredentialsGrantsPerSecond: 20,
		RefreshTokenGrantsPerSecond:      30,
		CachedSessions:                   100,
	}

	tests := []struct {
		name          string
		sizings       []*RealmSizing
		defaultSizing *RealmSizing
		want          RealmSizing
	}{
		{
			name: "sum non-nil sizings",
			sizings: []*RealmSizing{
				{
					LoginsPerSecond:                  5,
					ClientCredentialsGrantsPerSecond: 10,
					RefreshTokenGrantsPerSecond:      15,
					CachedSessions:                   50,
				},
				{
					LoginsPerSecond:                  15,
					ClientCredentialsGrantsPerSecond: 20,
					RefreshTokenGrantsPerSecond:      25,
					CachedSessions:                   150,
				},
			},
			defaultSizing: defaultSizing,
			want: RealmSizing{
				LoginsPerSecond:                  20,
				ClientCredentialsGrantsPerSecond: 30,
				RefreshTokenGrantsPerSecond:      40,
				CachedSessions:                   200,
			},
		},
		{
			name: "use defaultSizing for nil sizings",
			sizings: []*RealmSizing{
				nil,
				{
					LoginsPerSecond:                  5,
					ClientCredentialsGrantsPerSecond: 10,
					RefreshTokenGrantsPerSecond:      15,
					CachedSessions:                   50,
				},
			},
			defaultSizing: defaultSizing,
			want: RealmSizing{
				LoginsPerSecond:                  15,
				ClientCredentialsGrantsPerSecond: 30,
				RefreshTokenGrantsPerSecond:      45,
				CachedSessions:                   150,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SumSizing(tt.sizings, tt.defaultSizing)
			if got != tt.want {
				t.Errorf("SumSizing() = %v, want %v", got, tt.want)
			}
		})
	}
}

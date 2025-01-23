package main

import (
	"encoding/json"
	"testing"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
)

func TestGenerateHostname(t *testing.T) {
	clusterDefaults := cistore.ClusterDefaults{
		ClusterName: "cluster",
		ShortName:   "cl",
		NidLength:   4,
	}

	tests := []struct {
		clusterName string
		component   cistore.OpenCHAMIComponent
		expected    string
	}{
		{
			clusterName: "cluster",
			component: cistore.OpenCHAMIComponent{
				Component: base.Component{
					Role: "compute",
					NID:  json.Number("1234"),
				},
			},
			expected: "cl1234",
		},
		{
			clusterName: "cluster",
			component: cistore.OpenCHAMIComponent{
				Component: base.Component{
					Role: "io",
					NID:  json.Number("12"),
				},
			},
			expected: "cl0012",
		},
		{
			clusterName: "cluster",
			component: cistore.OpenCHAMIComponent{
				Component: base.Component{
					Role: "front_end",
					NID:  json.Number("34"),
				},
			},
			expected: "cl0034",
		},
		{
			clusterName: "cluster",
			component: cistore.OpenCHAMIComponent{
				Component: base.Component{
					Role: "unknown",
					NID:  json.Number("5678"),
				},
			},
			expected: "cl5678",
		},
		{
			clusterName: "cluster",
			component: cistore.OpenCHAMIComponent{
				Component: base.Component{
					NID: json.Number("5678"),
				},
			},
			expected: "cl5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := generateHostname(tt.clusterName, clusterDefaults.ShortName, clusterDefaults.NidLength, tt.component)
			if got != tt.expected {
				t.Errorf("generateHostname() = %v, want %v", got, tt.expected)
			}
		})
	}
}

package main

import (
	"encoding/json"
	"testing"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
)

func TestGenerateHostname(t *testing.T) {
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
			expected: "cl-io12",
		},
		{
			clusterName: "cluster",
			component: cistore.OpenCHAMIComponent{
				Component: base.Component{
					Role: "front_end",
					NID:  json.Number("34"),
				},
			},
			expected: "cl-fe34",
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
			got := generateHostname(tt.clusterName, tt.component)
			if got != tt.expected {
				t.Errorf("generateHostname() = %v, want %v", got, tt.expected)
			}
		})
	}
}

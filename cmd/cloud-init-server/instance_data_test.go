package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSMDClient struct {
	mock.Mock
}

func (m *MockSMDClient) ComponentInformation(id string) (base.Component, error) {
	args := m.Called(id)
	return args.Get(0).(base.Component), args.Error(1)
}

func (m *MockSMDClient) GroupMembership(id string) ([]string, error) {
	args := m.Called(id)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSMDClient) IDfromMAC(mac string) (string, error) {
	args := m.Called(mac)
	return args.String(0), args.Error(1)
}

func (m *MockSMDClient) IDfromIP(ipaddr string) (string, error) {
	args := m.Called(ipaddr)
	return args.String(0), args.Error(1)
}

func (m *MockSMDClient) IPfromID(id string) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockSMDClient) PopulateNodes() {
	m.Called()
}

func TestInstanceDataHandler(t *testing.T) {
	mockSMDClient := new(MockSMDClient)
	clusterName := "test-cluster"

	component := base.Component{
		ID:   "test-id",
		Role: "compute",
		NID:  json.Number(fmt.Sprint(1)),
	}

	mockSMDClient.On("ComponentInformation", "192.168.1.1").Return(component, nil)
	mockSMDClient.On("GroupMembership", "192.168.1.1").Return([]string{"group1", "group2"}, nil)

	handler := InstanceDataHandler(mockSMDClient, clusterName)

	t.Run("returns instance data as json", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/latest/instance-data", nil)
		req.RemoteAddr = "192.168.1.1"
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var responseData citypes.ClusterData
		err := json.NewDecoder(resp.Body).Decode(&responseData)
		assert.NoError(t, err)
		assert.Equal(t, "OpenCHAMI", responseData.InstanceData.V1.CloudName)
		assert.Equal(t, "lanl-yellow", responseData.InstanceData.V1.AvailabilityZone)
		assert.Equal(t, "t2.micro", responseData.InstanceData.V1.InstanceType)
		assert.Equal(t, "us-west", responseData.InstanceData.V1.Region)
		assert.Equal(t, "OpenCHAMI", responseData.InstanceData.V1.CloudProvider)
		assert.Equal(t, []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD..."}, responseData.InstanceData.V1.PublicKeys)
		assert.Equal(t, "1.0", responseData.InstanceData.V1.VendorData.Version)
		assert.Len(t, responseData.InstanceData.V1.VendorData.Groups, 2)
	})

	t.Run("returns 404 if component information is not available", func(t *testing.T) {
		mockSMDClient.On("ComponentInformation", "192.168.1.2").Return(base.Component{}, assert.AnError)

		req := httptest.NewRequest("GET", "/latest/instance-data", nil)
		req.RemoteAddr = "192.168.1.2"
		w := httptest.NewRecorder()

		handler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, "Node not found in SMD. Instance-data not available\n", w.Body.String())
	})

	t.Run("uses X-Forwarded-For header if present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/latest/instance-data", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.3")
		w := httptest.NewRecorder()

		mockSMDClient.On("ComponentInformation", "192.168.1.3").Return(component, nil)
		mockSMDClient.On("GroupMembership", "192.168.1.3").Return([]string{"group1", "group2"}, nil)

		handler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})
}

package main

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
)

// TestBase64Decoding tests that base64 decoding doesn't produce trailing NUL bytes
func TestBase64Decoding(t *testing.T) {
	// Create test content
	originalContent := `#cloud-config
packages:
  - wget
  - curl
write_files:
  - path: /etc/test.conf
    content: |
      test configuration file
      with multiple lines
    permissions: '0600'`

	// Encode it to base64
	encodedContent := base64.StdEncoding.EncodeToString([]byte(originalContent))

	// Create test group data with base64 encoded content
	store := memstore.NewMemStore()
	groupData := cistore.GroupData{
		Name:        "test-base64",
		Description: "Test group for base64 decoding",
		File: cistore.CloudConfigFile{
			Content:  []byte(encodedContent),
			Encoding: "base64",
		},
	}

	err := store.AddGroupData("test-base64", groupData)
	if err != nil {
		t.Fatalf("Failed to add group data: %v", err)
	}

	// Retrieve and decode (simulating what the handler does)
	retrievedData, err := store.GetGroupData("test-base64")
	if err != nil {
		t.Fatalf("Failed to retrieve group data: %v", err)
	}

	// Perform the base64 decoding using our new approach
	if retrievedData.File.Encoding == "base64" {
		decodedContent, err := base64.StdEncoding.DecodeString(string(retrievedData.File.Content))
		if err != nil {
			t.Fatalf("Failed to base64-decode content: %v", err)
		}
		retrievedData.File.Content = decodedContent
		retrievedData.File.Encoding = "plain"
	}

	// Verify the decoded content
	decodedStr := string(retrievedData.File.Content)

	// Check that content matches original
	if decodedStr != originalContent {
		t.Errorf("Decoded content doesn't match original.\nExpected:\n%s\nGot:\n%s", originalContent, decodedStr)
	}

	// Check for trailing NUL bytes (the main issue from PR #86)
	if strings.Contains(decodedStr, "\x00") {
		t.Error("Decoded content contains NUL bytes (\\x00)")
	}

	// Check for the specific pattern mentioned in the PR
	if strings.HasSuffix(decodedStr, "\x00") {
		t.Error("Decoded content has trailing NUL bytes")
	}

	// Verify length is exactly what we expect
	if len(decodedStr) != len(originalContent) {
		t.Errorf("Decoded content length mismatch. Expected %d, got %d", len(originalContent), len(decodedStr))
	}

	t.Logf("Successfully decoded %d bytes without NUL bytes", len(decodedStr))
}

// TestBase64DecodingWithVariousContent tests decoding with different content sizes
func TestBase64DecodingWithVariousContent(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "short content",
			content: "test",
		},
		{
			name:    "medium content",
			content: strings.Repeat("Hello, World! ", 100),
		},
		{
			name:    "long content",
			content: strings.Repeat("This is a longer test string with various characters: !@#$%^&*()_+-={}[]|\\:;\"'<>?,./\n", 50),
		},
		{
			name:    "content with newlines",
			content: "#cloud-config\npackages:\n  - vim\n  - git\nruncmd:\n  - echo 'Hello World'\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode to base64
			encodedContent := base64.StdEncoding.EncodeToString([]byte(tc.content))

			// Decode using our new approach
			decodedContent, err := base64.StdEncoding.DecodeString(encodedContent)
			if err != nil {
				t.Fatalf("Failed to decode base64 content: %v", err)
			}

			decodedStr := string(decodedContent)

			// Verify content matches
			if decodedStr != tc.content {
				t.Errorf("Content mismatch for %s", tc.name)
			}

			// Check for NUL bytes
			if strings.Contains(decodedStr, "\x00") {
				t.Errorf("Content contains NUL bytes for %s", tc.name)
			}

			// Check length
			if len(decodedStr) != len(tc.content) {
				t.Errorf("Length mismatch for %s. Expected %d, got %d", tc.name, len(tc.content), len(decodedStr))
			}
		})
	}
}

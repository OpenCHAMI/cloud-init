package cistore

import (
	"encoding/base64"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudConfigFile_UnmarshalJSON_Plain(t *testing.T) {
	jsonData := []byte(`{
        "filename": "myconfig.yaml",
        "encoding": "plain",
        "content": "#cloud-config\nusers:\n  - name: test"
    }`)

	var f CloudConfigFile
	err := json.Unmarshal(jsonData, &f)
	assert.NoError(t, err)
	assert.Equal(t, "myconfig.yaml", f.Name)
	assert.Equal(t, "plain", f.Encoding)
	assert.Equal(t, []byte("#cloud-config\nusers:\n  - name: test"), f.Content)
}

func TestCloudConfigFile_UnmarshalYAML_Plain(t *testing.T) {
	yamlData := []byte(`filename: myconfig.yaml
encoding: plain
content: "#cloud-config\nusers:\n  - name: test"`)

	var f CloudConfigFile
	err := yaml.Unmarshal(yamlData, &f)
	assert.NoError(t, err)
	assert.Equal(t, "myconfig.yaml", f.Name)
	assert.Equal(t, "plain", f.Encoding)
	assert.Equal(t, []byte("#cloud-config\nusers:\n  - name: test"), f.Content)
}

func TestCloudConfigFile_UnmarshalJSON_Base64(t *testing.T) {
	encodedContent := base64.StdEncoding.EncodeToString([]byte("#cloud-config\nusers:\n  - name: test"))
	jsonData := []byte(`{
        "filename": "myconfig.yaml",
        "encoding": "base64",
        "content": "` + encodedContent + `"
    }`)

	var f CloudConfigFile
	err := json.Unmarshal(jsonData, &f)
	assert.NoError(t, err, "Unmarshal should succeed for base64 content")
	assert.Equal(t, "myconfig.yaml", f.Name)
	assert.Equal(t, "base64", f.Encoding)
	// Even though "encoding" is "base64" in JSON, the unmarshaler does NOT auto-decode.
	assert.Equal(t, []byte(encodedContent), f.Content)
}

func TestCloudConfigFile_UnmarshalYAML_Base64(t *testing.T) {
	encodedContent := base64.StdEncoding.EncodeToString([]byte("#cloud-config\nusers:\n  - name: test"))
	yamlData := []byte(`filename: myconfig.yaml
encoding: base64
content: ` + encodedContent)

	var f CloudConfigFile
	err := yaml.Unmarshal(yamlData, &f)
	assert.NoError(t, err)
	assert.Equal(t, "myconfig.yaml", f.Name)
	assert.Equal(t, "base64", f.Encoding)
	assert.Equal(t, []byte(encodedContent), f.Content)
}

func TestCloudConfigFile_MarshalJSON_Plain(t *testing.T) {
	f := CloudConfigFile{
		Name:     "plainconfig.yaml",
		Encoding: "plain",
		Content:  []byte("#cloud-config\nusers:\n  - name: test"),
	}

	data, err := json.Marshal(f)
	assert.NoError(t, err)

	var out map[string]interface{}
	err = json.Unmarshal(data, &out)
	assert.NoError(t, err)
	assert.Equal(t, "plainconfig.yaml", out["filename"])
	assert.Equal(t, "plain", out["encoding"])
	assert.Equal(t, "#cloud-config\nusers:\n  - name: test", out["content"])
}

func TestCloudConfigFile_MarshalYAML_Plain(t *testing.T) {
	f := CloudConfigFile{
		Name:     "plainconfig.yaml",
		Encoding: "plain",
		Content:  []byte("#cloud-config\nusers:\n  - name: test"),
	}

	data, err := yaml.Marshal(f)
	assert.NoError(t, err)

	var out map[string]interface{}
	err = yaml.Unmarshal(data, &out)
	assert.NoError(t, err)
	assert.Equal(t, "plainconfig.yaml", out["filename"])
	assert.Equal(t, "plain", out["encoding"])
	assert.Equal(t, "#cloud-config\nusers:\n  - name: test", out["content"])
}

func TestCloudConfigFile_MarshalJSON_Base64(t *testing.T) {
	originalConfig := "#cloud-config\nusers:\n  - name: test"
	b64Config := base64.StdEncoding.EncodeToString([]byte(originalConfig))
	f := CloudConfigFile{
		Name:     "encodedconfig.yaml",
		Encoding: "base64",
		Content:  []byte(b64Config),
	}

	data, err := json.Marshal(f)
	assert.NoError(t, err)

	var out map[string]interface{}
	err = json.Unmarshal(data, &out)
	assert.NoError(t, err)
	assert.Equal(t, "encodedconfig.yaml", out["filename"])
	assert.Equal(t, "base64", out["encoding"])
	assert.Equal(t, b64Config, out["content"])
}

func TestCloudConfigFile_MarshalYAML_Base64(t *testing.T) {
	originalConfig := "#cloud-config\nusers:\n  - name: test"
	b64Config := base64.StdEncoding.EncodeToString([]byte(originalConfig))
	f := CloudConfigFile{
		Name:     "encodedconfig.yaml",
		Encoding: "base64",
		Content:  []byte(b64Config),
	}

	data, err := yaml.Marshal(f)
	assert.NoError(t, err)

	var out map[string]interface{}
	err = yaml.Unmarshal(data, &out)
	assert.NoError(t, err)
	assert.Equal(t, "encodedconfig.yaml", out["filename"])
	assert.Equal(t, "base64", out["encoding"])
	assert.Equal(t, b64Config, out["content"])
}

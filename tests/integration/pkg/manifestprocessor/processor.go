package manifestprocessor

import (
	"bytes"
	"fmt"
	yaml3 "gopkg.in/yaml.v3"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path"
)

func ParseYamlFromFile(fileName string, directory string) ([]unstructured.Unstructured, error) {
	rawData, err := os.ReadFile(path.Join(directory, fileName))
	if err != nil {
		return nil, err
	}

	var manifests []unstructured.Unstructured
	decoder := yaml3.NewDecoder(bytes.NewBufferString(string(rawData)))
	for {
		var d map[string]interface{}
		if err := decoder.Decode(&d); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("Document decode failed: %w", err)
		}
		manifests = append(manifests, unstructured.Unstructured{Object: d})
	}
	return manifests, nil
}

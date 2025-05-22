package manifestprocessor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	yaml3 "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ParseYamlFromFile(fileName string) ([]unstructured.Unstructured, error) {
	rawData, err := os.ReadFile(path.Join("steps", fileName))
	if err != nil {
		return nil, err
	}

	var manifests []unstructured.Unstructured
	decoder := yaml3.NewDecoder(bytes.NewBuffer(rawData))
	for {
		var d map[string]interface{}
		if err := decoder.Decode(&d); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("document decode failed: %w", err)
		}
		manifests = append(manifests, unstructured.Unstructured{Object: d})
	}
	return manifests, nil
}

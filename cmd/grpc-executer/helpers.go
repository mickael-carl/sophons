package main

import (
	"fmt"

	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mickael-carl/sophons/pkg/exec"
)

// convertProtoStructToMap converts a google.protobuf.Struct to a map[string]any.
func convertProtoStructToMap(protoStruct *structpb.Struct) (map[string]any, error) {
	if protoStruct == nil {
		return make(map[string]any), nil
	}
	return protoStruct.AsMap(), nil
}

// convertMapToProtoStruct converts a map[string]any to a google.protobuf.Struct.
func convertMapToProtoStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		return nil, nil
	}
	return structpb.NewStruct(m)
}

// resultToMap converts an exec.Result to a map[string]any.
// This is needed to serialize results to protobuf Struct.
func resultToMap(result exec.Result) (map[string]any, error) {
	if result == nil {
		return make(map[string]any), nil
	}

	// Use YAML marshaling as an intermediate format to convert to map
	yamlBytes, err := yaml.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to YAML: %w", err)
	}

	var resultMap map[string]any
	if err := yaml.Unmarshal(yamlBytes, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result from YAML: %w", err)
	}

	return resultMap, nil
}

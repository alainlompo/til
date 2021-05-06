package transformers

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"bridgedl/internal/sdk/k8s"
	"bridgedl/translation"
)

type Bumblebee struct{}

var (
	_ translation.Decodable    = (*Bumblebee)(nil)
	_ translation.Translatable = (*Bumblebee)(nil)
	_ translation.Addressable  = (*Bumblebee)(nil)
)

// Spec implements translation.Decodable.
func (*Bumblebee) Spec() hcldec.Spec {
	nestedSpec := &hcldec.BlockListSpec{
		TypeName: "operation",
		Nested: &hcldec.ObjectSpec{
			"operation": &hcldec.BlockLabelSpec{
				Index: 0,
				Name:  "operation",
			},
			"path": &hcldec.BlockListSpec{
				TypeName: "path",
				MinItems: 1,
				Nested: &hcldec.ObjectSpec{
					"key": &hcldec.AttrSpec{
						Name: "key",
						Type: cty.String,
					},
					"value": &hcldec.AttrSpec{
						Name: "value",
						Type: cty.String,
					},
				},
			},
		},
	}

	return &hcldec.ObjectSpec{
		"context": &hcldec.BlockSpec{
			TypeName: "context",
			Nested:   nestedSpec,
		},
		"data": &hcldec.BlockSpec{
			TypeName: "data",
			Nested:   nestedSpec,
		},
	}

	/*
		Example of value decoded from the spec above, for a "context"
		and a "data" block, both containing two nested "operation" blocks
		with one or more "path" sub-blocks:

		v: (map[string]interface {}) (len=2) {
		 "context": ([]interface {}) (len=2) {
		  (map[string]interface {}) (len=2) {
		   "operation": (string) "store",
		   "path": ([]interface {}) (len=2) {
		    (map[string]interface {}) (len=2) {
		     "key": (string) "$id",
		     "value": (string) "id"
		    },
		    (map[string]interface {}) (len=2) {
		     "key": (string) "$type",
		     "value": (string) "type"
		    }
		   }
		  },
		  (map[string]interface {}) (len=2) {
		   "operation": (string) "add",
		   "path": ([]interface {}) (len=1) {
		    (map[string]interface {}) (len=2) {
		     "key": (string) "idtype",
		     "value": (string) "${id}-${type}"
		    }
		   }
		  }
		 },
		 "data": ([]interface {}) (len=2) { ... }
		}
	*/
}

// Manifests implements translation.Translatable.
func (*Bumblebee) Manifests(id string, config, eventDst cty.Value) []interface{} {
	var manifests []interface{}

	transf := &unstructured.Unstructured{}
	transf.SetAPIVersion("flow.triggermesh.io/v1alpha1")
	transf.SetKind("Transformation")
	transf.SetName(k8s.RFC1123Name(id))

	context := parseBumblebeeOperations(config.GetAttr("context").AsValueSlice())
	_ = unstructured.SetNestedSlice(transf.Object, context, "spec", "context")

	data := parseBumblebeeOperations(config.GetAttr("data").AsValueSlice())
	_ = unstructured.SetNestedSlice(transf.Object, data, "spec", "data")

	sinkRef := eventDst.GetAttr("ref")
	sink := map[string]interface{}{
		"apiVersion": sinkRef.GetAttr("apiVersion").AsString(),
		"kind":       sinkRef.GetAttr("kind").AsString(),
		"name":       sinkRef.GetAttr("name").AsString(),
	}
	_ = unstructured.SetNestedMap(transf.Object, sink, "spec", "sink", "ref")

	return append(manifests, transf)
}

// Address implements translation.Addressable.
func (*Bumblebee) Address(id string, _, _ cty.Value) cty.Value {
	return k8s.NewDestination("flow.triggermesh.io/v1alpha1", "Transformation", k8s.RFC1123Name(id))
}

func parseBumblebeeOperations(operationVals []cty.Value) []interface{} {
	operations := make([]interface{}, 0, len(operationVals))

	for _, operationVal := range operationVals {
		operationValMap := operationVal.AsValueMap()

		operation := operationValMap["operation"].AsString()

		pathVals := operationValMap["path"].AsValueSlice()
		paths := make([]interface{}, 0, len(pathVals))
		for _, pathVal := range pathVals {
			path := make(map[string]interface{})

			pathValMap := pathVal.AsValueMap()

			if key := pathValMap["key"]; !key.IsNull() {
				path["key"] = key.AsString()
			}
			if value := pathValMap["value"]; !value.IsNull() {
				path["value"] = value.AsString()
			}

			paths = append(paths, path)
		}

		operations = append(operations, map[string]interface{}{
			"operation": operation,
			"paths":     paths,
		})
	}

	return operations
}

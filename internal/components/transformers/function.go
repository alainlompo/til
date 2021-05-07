package transformers

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"bridgedl/internal/sdk/k8s"
	"bridgedl/translation"
)

type Function struct{}

var (
	_ translation.Decodable    = (*Function)(nil)
	_ translation.Translatable = (*Function)(nil)
	_ translation.Addressable  = (*Function)(nil)
)

// Spec implements translation.Decodable.
func (*Function) Spec() hcldec.Spec {
	return &hcldec.ObjectSpec{
		"runtime": &hcldec.AttrSpec{
			Name:     "runtime",
			Type:     cty.String,
			Required: true,
		},
		"code": &hcldec.AttrSpec{
			Name:     "code",
			Type:     cty.String,
			Required: true,
		},
		"entrypoint": &hcldec.AttrSpec{
			Name:     "entrypoint",
			Type:     cty.String,
			Required: false,
		},
		"public": &hcldec.AttrSpec{
			Name:     "public",
			Type:     cty.Bool,
			Required: false,
		},
	}
}

// Manifests implements translation.Translatable.
func (*Function) Manifests(id string, config, eventDst cty.Value) []interface{} {
	var manifests []interface{}

	name := k8s.RFC1123Name(id)

	code := config.GetAttr("code").AsString()

	switch runtime := config.GetAttr("runtime").AsString(); runtime {
	case "js":
		t := k8s.NewObject("targets.triggermesh.io/v1alpha1", "InfraTarget", name)

		t.SetNestedField(code, "spec", "script", "code")

		// route responses via a channel subscription
		ch := k8s.NewChannel(name)
		subs := k8s.NewSubscription(name, name,
			k8s.NewDestination("targets.triggermesh.io/v1alpha1", "InfraTarget", name),
			eventDst,
		)

		manifests = append(manifests, t.Unstructured(), ch, subs)

	default:
		f := k8s.NewObject("flow.triggermesh.io/v1alpha1", "Function", name)

		f.SetNestedField(runtime, "spec", "runtime")
		f.SetNestedField(code, "spec", "code")

		entrypoint := "main"
		if v := config.GetAttr("entrypoint"); !v.IsNull() {
			entrypoint = v.AsString()
		}
		f.SetNestedField(entrypoint, "spec", "entrypoint")

		sink := k8s.DecodeDestination(eventDst)
		f.SetNestedMap(sink, "spec", "sink", "ref")

		public := config.GetAttr("public").True()
		f.SetNestedField(public, "spec", "public")

		manifests = append(manifests, f.Unstructured())
	}

	return manifests
}

// Address implements translation.Addressable.
func (*Function) Address(id string, config, _ cty.Value) cty.Value {
	name := k8s.RFC1123Name(id)

	if config.GetAttr("runtime").AsString() == "js" {
		return k8s.NewDestination(k8s.APIMessaging, "Channel", name)
	}
	return k8s.NewDestination("flow.triggermesh.io/v1alpha1", "Function", name)
}

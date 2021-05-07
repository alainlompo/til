package sources

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"bridgedl/internal/sdk"
	"bridgedl/internal/sdk/k8s"
	"bridgedl/internal/sdk/secrets"
	"bridgedl/translation"
)

type AzureActivityLogs struct{}

var (
	_ translation.Decodable    = (*AzureActivityLogs)(nil)
	_ translation.Translatable = (*AzureActivityLogs)(nil)
)

// Spec implements translation.Decodable.
func (*AzureActivityLogs) Spec() hcldec.Spec {
	return &hcldec.ObjectSpec{
		"event_hub_id": &hcldec.AttrSpec{
			Name:     "event_hub_id",
			Type:     cty.String,
			Required: true,
		},
		"event_hubs_sas_policy": &hcldec.AttrSpec{
			Name:     "event_hubs_sas_policy",
			Type:     cty.String,
			Required: false,
		},
		"categories": &hcldec.AttrSpec{
			Name:     "categories",
			Type:     cty.List(cty.String),
			Required: false,
		},
		"auth": &hcldec.AttrSpec{
			Name:     "auth",
			Type:     k8s.ObjectReferenceCty,
			Required: true,
		},
	}
}

// Manifests implements translation.Translatable.
func (*AzureActivityLogs) Manifests(id string, config, eventDst cty.Value) []interface{} {
	var manifests []interface{}

	s := k8s.NewObject("sources.triggermesh.io/v1alpha1", "AzureActivityLogsSource", id)

	eventHubID := config.GetAttr("event_hub_id").AsString()
	s.SetNestedField(eventHubID, "spec", "eventHubID")

	if v := config.GetAttr("event_hubs_sas_policy"); !v.IsNull() {
		eventHubsSASPolicy := v.AsString()
		s.SetNestedField(eventHubsSASPolicy, "spec", "eventHubsSASPolicy")
	}

	if v := config.GetAttr("categories"); !v.IsNull() {
		categories := sdk.DecodeStringSlice(v)
		s.SetNestedSlice(categories, "spec", "categories")
	}

	authSecretName := config.GetAttr("auth").GetAttr("name").AsString()
	tenantIDSecretRef, clientIDSecretRef, clientSecrSecretRef := secrets.SecretKeyRefsAzureSP(authSecretName)
	s.SetNestedMap(tenantIDSecretRef, "spec", "auth", "servicePrincipal", "tenantID", "valueFromSecret")
	s.SetNestedMap(clientIDSecretRef, "spec", "auth", "servicePrincipal", "clientID", "valueFromSecret")
	s.SetNestedMap(clientSecrSecretRef, "spec", "auth", "servicePrincipal", "clientSecret", "valueFromSecret")

	sinkRef := eventDst.GetAttr("ref")
	sink := map[string]interface{}{
		"apiVersion": sinkRef.GetAttr("apiVersion").AsString(),
		"kind":       sinkRef.GetAttr("kind").AsString(),
		"name":       sinkRef.GetAttr("name").AsString(),
	}
	s.SetNestedMap(sink, "spec", "sink", "ref")

	return append(manifests, s.Unstructured())
}

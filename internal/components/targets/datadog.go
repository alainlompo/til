/*
Copyright 2021 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package targets

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"til/config/globals"
	"til/internal/sdk/k8s"
	"til/internal/sdk/secrets"
	"til/translation"
)

type Datadog struct{}

var (
	_ translation.Decodable    = (*Datadog)(nil)
	_ translation.Translatable = (*Datadog)(nil)
	_ translation.Addressable  = (*Datadog)(nil)
)

// Spec implements translation.Decodable.
func (*Datadog) Spec() hcldec.Spec {
	return &hcldec.ObjectSpec{
		"metric_prefix": &hcldec.AttrSpec{
			Name:     "metric_prefix",
			Type:     cty.String,
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
func (*Datadog) Manifests(id string, config, eventDst cty.Value, glb globals.Accessor) []interface{} {
	var manifests []interface{}

	name := k8s.RFC1123Name(id)

	t := k8s.NewObject(k8s.APITargets, "DatadogTarget", name)

	if v := config.GetAttr("metric_prefix"); !v.IsNull() {
		mp := config.GetAttr("metric_prefix").AsString()
		t.SetNestedField(mp, "spec", "metricPrefix")
	}

	authSecretName := config.GetAttr("auth").GetAttr("name").AsString()
	apiKeySecretRef := secrets.SecretKeyRefsDatadog(authSecretName)
	t.SetNestedMap(apiKeySecretRef, "spec", "apiKey", "secretKeyRef")

	manifests = append(manifests, t.Unstructured())

	if !eventDst.IsNull() {
		ch := k8s.NewChannel(name)

		subscriber := k8s.NewDestination(k8s.APITargets, "DatadogTarget", name)

		sbOpts := []k8s.SubscriptionOption{k8s.ReplyDest(eventDst)}
		sbOpts = k8s.AppendDeliverySubscriptionOptions(sbOpts, glb)

		subs := k8s.NewSubscription(name, name, subscriber, sbOpts...)

		manifests = append(manifests, ch, subs)
	}

	return manifests
}

// Address implements translation.Addressable.
func (*Datadog) Address(id string, _, eventDst cty.Value) cty.Value {
	name := k8s.RFC1123Name(id)

	if eventDst.IsNull() {
		return k8s.NewDestination(k8s.APITargets, "DatadogTarget", name)
	}
	return k8s.NewDestination(k8s.APIMessaging, "Channel", name)
}

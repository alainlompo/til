package sources

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"bridgedl/internal/sdk/secrets"
	"bridgedl/k8s"
	"bridgedl/translation"
)

type Kafka struct{}

var (
	_ translation.Decodable    = (*Kafka)(nil)
	_ translation.Translatable = (*Kafka)(nil)
)

// Spec implements translation.Decodable.
func (*Kafka) Spec() hcldec.Spec {
	return &hcldec.ObjectSpec{
		"consumer_group": &hcldec.AttrSpec{
			Name:     "consumer_group",
			Type:     cty.String,
			Required: false,
		},
		"bootstrap_servers": &hcldec.AttrSpec{
			Name:     "bootstrap_servers",
			Type:     cty.List(cty.String),
			Required: true,
		},
		"topics": &hcldec.AttrSpec{
			Name:     "topics",
			Type:     cty.List(cty.String),
			Required: true,
		},
		"sasl_auth": &hcldec.AttrSpec{
			Name:     "sasl_auth",
			Type:     k8s.ObjectReferenceCty,
			Required: false,
		},
		"tls": &hcldec.ValidateSpec{
			Wrapped: &hcldec.AttrSpec{
				Name:     "tls",
				Type:     cty.DynamicPseudoType,
				Required: false,
			},
			Func: validateKafkaAttrTLS,
		},
	}
}

// Manifests implements translation.Translatable.
func (*Kafka) Manifests(id string, config, eventDst cty.Value) []interface{} {
	var manifests []interface{}

	s := &unstructured.Unstructured{}
	s.SetAPIVersion("sources.knative.dev/v1beta1")
	s.SetKind("KafkaSource")
	s.SetName(k8s.RFC1123Name(id))

	if v := config.GetAttr("consumer_group"); !v.IsNull() {
		consumerGroup := v.AsString()
		_ = unstructured.SetNestedField(s.Object, consumerGroup, "spec", "consumerGroup")
	}

	var bootstrapServers []interface{}
	bSrvsIter := config.GetAttr("bootstrap_servers").ElementIterator()
	for bSrvsIter.Next() {
		_, srv := bSrvsIter.Element()
		bootstrapServers = append(bootstrapServers, srv.AsString())
	}
	_ = unstructured.SetNestedSlice(s.Object, bootstrapServers, "spec", "bootstrapServers")

	var topics []interface{}
	topicsIter := config.GetAttr("topics").ElementIterator()
	for topicsIter.Next() {
		_, topic := topicsIter.Element()
		topics = append(topics, topic.AsString())
	}
	_ = unstructured.SetNestedSlice(s.Object, topics, "spec", "topics")

	if v := config.GetAttr("sasl_auth"); !v.IsNull() {
		saslAuthSecretName := v.GetAttr("name").AsString()
		saslMech, saslUser, saslPasswd, _, _, _ := secrets.SecretKeyRefsKafka(saslAuthSecretName)
		_ = unstructured.SetNestedField(s.Object, true, "spec", "net", "sasl", "enable")
		_ = unstructured.SetNestedMap(s.Object, saslMech, "spec", "net", "sasl", "type", "secretKeyRef")
		_ = unstructured.SetNestedMap(s.Object, saslUser, "spec", "net", "sasl", "user", "secretKeyRef")
		_ = unstructured.SetNestedMap(s.Object, saslPasswd, "spec", "net", "sasl", "password", "secretKeyRef")
	}

	if v := config.GetAttr("tls"); !v.IsNull() {
		if k8s.IsObjectReference(v) {
			tlsSecretName := v.GetAttr("name").AsString()
			_, _, _, caCert, cert, key := secrets.SecretKeyRefsKafka(tlsSecretName)
			_ = unstructured.SetNestedField(s.Object, true, "spec", "net", "tls", "enable")
			_ = unstructured.SetNestedMap(s.Object, caCert, "spec", "net", "tls", "caCert", "secretKeyRef")
			_ = unstructured.SetNestedMap(s.Object, cert, "spec", "net", "tls", "cert", "secretKeyRef")
			_ = unstructured.SetNestedMap(s.Object, key, "spec", "net", "tls", "key", "secretKeyRef")
			// The protocol selection happens at runtime, based on the
			// presence or not of the above keys in the referenced Secret.
			// By marking each of these keys as optional, we attempt to
			// provide configuration parity with the "kafka" target, which
			// uses this same approach.
			_ = unstructured.SetNestedField(s.Object, true, "spec", "net", "tls", "caCert", "secretKeyRef", "optional")
			_ = unstructured.SetNestedField(s.Object, true, "spec", "net", "tls", "cert", "secretKeyRef", "optional")
			_ = unstructured.SetNestedField(s.Object, true, "spec", "net", "tls", "key", "secretKeyRef", "optional")
		} else if v.True() {
			_ = unstructured.SetNestedField(s.Object, true, "spec", "net", "tls", "enable")
		}
	}

	sinkRef := eventDst.GetAttr("ref")
	sink := map[string]interface{}{
		"apiVersion": sinkRef.GetAttr("apiVersion").AsString(),
		"kind":       sinkRef.GetAttr("kind").AsString(),
		"name":       sinkRef.GetAttr("name").AsString(),
	}
	_ = unstructured.SetNestedMap(s.Object, sink, "spec", "sink", "ref")

	return append(manifests, s)
}

func validateKafkaAttrTLS(val cty.Value) hcl.Diagnostics {
	var diags hcl.Diagnostics

	if !(k8s.IsObjectReference(val) || val.Type() == cty.Bool) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attributes type",
			Detail:   `The "tls" attribute accepts either a secret reference or a boolean.`,
		})
	}

	return diags
}

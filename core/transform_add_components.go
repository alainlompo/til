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

package core

import (
	"github.com/hashicorp/hcl/v2"

	"til/config"
	"til/config/addr"
	"til/graph"
)

// MessagingComponentVertex is implemented by all messaging components of a
// Bridge which are represented by a graph.Vertex.
type MessagingComponentVertex interface {
	ComponentAddr() addr.MessagingComponent
	Implementation() interface{}
}

// AddComponentsTransformer is a GraphTransformer that adds all messaging
// components described in a Bridge as vertices of a graph, without connecting
// them.
type AddComponentsTransformer struct {
	Bridge *config.Bridge
}

var _ GraphTransformer = (*AddComponentsTransformer)(nil)

// Transform implements GraphTransformer.
func (t *AddComponentsTransformer) Transform(g *graph.DirectedGraph) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, ch := range t.Bridge.Channels {
		v := &ChannelVertex{
			Addr: addr.Channel{
				Identifier: ch.Identifier,
			},
			Channel: ch,
		}
		g.Add(v)
	}

	for _, rtr := range t.Bridge.Routers {
		v := &RouterVertex{
			Addr: addr.Router{
				Identifier: rtr.Identifier,
			},
			Router: rtr,
		}
		g.Add(v)
	}

	for _, trsf := range t.Bridge.Transformers {
		v := &TransformerVertex{
			Addr: addr.Transformer{
				Identifier: trsf.Identifier,
			},
			Transformer: trsf,
		}
		g.Add(v)
	}

	for _, src := range t.Bridge.Sources {
		v := &SourceVertex{
			Source: src,
		}
		g.Add(v)
	}

	for _, trg := range t.Bridge.Targets {
		v := &TargetVertex{
			Addr: addr.Target{
				Identifier: trg.Identifier,
			},
			Target: trg,
		}
		g.Add(v)
	}

	return diags
}

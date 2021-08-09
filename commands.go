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

package main

import (
	"flag"
	"fmt"
	"io"

	"til/config/file"
	"til/core"
	"til/encoding"
	"til/graph/dot"
)

// CLI subcommands
const (
	cmdGenerate = "generate"
	cmdValidate = "validate"
	cmdGraph    = "graph"
)

// usage is a usageFn for the top level command.
func usage(cmdName string) string {
	return "Interpreter for TriggerMesh's Integration Language.\n" +
		"\n" +
		"USAGE:\n" +
		"    " + cmdName + " <command>\n" +
		"\n" +
		"COMMANDS:\n" +
		"    " + cmdGenerate + "     Generate Kubernetes manifests for deploying a Bridge.\n" +
		"    " + cmdValidate + "     Validate a Bridge description.\n" +
		"    " + cmdGraph + "        Represent a Bridge as a directed graph in DOT format.\n"
}

// usageGenerate is a usageFn for the "generate" subcommand.
func usageGenerate(cmdName string) string {
	return "Generates the Kubernetes manifests which allow a Bridge to be deployed " +
		"to TriggerMesh, and writes them to standard output.\n" +
		"\n" +
		"USAGE:\n" +
		"    " + cmdName + " " + cmdGenerate + " FILE [OPTION]...\n" +
		"\n" +
		"OPTIONS:\n" +
		"    --bridge     Output a Bridge object instead of a List-manifest.\n" +
		"    --yaml       Output generated manifests in YAML format.\n"
}

// usageValidate is a usageFn for the "validate" subcommand.
func usageValidate(cmdName string) string {
	return "Verifies that a Bridge is syntactically valid and can be generated. " +
		"Returns with an exit code of 0 in case of success, with an exit code of 1 " +
		"otherwise.\n" +
		"\n" +
		"USAGE:\n" +
		"    " + cmdName + " " + cmdValidate + " FILE\n"
}

// usageGraph is a usageFn for the "usage" subcommand.
func usageGraph(cmdName string) string {
	return "Generates a DOT representation of a Bridge and writes it to standard " +
		"output.\n" +
		"\n" +
		"USAGE:\n" +
		"    " + cmdName + " " + cmdGraph + " FILE\n"
}

type usageFn func(cmdName string) string

// setUsageFn uses the given usageFn to set the Usage function of the provided
// flag.FlagSet.
func setUsageFn(f *flag.FlagSet, u usageFn) {
	f.Usage = func() {
		fmt.Fprint(f.Output(), u(f.Name()))
	}
}

type Command interface {
	Run(args ...string) error
}

var (
	_ Command = (*GenerateCommand)(nil)
	_ Command = (*ValidateCommand)(nil)
	_ Command = (*GraphCommand)(nil)
)

type GenericCommand struct {
	stdout  io.Writer
	flagSet *flag.FlagSet
}

type GenerateCommand struct {
	GenericCommand

	// flags
	bridge bool
	yaml   bool
}

// Run implements Command.
func (c *GenerateCommand) Run(args ...string) error {
	setUsageFn(c.flagSet, usageGenerate)
	c.flagSet.BoolVar(&c.bridge, "bridge", false, "")
	c.flagSet.BoolVar(&c.yaml, "yaml", false, "")

	pos, flags := splitArgs(1, args)
	_ = c.flagSet.Parse(flags) // ignore err; the FlagSet uses ExitOnError

	if len(pos) != 1 {
		return fmt.Errorf("unexpected number of positional arguments.\n\n%s", usageGenerate(c.flagSet.Name()))
	}
	filePath := pos[0]

	// value to use as the Bridge identifier in case none is defined in the
	// parsed Bridge description
	const defaultBridgeIdentifier = "til_generated"

	brg, diags := file.NewParser().LoadBridge(filePath)
	if diags.HasErrors() {
		return diags
	}

	ctx, diags := core.NewContext(brg)
	if diags.HasErrors() {
		return diags
	}

	manifests, diags := ctx.Generate()
	if diags.HasErrors() {
		return diags
	}

	brgID := brg.Identifier
	if brgID == "" {
		brgID = defaultBridgeIdentifier
	}
	s := encoding.NewSerializer(brgID)

	var w encoding.ManifestsWriterFunc

	switch {
	case c.bridge && c.yaml:
		w = s.WriteBridgeYAML
	case c.bridge:
		w = s.WriteBridgeJSON
	case c.yaml:
		w = s.WriteManifestsYAML
	default:
		w = s.WriteManifestsJSON
	}

	return w(c.stdout, manifests)
}

type ValidateCommand struct {
	GenericCommand
}

// Run implements Command.
func (c *ValidateCommand) Run(args ...string) error {
	setUsageFn(c.flagSet, usageValidate)

	pos, flags := splitArgs(1, args)
	_ = c.flagSet.Parse(flags) // ignore err; the FlagSet uses ExitOnError

	if len(pos) != 1 {
		return fmt.Errorf("unexpected number of positional arguments.\n\n%s", usageValidate(c.flagSet.Name()))
	}
	filePath := pos[0]

	brg, diags := file.NewParser().LoadBridge(filePath)
	if diags.HasErrors() {
		return diags
	}

	ctx, diags := core.NewContext(brg)
	if diags.HasErrors() {
		return diags
	}

	if _, diags := ctx.Generate(); diags.HasErrors() {
		return diags
	}

	return nil
}

type GraphCommand struct {
	GenericCommand
}

// Run implements Command.
func (c *GraphCommand) Run(args ...string) error {
	setUsageFn(c.flagSet, usageGraph)

	pos, flags := splitArgs(1, args)
	_ = c.flagSet.Parse(flags) // ignore err; the FlagSet uses ExitOnError

	if len(pos) != 1 {
		return fmt.Errorf("unexpected number of positional arguments.\n\n%s", usageGraph(c.flagSet.Name()))
	}
	filePath := pos[0]

	brg, diags := file.NewParser().LoadBridge(filePath)
	if diags.HasErrors() {
		return diags
	}

	ctx, diags := core.NewContext(brg)
	if diags.HasErrors() {
		return diags
	}

	g, diags := ctx.Graph()
	if diags.HasErrors() {
		return diags
	}

	dg, err := dot.Marshal(g)
	if err != nil {
		return fmt.Errorf("marshaling graph to DOT: %w", err)
	}

	if _, err := c.stdout.Write(dg); err != nil {
		return fmt.Errorf("writing generated DOT graph: %w", err)
	}

	return nil
}

// splitArgs attempts to separate n positional arguments from the rest of the
// given arguments list. The caller is responsible for ensuring that the
// correct number of positional arguments could be extracted.
//
// It is meant as a helper to implement CLI commands of the shape:
//   cmd ARG1 ARG2 [flags]
func splitArgs(n int, args []string) ( /*positional*/ []string /*flags*/, []string) {
	if len(args) == 0 {
		return nil, nil
	}

	// no positional, or user passed only flags (e.g. "cmd -h")
	if n == 0 || (args[0] != "" && args[0][0] == '-') {
		return nil, args
	}

	if len(args) <= n {
		return args, nil
	}

	return args[:n], args[n:]
}

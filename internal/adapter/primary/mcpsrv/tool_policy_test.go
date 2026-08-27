package mcpsrv

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPToolPolicyInventory(t *testing.T) {
	const wantToolCount = 23
	if got := len(mcpToolPolicies); got != wantToolCount {
		t.Fatalf("tool policy count = %d, want %d", got, wantToolCount)
	}

	wantMutations := []string{
		"mosaic_hexxla_delete_cell",
		"mosaic_hexxla_link_cells",
		"mosaic_hexxla_mark_conflict",
		"mosaic_hexxla_mark_supersedes",
		"mosaic_hexxla_put_cell",
		"mosaic_hexxla_put_embedding",
		"mosaic_hexxla_put_facet",
		"mosaic_hexxla_resolve_seam",
	}
	if got := MutationToolNames(); !slices.Equal(got, wantMutations) {
		t.Fatalf("mutation tools = %v, want %v", got, wantMutations)
	}
}

func TestAddToolRejectsUnclassifiedTool(t *testing.T) {
	server := NewServer("policy-test", "test", nil)
	err := addTool(server, &mcp.Tool{Name: "unclassified"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct{}, error) {
		return nil, struct{}{}, nil
	})
	if err == nil {
		t.Fatal("addTool did not reject an unclassified tool")
	}
}

func TestMCPToolAnnotations(t *testing.T) {
	for name, policy := range mcpToolPolicies {
		server := NewServer("policy-test", "test", nil)
		tool := &mcp.Tool{Name: name}
		err := addTool(server, tool, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct{}, error) {
			return nil, struct{}{}, nil
		})
		if err != nil {
			t.Fatalf("add classified tool %q: %v", name, err)
		}

		annotations := tool.Annotations
		if annotations == nil {
			t.Fatalf("tool %q has no annotations", name)
		}
		if annotations.ReadOnlyHint != policy.readOnly {
			t.Errorf("tool %q readOnlyHint = %t, want %t", name, annotations.ReadOnlyHint, policy.readOnly)
		}
		if annotations.DestructiveHint == nil || *annotations.DestructiveHint != policy.destructive {
			t.Errorf("tool %q destructiveHint = %v, want %t", name, annotations.DestructiveHint, policy.destructive)
		}
		if annotations.IdempotentHint != policy.idempotent {
			t.Errorf("tool %q idempotentHint = %t, want %t", name, annotations.IdempotentHint, policy.idempotent)
		}
		if annotations.OpenWorldHint == nil || *annotations.OpenWorldHint != policy.openWorld {
			t.Errorf("tool %q openWorldHint = %v, want %t", name, annotations.OpenWorldHint, policy.openWorld)
		}
	}
}

func TestProductionRegistrationsUsePolicyWrapper(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	files := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "tool_policy.go" {
			continue
		}
		parsed, err := parser.ParseFile(files, filepath.Clean(name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "AddTool" {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "mcp" {
				t.Errorf("%s uses mcp.AddTool directly; use addTool so safety classification cannot be bypassed", files.Position(call.Pos()))
			}
			return true
		})
	}
}

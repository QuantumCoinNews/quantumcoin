package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

func TestExecutableEntryOnlyDelegatesToNodeApplication(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Imports) != 0 || len(file.Decls) != 1 {
		t.Fatal("main.go must contain only the executable entry, without service dependencies")
	}
	entry := runtimePolicyFunctionV1(t, "main.go", "main")
	if len(entry.Body.List) != 1 {
		t.Fatal("entry must delegate exactly once")
	}
	statement, ok := entry.Body.List[0].(*ast.ExprStmt)
	if !ok {
		t.Fatal("entry is not a direct application call")
	}
	call, ok := statement.X.(*ast.CallExpr)
	if !ok || runtimePolicyCallNameV1(call.Fun) != "runNodeCLI" || len(call.Args) != 0 {
		t.Fatal("entry bypasses canonical command dispatcher")
	}
}

func TestCommandAdaptersCannotConstructIndependentRuntimeServices(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if filepath.Ext(path) != ".go" || len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		aliases := map[string]string{}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			alias := filepath.Base(importPath)
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			aliases[alias] = importPath
		}
		ast.Inspect(file, func(n ast.Node) bool {
			var target ast.Expr
			switch value := n.(type) {
			case *ast.CallExpr:
				target = value.Fun
			case *ast.CompositeLit:
				target = value.Type
			default:
				return true
			}
			selector, ok := target.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			prefix, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			owner := aliases[prefix.Name]
			forbidden := false
			switch owner {
			case "quantumcoin/blockchain":
				forbidden = selector.Sel.Name == "NewBlockchain" || selector.Sel.Name == "LoadBlockchainFromFile"
			case "quantumcoin/node":
				forbidden = selector.Sel.Name == "NewRuntime" || selector.Sel.Name == "Runtime"
			case "quantumcoin/api":
				forbidden = selector.Sel.Name == "StartHTTP"
			case "net/http":
				forbidden = selector.Sel.Name == "ListenAndServe" || selector.Sel.Name == "ListenAndServeTLS" || selector.Sel.Name == "Server"
			case "net":
				forbidden = selector.Sel.Name == "Listen" || selector.Sel.Name == "ListenPacket" || selector.Sel.Name == "ListenTCP"
			case "quantumcoin/p2p":
				forbidden = selector.Sel.Name == "StartServer" || selector.Sel.Name == "NewServerV1"
			}
			if forbidden {
				t.Errorf("%s constructs %s.%s outside canonical node ownership", path, owner, selector.Sel.Name)
			}
			return true
		})
	}
}

func TestStandaloneMinerUsesCanonicalBootstrapBeforeWork(t *testing.T) {
	function := runtimePolicyFunctionV1(t, "miner/adapter_qc_local.go", "NewQCLocalAdapter")
	bootstrap := runtimePolicyFirstCallV1(function, "node.BootstrapWithConfig")
	if bootstrap == token.NoPos {
		t.Fatal("standalone miner bypasses canonical startup policy")
	}
	if snapshot := runtimePolicyFirstCallV1(function, "runtime.Snapshot"); snapshot == token.NoPos || snapshot < bootstrap {
		t.Fatal("standalone miner must obtain its chain from verified runtime bootstrap")
	}
	for _, name := range []string{"blockchain.NewBlockchain", "blockchain.LoadBlockchainFromFile", "chain.BindCanonicalActivationPolicySynchronizedV1"} {
		if runtimePolicyFirstCallV1(function, name) != token.NoPos {
			t.Errorf("standalone miner maintains duplicate startup: %s", name)
		}
	}
}

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func runtimePolicyFunctionV1(
	t *testing.T,
	path string,
	functionName string,
) *ast.FuncDecl {
	t.Helper()

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(
		fileSet,
		path,
		nil,
		parser.SkipObjectResolution,
	)
	if err != nil {
		t.Fatalf(
			"parse %s: %v",
			path,
			err,
		)
	}

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name == nil {
			continue
		}

		if function.Name.Name == functionName {
			return function
		}
	}

	t.Fatalf(
		"function %s not found in %s",
		functionName,
		path,
	)

	return nil
}

func runtimePolicyCallNameV1(
	expression ast.Expr,
) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name

	case *ast.SelectorExpr:
		prefix := runtimePolicyCallNameV1(value.X)
		if prefix == "" {
			return value.Sel.Name
		}

		return prefix + "." + value.Sel.Name

	default:
		return ""
	}
}

func runtimePolicyFirstCallV1(
	function *ast.FuncDecl,
	callName string,
) token.Pos {
	if function == nil || function.Body == nil {
		return token.NoPos
	}

	position := token.NoPos

	ast.Inspect(function.Body, func(node ast.Node) bool {
		if position != token.NoPos {
			return false
		}

		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		if runtimePolicyCallNameV1(call.Fun) ==
			callName {
			position = call.Pos()
			return false
		}

		return true
	})

	return position
}

func TestMainServicesRequireRuntimePolicyBootstrapV1(
	t *testing.T,
) {
	mainFunction := runtimePolicyFunctionV1(
		t,
		"node_app_cli.go",
		"runNodeCLI",
	)

	bootstrapPosition := runtimePolicyFirstCallV1(
		mainFunction,
		"qnode.Bootstrap",
	)
	if bootstrapPosition == token.NoPos {
		t.Fatal(
			"main does not call canonical node bootstrap",
		)
	}

	for _, serviceCall := range []string{
		"startHTTPAPI",
		"startContinuousMining",
		"nodeRuntime.ServeP2P",
	} {
		servicePosition := runtimePolicyFirstCallV1(
			mainFunction,
			serviceCall,
		)
		if servicePosition == token.NoPos {
			t.Fatalf(
				"main service call not found: %s",
				serviceCall,
			)
		}

		if servicePosition < bootstrapPosition {
			t.Fatalf(
				"service %s starts before canonical bootstrap",
				serviceCall,
			)
		}
	}
}

func TestBootstrapPolicyGatePrecedesRuntimeSideEffectsV1(
	t *testing.T,
) {
	wrapper := runtimePolicyFunctionV1(t, "node/bootstrap.go", "bootstrapWithHardforkReleaseV1")
	if runtimePolicyFirstCallV1(wrapper, "bootstrapWithNFTStartupV1") == token.NoPos {
		t.Fatal("hardfork bootstrap bypasses the NFT startup policy boundary")
	}
	bootstrapFunction := runtimePolicyFunctionV1(
		t,
		"node/bootstrap.go",
		"bootstrapWithNFTStartupV1",
	)

	gatePosition := runtimePolicyFirstCallV1(
		bootstrapFunction,
		"validateRuntimePolicyGateWithActivationV1",
	)
	if gatePosition == token.NoPos {
		t.Fatal(
			"NFT startup does not call runtime policy gate",
		)
	}

	for _, sideEffectCall := range []string{
		"internal.SetBonusFile",
		"loadBlockchainWithNFTStartupV1",
	} {
		sideEffectPosition := runtimePolicyFirstCallV1(
			bootstrapFunction,
			sideEffectCall,
		)
		if sideEffectPosition == token.NoPos {
			t.Fatalf(
				"bootstrap side-effect call not found: %s",
				sideEffectCall,
			)
		}

		if sideEffectPosition < gatePosition {
			t.Fatalf(
				"runtime side effect %s occurs before policy gate",
				sideEffectCall,
			)
		}
	}
}

func TestBootstrapDelegatesToCanonicalConfigBoundaryV1(
	t *testing.T,
) {
	bootstrapFunction := runtimePolicyFunctionV1(
		t,
		"node/bootstrap.go",
		"Bootstrap",
	)

	delegationPosition := runtimePolicyFirstCallV1(
		bootstrapFunction,
		"BootstrapWithConfig",
	)
	if delegationPosition == token.NoPos {
		t.Fatal(
			"Bootstrap does not delegate to BootstrapWithConfig",
		)
	}
}

func TestBootstrapUsesOnlyCanonicalHardforkReleaseV1(t *testing.T) {
	f := runtimePolicyFunctionV1(t, "node/bootstrap.go", "BootstrapWithConfig")
	if runtimePolicyFirstCallV1(f, "consensus.LoadCanonicalHardforkReleaseV1") == token.NoPos {
		t.Fatal("public bootstrap lost canonical trust source")
	}
	if runtimePolicyFirstCallV1(f, "bootstrapWithHardforkReleaseV1") == token.NoPos {
		t.Fatal("public bootstrap bypassed verified startup path")
	}
}

func TestBootstrapValidatesChainUnconditionallyBeforePublishingRuntime(t *testing.T) {
	function := runtimePolicyFunctionV1(t, "node/bootstrap.go", "bootstrapWithNFTStartupV1")
	validation := token.NoPos
	for _, statement := range function.Body.List {
		guard, ok := statement.(*ast.IfStmt)
		if !ok || guard.Init == nil {
			continue
		}
		ast.Inspect(guard.Init, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok && runtimePolicyCallNameV1(call.Fun) == "chain.ValidateBlockchainSynchronized" {
				validation = call.Pos()
			}
			return true
		})
	}
	publication := runtimePolicyFirstCallV1(function, "NewRuntime")
	if validation == token.NoPos || publication == token.NoPos || validation > publication {
		t.Fatal("full chain validation must be unconditional and precede runtime construction")
	}
	loader := runtimePolicyFunctionV1(t, "node/bootstrap.go", "loadOrCreateBlockchainWithContextV1")
	if runtimePolicyFirstCallV1(loader, "blockchain.LoadBlockchainFromFileWithLocalContextV1") == token.NoPos || runtimePolicyFirstCallV1(loader, "blockchain.LoadBlockchainFromFile") != token.NoPos {
		t.Fatal("startup must validate persisted history against trusted local context")
	}
}

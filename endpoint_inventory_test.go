package main

import (
	"encoding/csv"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNodeEndpointInventoryCoversCanonicalMux(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "node_app_router.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		pathArg, literal := call.Args[0].(*ast.BasicLit)
		if !ok || selector.Sel.Name != "HandleFunc" || !literal || pathArg.Kind != token.STRING {
			return true
		}
		path, _ := strconv.Unquote(pathArg.Value)
		if !strings.HasPrefix(path, "/api/") || path == "/api/" {
			return true
		}
		protectedRoute := false
		ast.Inspect(call.Args[1], func(node ast.Node) bool {
			if name, ok := node.(*ast.Ident); ok && name.Name == "protected" {
				protectedRoute = true
			}
			return true
		})
		routes[path] = protectedRoute
		return true
	})
	inventory, err := os.Open("docs/node-endpoints.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer inventory.Close()
	rows, err := csv.NewReader(inventory).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows[1:] {
		method, path, auth, state, req, resp := row[0], row[1], row[2], row[3], row[4], row[5]
		protectedRoute, found := routes[path]
		if !found || protectedRoute != strings.HasPrefix(auth, "node-token") {
			t.Errorf("route inventory drift: %s", path)
		}
		if (method != "GET" && method != "POST") || state == "" || req == "" || resp == "" {
			t.Errorf("incomplete route inventory: %s", path)
		}
		delete(routes, path)
	}
	for path := range routes {
		t.Errorf("undocumented node route: %s", path)
	}
}

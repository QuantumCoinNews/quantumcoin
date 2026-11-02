package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalMempoolAuthorityV1(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		normalized := filepath.ToSlash(path)

		if !strings.HasSuffix(normalized, ".go") ||
			strings.HasSuffix(normalized, "_test.go") ||
			strings.HasPrefix(normalized, "internal/") {
			return nil
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		text := string(source)

		if strings.Contains(text, "globalMempool") {
			t.Errorf("production source still references globalMempool: %s", path)
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			return err
		}

		for _, imp := range file.Imports {
			if strings.Trim(imp.Path.Value, `"`) != "quantumcoin/internal" {
				continue
			}

			alias := "internal"
			if imp.Name != nil {
				alias = imp.Name.Name
			}

			ast.Inspect(file, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				ident, ok := selector.X.(*ast.Ident)
				if !ok || ident.Name != alias {
					return true
				}

				if selector.Sel.Name == "Mempool" ||
					selector.Sel.Name == "NewMempool" {
					t.Errorf(
						"production internal mempool selector: %s: %s",
						path,
						selector.Sel.Name,
					)
				}

				return true
			})
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	mainSource, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}

	mainText := string(mainSource)

	if !strings.Contains(mainText, "bc.AddTransaction(tx)") {
		t.Fatal("canonical bc.AddTransaction(tx) yolu yok")
	}

	if !strings.Contains(mainText, "PendingTransactionCountSnapshot()") {
		t.Fatal("canonical pending count metriği yok")
	}

	runtimeSource, err := os.ReadFile("node/runtime.go")
	if err != nil {
		t.Fatal(err)
	}

	runtimeText := string(runtimeSource)

	if strings.Contains(runtimeText, "quantumcoin/internal") ||
		strings.Contains(runtimeText, "Mempool") {
		t.Fatal("node runtime internal mempool sahipliğini sürdürüyor")
	}

	bootstrapSource, err := os.ReadFile("node/bootstrap.go")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(bootstrapSource), "NewMempool") {
		t.Fatal("bootstrap internal.NewMempool çağırıyor")
	}
}

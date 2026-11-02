package main

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestParseBlocksPaginationBoundsRequests(
	t *testing.T,
) {
	tests := []struct {
		name       string
		target     string
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "defaults",
			target:     "/api/blocks",
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "explicit",
			target:     "/api/blocks?limit=50&offset=7",
			wantLimit:  50,
			wantOffset: 7,
		},
		{
			name:       "hard cap",
			target:     "/api/blocks?limit=500000&offset=2",
			wantLimit:  200,
			wantOffset: 2,
		},
		{
			name:       "invalid values preserve safe defaults",
			target:     "/api/blocks?limit=bad&offset=bad",
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "non positive limit and negative offset",
			target:     "/api/blocks?limit=-10&offset=-5",
			wantLimit:  20,
			wantOffset: 0,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				request :=
					httptest.NewRequest(
						"GET",
						test.target,
						nil,
					)

				limit, offset :=
					parseBlocksPagination(
						request,
					)

				if limit != test.wantLimit {
					t.Fatalf(
						"limit=%d want=%d",
						limit,
						test.wantLimit,
					)
				}

				if offset != test.wantOffset {
					t.Fatalf(
						"offset=%d want=%d",
						offset,
						test.wantOffset,
					)
				}
			},
		)
	}
}

func TestTransactionStatusHandlerUsesIndexedLookup(
	t *testing.T,
) {
	sourceBytes, err :=
		os.ReadFile(
			"main.go",
		)

	if err != nil {
		t.Fatalf(
			"read main.go: %v",
			err,
		)
	}

	source := string(sourceBytes)

	handler :=
		extractRootFunctionForScalingTest(
			t,
			source,
			"handleTxStatus",
		)

	if !strings.Contains(
		handler,
		"TransactionLocationSnapshot(",
	) {
		t.Fatal(
			"tx status indexed location lookup missing",
		)
	}

	for _, forbidden := range []string{
		"AllBlocksSnapshot(",
		"PendingTransactionsSnapshot(",
		"for _, block :=",
		"for _, tx :=",
	} {
		if strings.Contains(
			handler,
			forbidden,
		) {
			t.Fatalf(
				"tx status still contains full-scan fragment %q",
				forbidden,
			)
		}
	}
}

func TestBlocksListHandlerUsesBoundedCoherentPage(
	t *testing.T,
) {
	sourceBytes, err :=
		os.ReadFile(
			"main.go",
		)

	if err != nil {
		t.Fatalf(
			"read main.go: %v",
			err,
		)
	}

	source := string(sourceBytes)

	handler :=
		extractRootFunctionForScalingTest(
			t,
			source,
			"handleBlocksList",
		)

	for _, required := range []string{
		"parseBlocksPagination(",
		"BlocksPageFromTipSnapshot(",
		`"X-Total-Count"`,
		`"X-Page-Limit"`,
		`"X-Page-Offset"`,
		`"X-Has-More"`,
		"writeOK(w, summaries)",
	} {
		if !strings.Contains(
			handler,
			required,
		) {
			t.Fatalf(
				"blocks handler missing %q",
				required,
			)
		}
	}

	if strings.Contains(
		handler,
		"BlocksRangeSnapshot(",
	) {
		t.Fatal(
			"blocks handler still uses split count/range snapshot path",
		)
	}
}

func extractRootFunctionForScalingTest(
	t *testing.T,
	source string,
	name string,
) string {
	t.Helper()

	startMarker :=
		"func " + name + "("

	start :=
		strings.Index(
			source,
			startMarker,
		)

	if start < 0 {
		t.Fatalf(
			"function %s not found",
			name,
		)
	}

	rest := source[start:]

	next :=
		strings.Index(
			rest[1:],
			"\nfunc ",
		)

	if next < 0 {
		return rest
	}

	return rest[:next+1]
}

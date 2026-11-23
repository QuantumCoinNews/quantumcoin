package node

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestHTTPResponseRegressionNFTMessage(t *testing.T) {
	var r *Runtime
	for _, mint := range []bool{false, true} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/nft/monthly/mint", nil)
		if mint {
			r.ServeNFTMonthlyMintV1(w, req)
		} else {
			r.ServeNFTMonthlyReceiptV1(w, req)
		}
		var failure struct {
			Success                *bool `json:"success"`
			Message, Code, Version string
		}
		if w.Code != 405 || json.Unmarshal(w.Body.Bytes(), &failure) != nil || failure.Success == nil || *failure.Success || failure.Message == "" || failure.Code != "method_not_allowed" || failure.Version != NFTMonthlyMintReceiptVersionV1 {
			t.Errorf("NFT error lacks message or lost code/version: %d %s", w.Code, w.Body.String())
		}
	}
}

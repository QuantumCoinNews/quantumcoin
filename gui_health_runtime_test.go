package main

import (
	"context"
	"net/http/httptest"
	"quantumcoin/apiclient"
	qnode "quantumcoin/node"
	"testing"
)

func TestGUIHealthAgainstCanonicalRootMux(t *testing.T) {
	handler := readOnlyMuxFixtureV1(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	endpoint := server.URL + apiclient.CanonicalHealthPathV1
	state, err := apiclient.ProbeHealth(context.Background(), endpoint, actualMuxTestTokenV1)
	if err != nil || !state.Reachable || !state.Authenticated {
		t.Fatalf("authenticated root health: %+v %v", state, err)
	}
	for _, token := range []string{"", "wrong-token"} {
		state, err = apiclient.ProbeHealth(context.Background(), endpoint, token)
		if err != nil || !state.Reachable || state.Authenticated {
			t.Fatalf("protected root health: %+v %v", state, err)
		}
	}
	previous := nodeRuntime
	nodeRuntime = qnode.NewRuntime(nil, cfg)
	defer func() { nodeRuntime = previous }()
	state, err = apiclient.ProbeHealth(context.Background(), endpoint, actualMuxTestTokenV1)
	if err == nil || state.Reachable {
		t.Fatalf("nil chain reported ready: %+v %v", state, err)
	}
}

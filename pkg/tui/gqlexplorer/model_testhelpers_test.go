package gqlexplorer

import (
	"testing"
	"time"

	"github.com/xaaha/hulak/pkg/tui"
)

func waitForMouseZone(t *testing.T, id string) (int, int) {
	return waitForMouseZoneMinHeight(t, id, 0)
}

func waitForMouseZoneMinHeight(t *testing.T, id string, minHeight int) (int, int) {
	t.Helper()
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		startX, startY, _, endY, ok := tui.ZoneBounds(id)
		if ok && endY-startY >= minHeight {
			return startX, startY
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("zone %q was not registered", id)
	return 0, 0
}

func sampleOps() []UnifiedOperation {
	return []UnifiedOperation{
		{Name: "getUser", Type: TypeQuery, Description: "fetch user", Endpoint: "http://api/gql"},
		{Name: "listUsers", Type: TypeQuery, Endpoint: "http://api/gql"},
		{Name: "createUser", Type: TypeMutation, Endpoint: "http://api/gql"},
		{Name: "deleteUser", Type: TypeMutation, Endpoint: "http://api/gql"},
		{
			Name:        "onMessage",
			Type:        TypeSubscription,
			Description: "new messages",
			Endpoint:    "http://api/gql",
		},
	}
}

func multiEndpointOps() []UnifiedOperation {
	return []UnifiedOperation{
		{Name: "getUser", Type: TypeQuery, Endpoint: "https://api.spacex.com/graphql"},
		{Name: "listRockets", Type: TypeQuery, Endpoint: "https://api.spacex.com/graphql"},
		{
			Name:     "getCountry",
			Type:     TypeQuery,
			Endpoint: "https://countries.trevorblades.com/graphql",
		},
		{Name: "createPost", Type: TypeMutation, Endpoint: "https://api.spacex.com/graphql"},
		{
			Name:     "updateCountry",
			Type:     TypeMutation,
			Endpoint: "https://countries.trevorblades.com/graphql",
		},
	}
}

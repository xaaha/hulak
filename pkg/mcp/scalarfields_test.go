package mcp

import (
	"context"
	"path/filepath"
	"testing"
)

func TestListRequests_278_SurvivesNonTextFields(t *testing.T) {
	root := projectDir(t)
	writeReq(t, root, "ok.hk.yaml")
	writeFileAt(t, filepath.Join(root, "kind.hk.yaml"), "kind: 5\nmethod: GET\nurl: http://x\n")
	writeFileAt(t, filepath.Join(root, "url.hk.yaml"), "method: GET\nurl: 8080\n")
	s, err := NewServer(map[string]string{"api": root}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Detail: true})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, r := range out.Projects[0].Requests {
		names[r.Name] = r.Kind
	}
	if len(names) != 3 || names["ok"] != "API" || names["kind"] != "5" {
		t.Errorf("requests = %v, want ok (API), kind (5) and url listed", names)
	}
}

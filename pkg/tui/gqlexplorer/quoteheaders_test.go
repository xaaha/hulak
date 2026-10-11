package gqlexplorer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

func TestCreateHulakRequestFile_278_D10_SavedHeadersRunLikeTheParent(t *testing.T) {
	parent := "kind: GraphQL\nurl: https://api.example.com/gql\nheaders:\n" +
		"  X-At: \"@me\"\n" +
		"  X-Tick: \"`x`\"\n" +
		"  Accept: \"*/*\"\n" +
		"  X-Hash: \"v #frag\"\n" +
		"  X-Zero: \"007\"\n" +
		"  X-Colon: \"a: b\"\n" +
		"  X-Tab: \"a\\tb\"\n" +
		"  X-Quote: 'say \"hi\"'\n" +
		"  X-Lines: \"one\\ntwo\"\n" +
		"  X-Null: \"null\"\n" +
		"  X-Tilde: \"~\"\n" +
		"  X-Dash: \"- a\"\n" +
		"  X-Bang: \"!x\"\n" +
		"  X-True: \"true\"\n" +
		"  X-List: \"[a]\"\n" +
		"  X-Anchor: \"&a\"\n" +
		"  X-Pipe: \"|\"\n" +
		"  X-Pct: \"%x\"\n" +
		"  X-Q: \"? x\"\n" +
		"  X-Space: \" lead\"\n" +
		"  \"@key\": v\n"
	_, dir := saveFromParent(t, parent)
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	want, err := graphql.ProcessGraphQLFile(filepath.Join(dir, "parent.hk.yaml"), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := graphql.ProcessGraphQLFile(filepath.Join(dir, "getUser.hk.yaml"), map[string]any{})
	if err != nil {
		saved, _ := os.ReadFile(filepath.Join(dir, "getUser.hk.yaml"))
		t.Fatalf("saved request does not run: %v\n%s", err, saved)
	}
	if !reflect.DeepEqual(got.Headers, want.Headers) {
		t.Errorf("saved headers = %#v, want the parent's %#v", got.Headers, want.Headers)
	}
}

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"

	"github.com/xaaha/hulak/pkg/envparser"
)

func schemaServer(t *testing.T, root string) (*Server, []byte) {
	t.Helper()
	schema, err := os.ReadFile("../../assets/schema.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(map[string]string{"api": root}, "v")
	if err != nil {
		t.Fatal(err)
	}
	s.SetRequestSchema(schema)
	return s, schema
}

func guideYAMLBlocks(t *testing.T) []string {
	t.Helper()
	matches := regexp.MustCompile("(?s)```yaml\n(.*?)```").FindAllStringSubmatch(authoringGuide, -1)
	if len(matches) == 0 {
		t.Fatal("the authoring guide holds no yaml examples")
	}
	blocks := make([]string, len(matches))
	for i, m := range matches {
		blocks[i] = m[1]
	}
	return blocks
}

func TestRequestFormat_260_GuideAndSchema(t *testing.T) {
	s, schema := schemaServer(t, projectDir(t))
	ctx := context.Background()

	_, out, err := s.handleRequestFormat(ctx, nil, requestFormatInput{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Guide != authoringGuide || out.Schema != "" {
		t.Errorf("default call: guide matches = %v, schema length = %d, want the guide and no schema",
			out.Guide == authoringGuide, len(out.Schema))
	}

	_, out, err = s.handleRequestFormat(ctx, nil, requestFormatInput{Schema: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.Schema != string(schema) || out.Guide != authoringGuide {
		t.Error("schema: true must return the guide and the configured schema unchanged")
	}

	bare, err := NewServer(map[string]string{"api": projectDir(t)}, "v")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := bare.handleRequestFormat(ctx, nil, requestFormatInput{Schema: true}); err == nil {
		t.Error("schema: true without a configured schema must be an error")
	}
}

func TestRequestFormat_260_GuideExamplesMatchTheSchema(t *testing.T) {
	s, raw := schemaServer(t, projectDir(t))
	var schema struct {
		Properties map[string]json.RawMessage
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	for i, block := range guideYAMLBlocks(t) {
		if err := s.validateRequestContent(block); err != nil {
			t.Errorf("guide example %d is not a valid request: %v\n%s", i+1, err, block)
		}
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(block), &doc); err != nil {
			t.Fatal(err)
		}
		for key := range doc {
			if _, ok := schema.Properties[key]; !ok {
				t.Errorf("guide example %d uses %q, which the schema does not define", i+1, key)
			}
		}
	}
}

func TestRequestFormat_260_GuideTokenChainResolves(t *testing.T) {
	root := projectDir(t)
	writeReq(t, filepath.Join(root, "auth"), "get_token.hk.yaml")
	writeFileAt(t, filepath.Join(root, "auth", "get_token.hk_response.json"), `{"access_token": "tok-123"}`)
	s, _ := schemaServer(t, root)

	var header string
	for _, block := range guideYAMLBlocks(t) {
		if strings.Contains(block, "getValueOf") {
			var doc struct {
				Headers map[string]string `yaml:"headers"`
			}
			if err := yaml.Unmarshal([]byte(block), &doc); err != nil {
				t.Fatal(err)
			}
			header = doc.Headers["Authorization"]
		}
	}
	if header == "" {
		t.Fatal("the guide has no getValueOf Authorization example")
	}

	var got any
	err := s.withProjectDir(root, func() error {
		var err error
		got, err = envparser.SubstituteVariables(header, map[string]any{}, "")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Bearer tok-123" {
		t.Errorf("the guide's token chain %q resolves to %v, want Bearer tok-123", header, got)
	}
}

func TestRequestFormat_260_GuideCoversTheSchema(t *testing.T) {
	raw, err := os.ReadFile("../../assets/schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties struct {
			Kind   struct{ Enum []string }
			Method struct{ Enum []string }
			Body   struct {
				OneOf []struct {
					Properties map[string]json.RawMessage
				}
			}
		}
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	var topLevel struct {
		Properties map[string]json.RawMessage
	}
	if err := json.Unmarshal(raw, &topLevel); err != nil {
		t.Fatal(err)
	}

	var names []string
	for name := range topLevel.Properties {
		names = append(names, name)
	}
	names = append(names, schema.Properties.Method.Enum...)
	for _, alt := range schema.Properties.Body.OneOf {
		for name := range alt.Properties {
			names = append(names, name)
		}
	}
	lowerGuide := strings.ToLower(authoringGuide)
	for _, kind := range schema.Properties.Kind.Enum {
		if !strings.Contains(lowerGuide, "`"+strings.ToLower(kind)+"`") {
			t.Errorf("the guide does not name kind %s", kind)
		}
	}
	for _, name := range names {
		if !strings.Contains(authoringGuide, "`"+name+"`") {
			t.Errorf("the guide does not name %s from the schema", name)
		}
	}
}

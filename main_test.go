package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"strings"
	"testing"
)

func TestCatalogJSONFixtures(t *testing.T) {
	for _, tc := range []struct {
		file  string
		value any
	}{
		{"services.json", new([]Service)},
		{"directory.json", new(DirectoryList)},
	} {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, tc.value); err != nil {
				t.Fatalf("parse existing catalog: %v", err)
			}
			switch v := tc.value.(type) {
			case *[]Service:
				if len(*v) == 0 || (*v)[0].Name == "" {
					t.Fatal("service catalog is empty or missing names")
				}
			case *DirectoryList:
				if len(v.Items) == 0 || v.Items[0].ID == "" {
					t.Fatal("API directory is empty or missing IDs")
				}
			}
		})
	}
}

func TestCatalogJSONStrictParsing(t *testing.T) {
	for _, input := range []string{
		`{"name":"first","name":"second"}`,
		"{\"name\":\"\xff\"}",
	} {
		var service Service
		if err := json.Unmarshal([]byte(input), &service); err == nil {
			t.Errorf("accepted invalid catalog JSON %q", input)
		}
	}

	var service Service
	if err := json.Unmarshal([]byte(`{"Name":"wrong case","name":"correct case"}`), &service); err != nil {
		t.Fatal(err)
	}
	if service.Name != "correct case" {
		t.Fatalf("name = %q, want correct case", service.Name)
	}
	var wrongCase Service
	if err := json.Unmarshal([]byte(`{"Name":"wrong case"}`), &wrongCase); err != nil || wrongCase.Name != "" {
		t.Fatalf("case-insensitive name matched: %+v, %v", wrongCase, err)
	}
}

func TestCatalogJSONOutput(t *testing.T) {
	services := []map[string]any{{"title": "Example", "name": "example.googleapis.com"}}
	data, err := json.Marshal(services, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n  {\n    \"name\": \"example.googleapis.com\",\n    \"title\": \"Example\"\n  }\n]"
	if string(data) != want {
		t.Fatalf("catalog output = %s, want %s", data, want)
	}

	data, err = json.Marshal(DirectoryList{Items: []APIEntry{{ID: "example"}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"labels"`) || !strings.Contains(string(data), `"preferred":false`) {
		t.Fatalf("unexpected directory output: %s", data)
	}
	data, err = json.Marshal(DirectoryList{})
	if err != nil || !strings.Contains(string(data), `"items":[]`) {
		t.Fatalf("nil items should encode as an empty array: %s, %v", data, err)
	}
}

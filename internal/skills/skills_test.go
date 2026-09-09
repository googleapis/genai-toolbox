// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package skills_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/skills"
)

func TestManifestMarshalJSON(t *testing.T) {
	tcs := []struct {
		desc string
		in   skills.Manifest
		want string
	}{
		{
			desc: "static skill lists its files",
			in: skills.Manifest{Refs: []skills.ResourceRef{
				{URI: "skill://analytics-guide/SKILL.md", Digest: "sha256:a1b2", Size: 2314},
				{URI: "skill://analytics-guide/references/queries.md", Digest: "sha256:c3d4", Size: 962},
			}},
			want: `[{"uri":"skill://analytics-guide/SKILL.md","digest":"sha256:a1b2","size":2314},` +
				`{"uri":"skill://analytics-guide/references/queries.md","digest":"sha256:c3d4","size":962}]`,
		},
		{
			desc: "dynamic skill collapses to the marker",
			in:   skills.Manifest{Dynamic: true},
			want: `"dynamic"`,
		},
		{
			desc: "dynamic wins over any refs left set",
			in:   skills.Manifest{Dynamic: true, Refs: []skills.ResourceRef{{URI: "skill://x/SKILL.md"}}},
			want: `"dynamic"`,
		},
		{
			desc: "unpopulated static manifest stays an array",
			in:   skills.Manifest{},
			want: `[]`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal() = %v, want nil", err)
			}
			if string(got) != tc.want {
				t.Errorf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestManifestUnmarshalJSON(t *testing.T) {
	tcs := []struct {
		desc    string
		in      string
		want    skills.Manifest
		wantErr string
	}{
		{
			desc: "file list",
			in:   `[{"uri":"skill://x/SKILL.md","digest":"sha256:a1b2","size":10}]`,
			want: skills.Manifest{Refs: []skills.ResourceRef{
				{URI: "skill://x/SKILL.md", Digest: "sha256:a1b2", Size: 10},
			}},
		},
		{
			desc: "dynamic marker",
			in:   `"dynamic"`,
			want: skills.Manifest{Dynamic: true},
		},
		{
			desc:    "any other string is not a third form",
			in:      `"static"`,
			wantErr: `only permitted string is "dynamic"`,
		},
		{
			desc:    "an object is not a manifest",
			in:      `{"refs":[]}`,
			wantErr: "must be an array",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			var got skills.Manifest
			err := json.Unmarshal([]byte(tc.in), &got)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Unmarshal() = nil, want error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Unmarshal() = %v, want error containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal() = %v, want nil", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestEntryMarshalJSON pins the shape SEP-2640 specifies for a skills/list
// entry: the URI addresses SKILL.md rather than the skill root, and frontmatter
// passes through verbatim because a host compares it field by field.
func TestEntryMarshalJSON(t *testing.T) {
	e := skills.Entry{
		URI: "skill://analytics-guide/SKILL.md",
		Frontmatter: map[string]any{
			"name":        "analytics-guide",
			"description": "Query and summarize the warehouse",
		},
		Resources: skills.Manifest{Refs: []skills.ResourceRef{
			{URI: "skill://analytics-guide/SKILL.md", Digest: "sha256:a1b2", Size: 2314},
		}},
	}

	got, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal() = %v, want nil", err)
	}
	want := `{"uri":"skill://analytics-guide/SKILL.md",` +
		`"frontmatter":{"description":"Query and summarize the warehouse","name":"analytics-guide"},` +
		`"resources":[{"uri":"skill://analytics-guide/SKILL.md","digest":"sha256:a1b2","size":2314}]}`
	if string(got) != want {
		t.Errorf("Marshal() =\n%s\nwant\n%s", got, want)
	}
}

func TestEntryRoundTripsDynamic(t *testing.T) {
	in := skills.Entry{
		URI:         "skill://drafting/SKILL.md",
		Frontmatter: map[string]any{"name": "drafting"},
		Resources:   skills.Manifest{Dynamic: true},
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal() = %v, want nil", err)
	}
	if !strings.Contains(string(data), `"resources":"dynamic"`) {
		t.Fatalf("Marshal() = %s, want resources to be the dynamic marker", data)
	}

	var got skills.Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() = %v, want nil", err)
	}
	if diff := cmp.Diff(in, got); diff != "" {
		t.Errorf("round trip mismatch (-want +got):\n%s", diff)
	}
}

// TestEntryMatchesSEPExample round-trips the worked example from SEP-2640's
// "Retrieval via skills/get" section. Byte-identical re-marshalling is the
// check that these types are wire-compatible with the spec rather than with
// our reading of it, and it catches a renamed or dropped field.
func TestEntryMatchesSEPExample(t *testing.T) {
	const sepExample = `{"uri":"skill://pdf-processing/SKILL.md",` +
		`"frontmatter":{"description":"Extract, fill, and assemble PDF documents","metadata":{"version":"2.1.0"},"name":"pdf-processing"},` +
		`"resources":[` +
		`{"uri":"skill://pdf-processing/SKILL.md","digest":"sha256:d5e6f7a8...","size":5120},` +
		`{"uri":"skill://pdf-processing/references/FORMS.md","digest":"sha256:e6f7a8b9...","size":18433},` +
		`{"uri":"skill://pdf-processing/scripts/extract.py","digest":"sha256:f7a8b9c0...","size":4096}]}`

	var e skills.Entry
	if err := json.Unmarshal([]byte(sepExample), &e); err != nil {
		t.Fatalf("Unmarshal() = %v, want nil", err)
	}
	if e.URI != "skill://pdf-processing/SKILL.md" {
		t.Errorf("URI = %q, want the SKILL.md URI", e.URI)
	}
	if got := len(e.Resources.Refs); got != 3 {
		t.Errorf("len(Refs) = %d, want 3", got)
	}
	if e.Resources.Dynamic {
		t.Error("Dynamic = true, want false for a manifest carrying a file list")
	}

	got, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal() = %v, want nil", err)
	}
	if string(got) != sepExample {
		t.Errorf("round trip differs from the SEP example:\n got %s\nwant %s", got, sepExample)
	}
}

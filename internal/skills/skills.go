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

// Package skills implements the io.modelcontextprotocol/skills extension
// (SEP-2640). A skill is a directory of files served over the Resources
// primitive: SKILL.md at its root, addressed as skill://<skill-path>/SKILL.md,
// plus any supporting files sharing that prefix.
package skills

import (
	"encoding/json"
	"fmt"
)

// DynamicMarker is what a skill publishes in place of a file list when its
// content is generated and cannot carry stable digests.
const DynamicMarker = "dynamic"

// ResourceRef is one file in a skill's manifest. Every field is required:
// size lets a host budget a skill before fetching anything, and a read whose
// length disagrees with it fails verification before any hashing.
type ResourceRef struct {
	URI    string `json:"uri"`
	Digest string `json:"digest"` // "sha256:" followed by 64 lowercase hex characters
	Size   int64  `json:"size"`   // raw byte length of the content the digest covers
}

// Manifest is a skill's complete file list, or the marker saying it has none.
// SEP-2640 admits exactly those two forms and no third, so the union is held in
// a struct that cannot express anything else and is collapsed to the wire shape
// at the marshalling boundary. A host reads a missing digest as "unverifiable",
// which is why a partially populated list is not a legal state.
type Manifest struct {
	// Refs lists every file of the skill, SKILL.md included. Ignored when
	// Dynamic is set.
	Refs []ResourceRef
	// Dynamic marks a skill whose content is generated, so no digest can be
	// published for it.
	Dynamic bool
}

// MarshalJSON emits the file list, or the string "dynamic".
func (m Manifest) MarshalJSON() ([]byte, error) {
	if m.Dynamic {
		return json.Marshal(DynamicMarker)
	}
	// A static skill always lists at least SKILL.md, so an empty Refs means the
	// manifest was never populated rather than that the skill has no files.
	// Emitting [] keeps the wire type an array either way.
	if len(m.Refs) == 0 {
		return json.Marshal([]ResourceRef{})
	}
	return json.Marshal(m.Refs)
}

// UnmarshalJSON accepts either form and rejects anything else, so a malformed
// entry fails here rather than surfacing as an empty file list.
//
// The type switch is load-bearing: encoding/json unmarshals a JSON null into
// both a string and a slice without error, leaving each at its zero value. So
// null would otherwise be reported as an empty-string marker, or worse, be
// accepted as an empty file list — and SEP-2640 requires an entry with no
// resources to be rejected outright.
func (m *Manifest) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid skill manifest: %w", err)
	}

	switch v := raw.(type) {
	case string:
		if v != DynamicMarker {
			return fmt.Errorf("invalid skill manifest %q: the only permitted string is %q", v, DynamicMarker)
		}
		m.Dynamic, m.Refs = true, nil
		return nil

	case []any:
		var refs []ResourceRef
		if err := json.Unmarshal(data, &refs); err != nil {
			return fmt.Errorf("invalid skill manifest: %w", err)
		}
		m.Dynamic, m.Refs = false, refs
		return nil

	default:
		return fmt.Errorf("invalid skill manifest: must be an array of {uri, digest, size} or the string %q", DynamicMarker)
	}
}

// Entry is one skill as skills/list and skills/get publish it.
type Entry struct {
	// URI addresses the skill's SKILL.md, not its root directory.
	URI string `json:"uri"`
	// Frontmatter is the SKILL.md YAML frontmatter verbatim. A host compares it
	// field by field against the file it fetches, so it must not be normalised.
	Frontmatter map[string]any `json:"frontmatter"`
	Resources   Manifest       `json:"resources"`
}

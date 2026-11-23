// Copyright 2026 PolitePixels Limited
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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package generate

import (
	"fmt"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// artefactPageTemplate formats a generated-artefact page from its label text, given
	// twice: once for the output and once for the page title.
	artefactPageTemplate = "<template><main id=\"fuzz-program\"><output data-fuzz-output=\"label\">%s</output></main></template>\n" +
		"<script type=\"application/x-go\">\npackage main\nimport \"piko.sh/piko\"\n" +
		"func Render(r *piko.RequestData, p piko.NoProps)(piko.NoResponse,piko.Metadata,error){return piko.NoResponse{},piko.Metadata{Title:%q},nil}\n</script>"
)

// addArtefacts adds source and expected observations for generated-artefact checks.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addArtefacts(c *casefile.Case, p Project) {
	label := fmt.Sprintf("Artifact %d", p.Count)
	for _, key := range []string{"artifact", "unaffected"} {
		text := key + " " + label
		c.Files["pages/"+key+".pk"] = fmt.Sprintf(artefactPageTemplate, text, text)
		c.Probes = append(c.Probes, programProbe("/"+key, programOutput("label", text)))
	}
	c.Files["actions/contract/echo.go"] = `package contract

import "piko.sh/piko"

type EchoAction struct{piko.ActionMetadata}
type EchoInput struct { Label string ` + "\x60json:\"label\"\x60" + ` }
type EchoResponse struct { Label string ` + "\x60json:\"label\"\x60" + ` }
func (a EchoAction) Call(in EchoInput)(EchoResponse,error){return EchoResponse{Label:in.Label},nil}
`
	c.Features = append(c.Features, "generated-go-registration", "manifest-regeneration", "typescript-action-exports", "runtime-symbols")
}

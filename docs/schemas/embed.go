// Package schemas supplies the exact published simulation input resources to
// the CLI. Embedding keeps validation offline and avoids a second schema copy.
package schemas

import "embed"

//go:embed openudon.simulate-input.v1.schema.json openudon.step-authoring.v1.schema.json
var SimulationInputResources embed.FS

// BrowserCaptureResources is the published, offline capture wire and reviewed-start schemas.
//
//go:embed openudon.browser-capture.v1.schema.json openudon.browser-capture-start.v1.schema.json
var BrowserCaptureResources embed.FS

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

// CatalogSourceResources is the exact additive confirmed-source request schema.
//
//go:embed openudon.step-source-catalog.v1.schema.json
var CatalogSourceResources embed.FS

// BrokerHandoffResources contains the additive, value-free Stage 9 schemas.
//
//go:embed openudon.broker-authority.v1.schema.json openudon.approval.v2.schema.json openudon.executor-run.v3.schema.json openudon.run-evidence.v4.schema.json
var BrokerHandoffResources embed.FS

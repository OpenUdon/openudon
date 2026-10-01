// Package schemas supplies the exact published simulation input resources to
// the CLI. Embedding keeps validation offline and avoids a second schema copy.
package schemas

import "embed"

//go:embed openudon.simulate-input.v1.schema.json openudon.step-authoring.v1.schema.json
var SimulationInputResources embed.FS

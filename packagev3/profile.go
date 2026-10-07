package packagev3

import (
	"encoding/json"
	"github.com/OpenUdon/openudon/wire"
	"github.com/OpenUdon/uws/runtimes"
)

func decodeProfile(data []byte, out *runtimes.OperationRuntime) error {
	if wire.DecodeStrictNumbers(data, out) != nil {
		return ErrPackage
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil || fields == nil {
		return ErrPackage
	}
	for key := range fields {
		switch key {
		case "type", "function", "arguments":
		default:
			return ErrPackage
		}
	}
	return nil
}

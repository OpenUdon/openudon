// Package brokerhandoff retains the legacy internal adapter over public authority.
package brokerhandoff

import "github.com/OpenUdon/openudon/authority"

const Version = authority.Version
const TransportVersion = authority.TransportVersion
const MaxOperations = authority.MaxOperations

type Authority = authority.Authority
type Operation = authority.Operation
type Binding = authority.Binding

func Identifier(value string) bool { return authority.Identifier(value) }

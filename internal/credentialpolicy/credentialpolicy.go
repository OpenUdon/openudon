// Package credentialpolicy preserves the legacy internal policy adapter.
package credentialpolicy

import policy "github.com/OpenUdon/openudon/credentialpolicy"

func ContainsLikelyValue(data []byte) bool  { return policy.ContainsLikelyValue(data) }
func IsSymbolicReference(value string) bool { return policy.IsSymbolicReference(value) }
func IsLikelyLiteral(value string) bool     { return policy.IsLikelyLiteral(value) }
func SafeMappingValue(value string) bool    { return policy.SafeMappingValue(value) }

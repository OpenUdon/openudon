#!/usr/bin/env bash
set -euo pipefail

blocked_imports='github\.com/OpenUdon/apitools/(llm|icot|context7)'
blocked_root_symbols='apitools\.(Artifact(Set)?|Assumption|Binding(Contract|Field|Ref)|BuildBindingContract|BuildReviewPackage|ChatClient|CompleteJSONWithFallback|ComputeReviewHandoffDigest|ContainsLikelyCredentialValue|Documentation(Context|Snippet)|Draft|Flow|Interactive|JSONCompletion|Leaf(Adapter|Options)|NewLeafAdapter|Question(Plan)?|Review(Handoff|State|Package|OwnerSplit|ExecutionPolicy|CredentialBindings|TrustedRunner)|Slot|SymbolicBinding|Transcript|ValidateReviewHandoff)'

if rg -n --glob '*.go' --glob '!*_test.go' "$blocked_imports" .; then
  echo "non-OpenAPI apitools package import found" >&2
  exit 1
fi

if rg -n --glob '*.go' --glob '!*_test.go' "$blocked_root_symbols" .; then
  echo "non-OpenAPI root apitools lifecycle symbol found" >&2
  exit 1
fi

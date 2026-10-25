# Ordinary authoring-card consumer qualification

These are the exact reusable standalone consumer inputs used for P14 publication
qualification. Copy the four .txt files to their un-suffixed names in a disposable
module. The Go module is pinned, with no workspace or replacements. Use cached
Go1.26.6, GOWORK=off, GOPROXY=off, GOSUMDB=off, GOAUTH=off and
GOTOOLCHAIN=local. P14_MODULE_DIR points to the exact ordinarily acquired module
Dir, solely to read its hash-bound conformance inputs, never to replace imports.
Run go test -json ./... -count=1, go vet ./..., and go build -o probe .
Run probe with that module's docs/fixtures/authoring-card-v1/complete.json.
The test executes all33 shared positive/refusal cases and exact schema/text
checks; the probe actually parses/validates/renders one typed card and returns
runtime BuildInfo. No card grants approval or operation authority. See
../../p14-published-qualification.json for original receipts and limitations.

# Independently produced runtime catalog fixture

Source: published private Udon M48 da43e57be37af4e18e633558580f740c525f139d,
module v0.0.0-20261007023819-da43e57be37a, cached Go 1.26.6, GOWORK=off,
GOPROXY=off, no directory replacements. RuntimeFunctionCatalog is independently
checked by VerifyRuntimeFunctionCatalog before writing these fixtures. The
fixture generator is /var/tmp/openudon-p09-runtime-fixtures-20261007; it copies
the qualified ordinary M48 consumer module closure without changing it.

catalog.json: 3,468 bytes; SHA-256
67933ec02e3b8808ef32637272295908661a39555c13f9f0bfb057af54097812.
shapes.json: 4,189 bytes; SHA-256
7b8c176eb07315ca076a282887c4f9a54cda810f49a8820eed21bde303bde35e.

Seven runtime-owned pure function entries retain partial native invocation/type
metadata. No function, provider, credential, random source or target is called
during production/verification. Default public SDK tests use exact independent
fixture comparisons; final ordinary published private-adapter qualification is
P09.5. No private runtime import enters OpenUdon's public/default build.

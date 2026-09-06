# Supervised application control

`icot control --protocol openudon.application-control.v1 --no-open` opens the
same single-workspace application as the UI over private stdin/stdout pipes.
Use the existing example, private-root and package scope/scratch/store options.
The default remains `openudon.registration-control.v1` for older supervisors.
Neither mode opens a listener or a controller browser.

The supervisor sends bounded NDJSON frames containing `version`, `operation`
and the operation's optional `request`. The application first emits a state,
then one state after each completed request. Each response carries `version`,
`application` (the existing API-v4 review resources), and optional fixed
`failure` and question-identity fields. This channel is private operator state, not a report or log.
Requests retain the exact revision and confirmation fields of the corresponding
UI operation. Unknown or duplicate fields are rejected.

| Operation | Existing request/behavior |
| --- | --- |
| `snapshot` | No request; refresh workspace and immutable package state. |
| `registration.start`, `registration.command`, `registration.cancel` | Registration API-v4 requests and closed observe/navigate/draft/review/finish union. |
| `browser.preflight` | Authoring and capture revisions; check installed Chromium readiness. |
| `capture.start`, `capture.respond`, `capture.cancel`, `capture.stage` | Authenticated capture requests and typed human checkpoint responses. |
| `author.journey`, `author.round`, `author.reopen` | Journey selection and existing structured authoring decisions. |
| `author.approve`, `author.resume`, `package.build` | Existing separate authoring approval, recovery and build confirmation. |
| `transaction.start`, `transaction.review`, `transaction.prepare`, `transaction.promote`, `transaction.cancel` | Existing transaction engine requests with exact revision/digest authority. |
| `transaction.inspect_recovery`, `transaction.recover`, `transaction.inspect_selected` | Existing exact generation inspection and digest-confirmed recovery requests. |
| `cancel`, `close` | No request; interrupt the current operation and end the application. |

Send one request and wait for its response before sending another. `close`,
`cancel`, EOF and context cancellation can interrupt an in-flight operation;
other pipelined commands terminate with `protocol` instead of being replayed.
An operation must join within 15 seconds after cancellation. The application
retains its nonrenewable 20-minute deadline and separate worker teardown bound.
The external supervisor must own and terminate the process tree if the
application cannot prove operation or worker teardown. A failed teardown never
permits another browser-sensitive operation in that application.

Registration candidates stay in memory after worker teardown. Transaction
review adopts that exact candidate into the ordinary authoring engine; package
build must pass before preparation and promotion. When the package lifecycle
is configured, authenticated capture retains its
independently reconstructed pair as a private candidate and exposes its
transaction for separate review. Review adopts both virtual sources and their
shared session semantics; ordinary authoring approval writes the canonical
profiles. Without package lifecycle configuration, the existing standalone
capture staging path remains available. Restart never reconstructs private candidate bytes from a public snapshot.

Real approvals remain operator decisions. The supervisor may display the
existing review resources and transmit an exact decision, but cannot infer an
approval from progress, elapsed time, a fixture response or a previous attempt.
Authenticated authoring keeps credential and verification entry in the headed
browser. Registration authoring allows observation only. The application
protocol has no runtime execution, credential-value, raw result, browser-state,
selector, script or arbitrary HTTP route operation. General execution continues
through selected packages and the existing OpenUdon/Udon/Browserdriver handoff.
Navigation/request policies are application enforcement, not network-wide
containment.

The synthetic qualification exercises the real shipped command, private pipes,
Browsertools workers, review and package promotion. Its automated checkpoint
responses and input-filling JavaScript exist only in loopback fixtures. Default
unit tests remain browser-free.

The separate trusted execution command accepts opt-in `openudon run
--interactive-browser` to forward a private human response stream through
the internal or external runner to Udon's existing verification interaction.
This requires a reviewed browser workflow and keeps the default noninteractive.
It supplies no answer, approval, credential or challenge decision automatically.
Responses are not serialized into run configuration or invocation evidence.

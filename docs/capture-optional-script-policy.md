# Private capture optional-script policy

Authenticated application control can select one reviewed denial rule with
`--capture-blocked-script-origin https://analytics.example.test`. It passes
immutable local configuration to Browsertools through the isolated worker's
`--blocked-script-origin`. The exact origin remains blocked; only its
non-navigation GET/Script denial may leave the session running.

Defaults remain strict. The application and browser protocol schemas, revisions,
private diagnostic readers, BAP/BCP sources and runtime execution policy are
unchanged. Request messages cannot set this field. Registration control and the
HTTP UI reject the local capture option. Configuration cannot overlap an approved
origin, and the worker blocks later attempts to admit it.

The consumer must bind the selected local option to its reviewed scope and exact
executable/source evidence. Origin selection grants no login POST approval and
no account action. Credentials still stay in the headed protected workflow.

The supervised_authenticated_package qualification stage includes an optional
script from a separately owned HTTPS loopback endpoint, requires zero endpoint
TCP arrivals, and completes synthetic authentication and profile/package review.
This exercises actual application-to-worker propagation. Generic guard/OOPIF
fixtures live in Browsertools. Full qualification and exact passing-byte adoption
remain required before changed-runtime use.

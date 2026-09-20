# E14 — Registration foreground and deadline integration

| Item | State | Notes |
| --- | --- | --- |
| E14.1 Bind and test the generic runtime successor | `[+]` | Adopt Browserdriver M12/Udon M40 exact source revisions; require a visible countdown at private input/human/submit checkpoints in the synthetic UI-to-runtime fixture. Public BRP/UWS and private-input separation remain unchanged. |
| E14.2 Verify and review the frozen integration | `[+]` | Full owner checks and fresh smoke pass. W16 acceptance v2 passes 39 native stages and nine W8M receipts; 20 source bindings/eight runtime hashes independently verified and exact tested bytes adopted. Bounded review iteration 1 has no P1/P2. |

The owner approved implementation, focused synthetic verification and runtime
qualification/adoption. W8M W16 owns its acceptance and consumed attempt
history. No live target contact or registration follows this engineering work.
Complete. Review iteration 1 has no open P1/P2. E13 and A29 remain completed history.

Owner fast checks, full Go tests, vet, make check and diff checks pass.
The first smoke stopped at the source gate before browser execution because
the Workspace Browsertools HEAD is a later revision than its exact lock.
Prepare exact local snapshots before the fresh integration smoke; do not move
the working sibling or relax the pin. E14.1 task review iteration 1 found no
P1/P2 in the additive fixture and lock integration. E14.2 review was pending at that source-freeze checkpoint.

Fresh exact-source registration smoke passes in 98.629 seconds. The first
native offline run stopped before acceptance/browser qualification because
the private launcher inherited an added umask 077: two permission-refusal
fixtures requested public modes which that umask silently made private.
Both failures reproduce with 077 and pass with the standard 022 umask.
Restore the standard launcher environment, preserve the failed offline report,
and run a fresh offline report plus acceptance v2 on the unchanged snapshots.
Reports and runtime-private files enforce their own restrictive modes.

The corrected fresh native offline report and its verifier pass. The complete
W8M acceptance v2 aggregate and native verifier pass, with 39 native stages
and nine W8M workflow receipts. All 20 clean source bindings and eight identical
runtime/module hashes across three retained passes match independent
recomputation. Pass one was adopted without rebuilding; its actual OpenUdon
binary verifies the retained native component. W15 and all real consumed
attempts remain unchanged; no live operation is armed.

The frozen integration is OpenUdon `f31ba1c94dec1b2a5a81f39cf0533c93059e3931`,
Browserdriver `9d13e8b35394b35c4e7fc17162336ac4e67be878`, Udon
`5ef6af99430cc246b354accd5840801534f0eb81`, and W8M
`e46ccd5da1162912cc15f96edf655443657fb0a8`. Source commits are published to
their existing origins; Browsertools/UWS/build-input pins are unchanged.
The full v2 aggregate took 63.460 minutes, including 47.103 minutes for native
qualification and 5.416, 5.376 and 5.456 minutes for its fresh consumer journeys.
Aggregate SHA-256: `b27e8d1bca72c784b42e0844aed29b0001a0b65dd03631b52aab2b256e8a5966`.

Final bounded review iteration 1 covered additive v5 timing, earliest deadline
selection, expiry enforcement, foreground failure handling, private form and
package separation, qualification freshness, exact source/runtime binding and
joined teardown. No P1/P2 remains. The E14 index/scope is reconciled; no evolution
version changes because the approved public/private direction is unchanged.
W8M W16 owns future desktop visibility, identity readiness and separate exact
live authority; its previous registration remains consumed and unsuccessful.

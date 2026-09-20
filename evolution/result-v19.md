# Typed MFA And Dashboard Output Authoring Result

OpenUdon `6a08b81317a852b9b8581c502eadbbdf591508e1` now requires
`browsertools.author-session.v2`, the `reviewed_mfa_kind` and
`reviewed_outputs` capabilities, and `maxOutputs: 16`. Credential and MFA
checkpoints complete through distinct `human_input_complete` messages. The
human chooses exactly one advertised MFA kind, and the model has no output or
MFA selection action.

Completion reviews zero through 16 safe current-observation outputs with
string, integer, number, Boolean, or presence semantics and exact-name or
unique-role locator modes. Returned selections, challenge traces, credential
slots, contexts, profile discriminators, reviews, digests, and full profile
contents are independently validated; expected profiles are deterministically
reconstructed so digest-updated substitution still fails. Browser 1.7 selects
UWS 1.9 while browser 1.5/1.6 retain UWS 1.7/1.8 selection.

| Layer | Published revision/version |
|---|---|
| UWS | `dd9eb32105131bdbc2855090ae0639b22d12de2b`; `v0.0.0-20260817013720-dd9eb3210513`; UWS 1.9/browser 1.7 |
| Browserdriver | `4df3c0f83f66cf9fc15f81170d31d355c9e341c2`; protocol v3 for browser 1.7 |
| Udon | `f1fac6713396f4fda33547ea3fcb296166c7f63b` (M32 implementation `d4d47f1c59d78091717ba01135db4f9effec7fe1`); UWS 1.9 lowering and v3 replay |
| Browsertools | `dd89956d02203a5c02aa4a7d13ac1e4fe040da05`; `v0.0.0-20260817022912-dd89956d0220`; author-session/result v2 |
| OpenUdon | `6a08b81317a852b9b8581c502eadbbdf591508e1`; strict v2 client/import and UWS selection |

The clean-revision matrix passed 11 required gates with no failure; the three
unrequested installed/headed opt-ins were skipped. Default evidence remained
browser-free, credential-free, provider-free, and network-free apart from
publication and pseudo-version resolution.

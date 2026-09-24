# UWS 1.11 adoption state

At OpenUdon HEAD `57e516fd06abc823a1eee8eb909c6b3bfe1c9144`, generation
chooses an oldest-sufficient UWS version up to 1.9.1 and Browser 1.8/1.9
profiles are rejected by local allowlists and the trusted runner. UWS
`e9b6181be0abb7f683fdb624d4dba282a59991d1` is published; Browsertools,
Browserdriver and Udon have separate adoption work. The three expired
registration fixture tests are an independent baseline failure.

[M84](../memory-bank/status-M84.md) records the remaining OpenUdon rows and
their dependency gates. This snapshot does not claim implementation or live
execution.

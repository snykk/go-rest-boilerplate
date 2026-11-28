// Package archtest checks the architecture's import rules on every build of the tests: a module reaches
// another only through that module's contract package (internal/modules/<name>/contract, its published data,
// events and errors), layers depend inwards, and platform knows no module. It holds tests only.
package archtest

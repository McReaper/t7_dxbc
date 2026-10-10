# Working on t7dxbc

How to change this repository without breaking what it guarantees. What it is, how to use it and the test commands are in `README.md`; the decompiler's design is in `go doc ./hlsl`.

## Layout

- `dxbc/` reads the format: the container, reflection, signatures, instruction decoding, and a disassembly identical to Microsoft's.
- `hlsl/` decompiles, and reads `dxbc` only through its exported API.
- `internal/oracle` and `internal/sim` are the two oracles; `internal/d3dc` loads Microsoft's compiler; `internal/genfixtures` builds the fixtures; `internal/fixture` finds them from any package's tests.
- `testdata/` holds the fixtures every package shares: `<name>.hlsl` (the source, its first line `// dxbc: <entry> <target>`), `.dxbc` and `.asm` (built by `go run ./internal/genfixtures`), `.decompiled.hlsl` (the golden decompilation).

## The oracles judge a change, not the golden files

A decompiler change is right when both oracles accept it on Windows: `internal/oracle` (the recompiled shader keeps the interface an engine binds) and `internal/sim` (the original and the recompiled shader compute the same on random inputs). They run on the fixtures, on random shaders (`TestRandom*`), and, for a change that alters what the game's shaders decompile to, on its shader cache. Only then record the goldens with `go test ./hlsl -run TestDecompileGolden -update`: a golden file says the output changed, never whether it is right.

## Every bug gets a test that failed without the fix

A fixture (HLSL written for the case, compiled by genfixtures) or a case in the random generators, checked to fail on the code before the fix; the commit body says it did.

## No game content in the repository

Never commit a shader from the game, its disassembly, its decompilation, or a file derived from them. Fixtures are HLSL written for this repository. Corpus runs read the cache from the install (`DXBC_CORPUS`) and keep their journals outside the repository.

## Controls are decoded in one place

An instruction's control bits (saturate, test polarity, offsets, return types, sync flags…) are read only through the methods in `dxbc/control.go`, which the Microsoft-checked disassembler uses too. The simulator and the decompiler never decode bits themselves: a misreading both shared would make the oracles agree on a wrong answer.

## Rechecking the game's cache

A full run takes hours, and recompiling a large compute shader takes up to 2.3 GB of memory, so set `DXBC_WORKERS`. After a decompiler change, decompile every shader of the cache with the old and the new code (pure Go, under a minute) and run the oracles only on those whose output changed: an unchanged decompilation keeps its result.

## The compiler's own faults

Don't chase these; the tests already allow for them:

- fxc flushes denormals and drops -0 in folded float constants: bit patterns travel in `uint4` banks.
- It reorders float operations by an ulp: the simulator compares floats with a tolerance.
- A NaN's sign follows the operands' order: either default NaN is one result.
- It fails with an internal error on about 1 random program in 10,000. A generated shader it fails on is skipped; a decompilation it fails on is counted, and fails the random tests past 1%.
- It miscompiled one correct decompilation at every optimization level.

## Mutation testing new code

Generate mutants with go-mutesting (`/no-exec`; under Git Bash, set `MSYS_NO_PATHCONV=1`) and run the tests against each with `DXBC_NO_GOLDEN=1`, as a golden file would kill every mutant whether the oracles notice or not. A mutant that survives gets a test that kills it, or a reason why it changes nothing observable.

## Fuzzing

`FuzzDecompile` and the `dxbc` fuzzers (`FuzzParse`, `FuzzDecodeProgram`, `FuzzReadReflection`, `FuzzReadSignature`) take any bytes: an error, never a panic or a hang. A failing input becomes a named seed.

## Code and commits

- Run `git config core.hooksPath .githooks` once: the pre-commit hook runs gofmt and golangci-lint.
- `.golangci.yml` caps a function at 80 lines, 60 statements, cyclomatic 20, cognitive 30, and a file at 500 code lines. Split by what the code does rather than raising a limit.
- Commits are conventional (`fix:`, `feat:`, `test(sim):`, `refactor:`, `docs:`): the subject says what changed, the body why and how it was verified.
- Docs state rules, not their history: how something was found belongs in the commit message.

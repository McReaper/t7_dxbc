// Package t7dxbc reads compiled Direct3D 10/11 shaders (DXBC, Shader Model 4
// and 5) and writes them back as HLSL.
//
// A DXBC file is a container of chunks: RDEF (resources and constant
// buffers, with their names), ISGN/OSGN and their variants (input and output
// signatures), SHDR/SHEX (the program, a stream of 32-bit tokens) and others
// this package ignores (statistics, debug info). Parse reads the container;
// the chunk readers check every offset against the chunk they read, so a
// truncated or hostile file is an error, never a panic.
//
// # Disassembly
//
// Program.Disassemble writes the instructions as Microsoft's disassembler
// does, number formats included, and an instruction's controls (_sat, _nz,
// offsets, return types) are decoded once, by methods on Instruction that
// everything else reads them through.
//
// # Decompilation
//
// Decompile writes HLSL for a pixel, vertex, compute or geometry shader that
// compiles back to the same interface: the constant buffers with every
// variable at its offset, the resources under their reflected names, a
// structured buffer with its element struct, and the signatures packed into
// the registers they had (the compiler packs elements by interpolation mode,
// which a vertex shader's outputs don't record, so the modes are chosen to
// reproduce the packing; an element alone in a register though an earlier
// one had room is declared as an array, whose elements each open a
// register). Each instruction becomes one or more statements.
//
// DXBC registers hold bits; HLSL variables have a type, and the compiler
// treats a float variable as a float, flushing a small integer's bits (a
// denormal) to zero. So a temporary component that ever receives an
// integer lives in a uint4 bank, i#, and the others in a float4 bank, r#; a
// float stored in i# keeps its bits. Inputs and outputs are read and written
// as their parameters' own types. What HLSL can't say directly goes through
// a helper the output carries: bit fields, firstbit_hi (which counts from the
// top bit, firstbithigh from the bottom), and a structured buffer's dword at
// an offset computed at run time, found by testing the offset against the
// element's layout.
//
// Not decompiled: hull and domain shaders, a geometry shader's streams other
// than 0, and the high half of imul and umul. One difference remains in what
// recompiles: a constant buffer the original declares but never reads is not
// declared again, so the recompiled shader has no binding for it. The
// compiler itself rarely crashes on, or miscompiles, a correct decompilation;
// nothing in the HLSL can avoid it.
//
// # Testing
//
// The decompiler is judged on Windows by two oracles at once
// (internal/oracle, then internal/sim): the HLSL is compiled again with
// Microsoft's compiler and must keep the interface, its declarations and its
// barriers; and the original and the recompiled program run in the simulator
// on the same random inputs and must agree. They judge the fixtures, random
// shaders the compiler builds from generated HLSL (pixel, compute, and
// signatures packed by random interpolation modes and arrays), and, opt in, a
// whole shader cache (TestCorpusAgrees).
package t7dxbc

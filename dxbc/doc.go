// Package dxbc reads compiled Direct3D 10/11 shaders (DXBC, Shader Model 4
// and 5) and disassembles them.
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
// does, number formats included; TestCorpusDisassembly checks it against
// Microsoft's on a whole shader cache. An instruction's controls (_sat, _nz,
// offsets, return types) are decoded once, by methods on Instruction that
// everything else reads them through.
package dxbc

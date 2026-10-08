// Package dxbc reads compiled Direct3D 10/11 shaders (DXBC, Shader Model 4
// and 5) and writes them back as HLSL.
//
// A DXBC file is a container of chunks: RDEF (resources and constant
// buffers, with their names), ISGN/OSGN and their variants (input and output
// signatures), SHDR/SHEX (the program, a stream of 32-bit tokens) and others
// this package ignores (statistics, debug info). Parse reads the container;
// the chunk readers check every offset against the chunk they read, so a
// truncated or hostile file is an error, never a panic.
package dxbc

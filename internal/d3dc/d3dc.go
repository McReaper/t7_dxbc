//go:build windows

// Package d3dc calls the HLSL compiler that ships with Windows
// (d3dcompiler_47.dll) without cgo, for the tests: it compiles the fixtures and
// disassembles shaders so the decoder can be checked against Microsoft's own
// output. The module itself never needs it.
package d3dc

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	dll         = syscall.NewLazyDLL("d3dcompiler_47.dll")
	compile     = dll.NewProc("D3DCompile")
	disassemble = dll.NewProc("D3DDisassemble")
)

// standardFileInclude is D3D_COMPILE_STANDARD_FILE_INCLUDE: #include resolves
// relative to the source file's own path.
const standardFileInclude = 1

// Available reports whether the compiler DLL can be loaded.
func Available() bool { return dll.Load() == nil }

// Compile compiles HLSL source for one entry point and target profile
// ("ps_5_0", "vs_5_0", "cs_5_0"). path names the source for #include and for
// the compiler's messages.
func Compile(src []byte, path, entry, target string) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("empty source")
	}
	cpath, err := cstr(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	centry, err := cstr(entry)
	if err != nil {
		return nil, err
	}
	ctarget, err := cstr(target)
	if err != nil {
		return nil, err
	}
	var code, msgs unsafe.Pointer
	hr, _, _ := compile.Call(uintptr(unsafe.Pointer(&src[0])), uintptr(len(src)), uintptr(unsafe.Pointer(cpath)), 0,
		standardFileInclude, uintptr(unsafe.Pointer(centry)), uintptr(unsafe.Pointer(ctarget)), 0, 0,
		uintptr(unsafe.Pointer(&code)), uintptr(unsafe.Pointer(&msgs)))
	text := ""
	if msgs != nil {
		text = string(blobBytes(msgs))
		release(msgs)
	}
	if hr != 0 {
		if code != nil {
			release(code)
		}
		return nil, fmt.Errorf("D3DCompile %#x: %s", uint32(hr), text)
	}
	out := append([]byte(nil), blobBytes(code)...)
	release(code)
	return out, nil
}

// Disassemble returns Microsoft's disassembly of a compiled shader.
func Disassemble(bytecode []byte) (string, error) {
	if len(bytecode) == 0 {
		return "", errors.New("empty bytecode")
	}
	var blob unsafe.Pointer
	hr, _, _ := disassemble.Call(uintptr(unsafe.Pointer(&bytecode[0])), uintptr(len(bytecode)), 0, 0, uintptr(unsafe.Pointer(&blob)))
	if hr != 0 {
		return "", fmt.Errorf("D3DDisassemble %#x", uint32(hr))
	}
	text := string(blobBytes(blob))
	release(blob)
	return text, nil
}

func cstr(s string) (*byte, error) { return syscall.BytePtrFromString(s) }

// An ID3DBlob is a COM object: QueryInterface, AddRef, Release,
// GetBufferPointer, GetBufferSize.
func vtable(blob unsafe.Pointer) *[5]uintptr { return (*[5]uintptr)(*(*unsafe.Pointer)(blob)) }

func blobBytes(blob unsafe.Pointer) []byte {
	v := vtable(blob)
	ptr, _, _ := syscall.SyscallN(v[3], uintptr(blob))
	n, _, _ := syscall.SyscallN(v[4], uintptr(blob))
	if ptr == 0 || n == 0 {
		return nil
	}
	// the blob owns the buffer at ptr: reinterpret the address rather than
	// convert it, which vet rightly flags for addresses Go allocated
	return unsafe.Slice(*(**byte)(unsafe.Pointer(&ptr)), int(n))
}

// release drops the blob; its return value is the remaining reference count.
func release(blob unsafe.Pointer) { _, _, _ = syscall.SyscallN(vtable(blob)[2], uintptr(blob)) }

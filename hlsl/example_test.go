package hlsl_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/McReaper/t7_dxbc/hlsl"
)

// A blur's vertex shader comes back with its outputs as the original
// declared them: four taps in an array, each in a register of its own.
func ExampleDecompile() {
	code, err := os.ReadFile(filepath.Join("..", "testdata", "vs_taps.dxbc"))
	if err != nil {
		log.Fatal(err)
	}
	src, err := hlsl.Decompile(code)
	if err != nil {
		log.Fatal(err)
	}
	entry := src[strings.Index(src, "void vs_main("):]
	fmt.Println(entry[:strings.Index(entry, ")")+1])
	// Output:
	// void vs_main(float4 in_POSITION0 : POSITION0,
	//   float2 in_TEXCOORD0 : TEXCOORD0,
	//   out float4 out_SV_Position0 : SV_Position0,
	//   out float2 out_TEXCOORD0[4] : TEXCOORD0,
	//   out float out_TEXCOORD4 : TEXCOORD4)
}

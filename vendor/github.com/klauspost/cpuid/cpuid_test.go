// Copyright (c) 2015 Klaus Post, released under MIT License. See LICENSE file.

package cpuid

import (
	"fmt"
	"testing"
)

// There is no real way to test a CPU identifier, since results will
// obviously differ on each machine.
func TestCPUID(t *testing.T) {
	n := maxFunctionID()
	t.Logf("Max Function:0x%x\n", n)
	n = maxExtendedFunction()
	t.Logf("Max Extended Function:0x%x\n", n)
	t.Log("Name:", CPU.BrandName)
	t.Log("PhysicalCores:", CPU.PhysicalCores)
	t.Log("ThreadsPerCore:", CPU.ThreadsPerCore)
	t.Log("LogicalCores:", CPU.LogicalCores)
	t.Log("Family", CPU.Family, "Model:", CPU.Model)
	t.Log("Features:", CPU.Features)
	t.Log("Cacheline bytes:", CPU.CacheLine)
	t.Log("L1 Instruction Cache:", CPU.Cache.L1I, "bytes")
	t.Log("L1 Data Cache:", CPU.Cache.L1D, "bytes")
	t.Log("L2 Cache:", CPU.Cache.L2, "bytes")
	t.Log("L3 Cache:", CPU.Cache.L3, "bytes")

	if CPU.SSE2() {
		t.Log("We have SSE2")
	}
}

func TestDumpCPUID(t *testing.T) {
	n := int(maxFunctionID())
	for i := 0; i <= n; i++ {
		a, b, c, d := cpuidex(uint32(i), 0)
		t.Logf("CPUID %08x: %08x-%08x-%08x-%08x", i, a, b, c, d)
		ex := uint32(1)
		for {
			a2, b2, c2, d2 := cpuidex(uint32(i), ex)
			if a2 == a && b2 == b && d2 == d || ex > 50 || a2 == 0 {
				break
			}
			t.Logf("CPUID %08x: %08x-%08x-%08x-%08x", i, a2, b2, c2, d2)
			a, b, c, d = a2, b2, c2, d2
			ex++
		}
	}
	n2 := maxExtendedFunction()
	for i := uint32(0x80000000); i <= n2; i++ {
		a, b, c, d := cpuid(i)
		t.Logf("CPUID %08x: %08x-%08x-%08x-%08x", i, a, b, c, d)
	}
}

func Example() {
	// Print basic CPU information:
	fmt.Println("Name:", CPU.BrandName)
	fmt.Println("PhysicalCores:", CPU.PhysicalCores)
	fmt.Println("ThreadsPerCore:", CPU.ThreadsPerCore)
	fmt.Println("LogicalCores:", CPU.LogicalCores)
	fmt.Println("Family", CPU.Family, "Model:", CPU.Model)
	fmt.Println("Features:", CPU.Features)
	fmt.Println("Cacheline bytes:", CPU.CacheLine)

	// Test if we have a specific feature:
	if CPU.SSE() {
		fmt.Println("We have Streaming SIMD Extensions")
	}
}

func TestBrandNameZero(t *testing.T) {
	if len(CPU.BrandName) > 0 {
		// Cut out last byte
		last := []byte(CPU.BrandName[len(CPU.BrandName)-1:])
		if last[0] == 0 {
			t.Fatal("last byte was zero")
		} else if last[0] == 32 {
			t.Fatal("whitespace wasn't trimmed")
		}
	}
}

// Generated here: http://play.golang.org/p/mko-0tFt0Q

// TestCmov tests Cmov() function
func TestCmov(t *testing.T) {
	got := CPU.Cmov()
	expected := CPU.Features&CMOV == CMOV
	if got != expected {
		t.Fatalf("Cmov: expected %v, got %v", expected, got)
	}
	t.Log("CMOV Support:", got)
}

// TestAmd3dnow tests Amd3dnow() function
func TestAmd3dnow(t *testing.T) {
	got := CPU.Amd3dnow()
	expected := CPU.Features&AMD3DNOW == AMD3DNOW
	if got != expected {
		t.Fatalf("Amd3dnow: expected %v, got %v", expected, got)
	}
	t.Log("AMD3DNOW Support:", got)
}

// TestAmd3dnowExt tests Amd3dnowExt() function
func TestAmd3dnowExt(t *testing.T) {
	got := CPU.Amd3dnowExt()
	expected := CPU.Features&AMD3DNOWEXT == AMD3DNOWEXT
	if got != expected {
		t.Fatalf("Amd3dnowExt: expected %v, got %v", expected, got)
	}
	t.Log("AMD3DNOWEXT Support:", got)
}

// TestMMX tests MMX() function
func TestMMX(t *testing.T) {
	got := CPU.MMX()
	expected := CPU.Features&MMX == MMX
	if got != expected {
		t.Fatalf("MMX: expected %v, got %v", expected, got)
	}
	t.Log("MMX Support:", got)
}

// TestMMXext tests MMXext() function
func TestMMXext(t *testing.T) {
	got := CPU.MMXExt()
	expected := CPU.Features&MMXEXT == MMXEXT
	if got != expected {
		t.Fatalf("MMXExt: expected %v, got %v", expected, got)
	}
	t.Log("MMXEXT Support:", got)
}

// TestSSE tests SSE() function
func TestSSE(t *testing.T) {
	got := CPU.SSE()
	expected := CPU.Features&SSE == SSE
	if got != expected {
		t.Fatalf("SSE: expected %v, got %v", expected, got)
	}
	t.Log("SSE Support:", got)
}

// TestSSE2 tests SSE2() function
func TestSSE2(t *testing.T) {
	got := CPU.SSE2()
	expected := CPU.Features&SSE2 == SSE2
	if got != expected {
		t.Fatalf("SSE2: expected %v, got %v", expected, got)
	}
	t.Log("SSE2 Support:", got)
}

// TestSSE3 tests SSE3() function
func TestSSE3(t *testing.T) {
	got := CPU.SSE3()
	expected := CPU.Features&SSE3 == SSE3
	if got != expected {
		t.Fatalf("SSE3: expected %v, got %v", expected, got)
	}
	t.Log("SSE3 Support:", got)
}

// TestSSSE3 tests SSSE3() function
func TestSSSE3(t *testing.T) {
	got := CPU.SSSE3()
	expected := CPU.Features&SSSE3 == SSSE3
	if got != expected {
		t.Fatalf("SSSE3: expected %v, got %v", expected, got)
	}
	t.Log("SSSE3 Support:", got)
}

// TestSSE4 tests SSE4() function
func TestSSE4(t *testing.T) {
	got := CPU.SSE4()
	expected := CPU.Features&SSE4 == SSE4
	if got != expected {
		t.Fatalf("SSE4: expected %v, got %v", expected, got)
	}
	t.Log("SSE4 Support:", got)
}

// TestSSE42 tests SSE42() function
func TestSSE42(t *testing.T) {
	got := CPU.SSE42()
	expected := CPU.Features&SSE42 == SSE42
	if got != expected {
		t.Fatalf("SSE42: expected %v, got %v", expected, got)
	}
	t.Log("SSE42 Support:", got)
}

// TestAVX tests AVX() function
func TestAVX(t *testing.T) {
	got := CPU.AVX()
	expected := CPU.Features&AVX == AVX
	if got != expected {
		t.Fatalf("AVX: expected %v, got %v", expected, got)
	}
	t.Log("AVX Support:", got)
}

// TestAVX2 tests AVX2() function
func TestAVX2(t *testing.T) {
	got := CPU.AVX2()
	expected := CPU.Features&AVX2 == AVX2
	if got != expected {
		t.Fatalf("AVX2: expected %v, got %v", expected, got)
	}
	t.Log("AVX2 Support:", got)
}

// TestFMA3 tests FMA3() function
func TestFMA3(t *testing.T) {
	got := CPU.FMA3()
	expected := CPU.Features&FMA3 == FMA3
	if got != expected {
		t.Fatalf("FMA3: expected %v, got %v", expected, got)
	}
	t.Log("FMA3 Support:", got)
}

// TestFMA4 tests FMA4() function
func TestFMA4(t *testing.T) {
	got := CPU.FMA4()
	expected := CPU.Features&FMA4 == FMA4
	if got != expected {
		t.Fatalf("FMA4: expected %v, got %v", expected, got)
	}
	t.Log("FMA4 Support:", got)
}

// TestXOP tests XOP() function
func TestXOP(t *testing.T) {
	got := CPU.XOP()
	expected := CPU.Features&XOP == XOP
	if got != expected {
		t.Fatalf("XOP: expected %v, got %v", expected, got)
	}
	t.Log("XOP Support:", got)
}

// TestF16C tests F16C() function
func TestF16C(t *testing.T) {
	got := CPU.F16C()
	expected := CPU.Features&F16C == F16C
	if got != expected {
		t.Fatalf("F16C: expected %v, got %v", expected, got)
	}
	t.Log("F16C Support:", got)
}

// TestCX16 tests CX16() function
func TestCX16(t *testing.T) {
	got := CPU.CX16()
	expected := CPU.Features&CX16 == CX16
	if got != expected {
		t.Fatalf("CX16: expected %v, got %v", expected, got)
	}
	t.Log("CX16 Support:", got)
}

// TestSGX tests SGX() function
func TestSGX(t *testing.T) {
	got := CPU.SGX.Available
	expected := CPU.Features&SGX == SGX
	if got != expected {
		t.Fatalf("SGX: expected %v, got %v", expected, got)
	}
	t.Log("SGX Support:", got)
}

// TestBMI1 tests BMI1() function
func TestBMI1(t *testing.T) {
	got := CPU.BMI1()
	expected := CPU.Features&BMI1 == BMI1
	if got != expected {
		t.Fatalf("BMI1: expected %v, got %v", expected, got)
	}
	t.Log("BMI1 Support:", got)
}

// TestBMI2 tests BMI2() function
func TestBMI2(t *testing.T) {
	got := CPU.BMI2()
	expected := CPU.Features&BMI2 == BMI2
	if got != expected {
		t.Fatalf("BMI2: expected %v, got %v", expected, got)
	}
	t.Log("BMI2 Support:", got)
}

// TestTBM tests TBM() function
func TestTBM(t *testing.T) {
	got := CPU.TBM()
	expected := CPU.Features&TBM == TBM
	if got != expected {
		t.Fatalf("TBM: expected %v, got %v", expected, got)
	}
	t.Log("TBM Support:", got)
}

// TestLzcnt tests Lzcnt() function
func TestLzcnt(t *testing.T) {
	got := CPU.Lzcnt()
	expected := CPU.Features&LZCNT == LZCNT
	if got != expected {
		t.Fatalf("Lzcnt: expected %v, got %v", expected, got)
	}
	t.Log("LZCNT Support:", got)
}

// TestLzcnt tests Lzcnt() function
func TestPopcnt(t *testing.T) {
	got := CPU.Popcnt()
	expected := CPU.Features&POPCNT == POPCNT
	if got != expected {
		t.Fatalf("Popcnt: expected %v, got %v", expected, got)
	}
	t.Log("POPCNT Support:", got)
}

// TestAesNi tests AesNi() function
func TestAesNi(t *testing.T) {
	got := CPU.AesNi()
	expected := CPU.Features&AESNI == AESNI
	if got != expected {
		t.Fatalf("AesNi: expected %v, got %v", expected, got)
	}
	t.Log("AESNI Support:", got)
}

// TestHTT tests HTT() function
func TestHTT(t *testing.T) {
	got := CPU.HTT()
	expected := CPU.Features&HTT == HTT
	if got != expected {
		t.Fatalf("HTT: expected %v, got %v", expected, got)
	}
	t.Log("HTT Support:", got)
}

// TestClmul tests Clmul() function
func TestClmul(t *testing.T) {
	got := CPU.Clmul()
	expected := CPU.Features&CLMUL == CLMUL
	if got != expected {
		t.Fatalf("Clmul: expected %v, got %v", expected, got)
	}
	t.Log("CLMUL Support:", got)
}

// TestSSE2Slow tests SSE2Slow() function
func TestSSE2Slow(t *testing.T) {
	got := CPU.SSE2Slow()
	expected := CPU.Features&SSE2SLOW == SSE2SLOW
	if got != expected {
		t.Fatalf("SSE2Slow: expected %v, got %v", expected, got)
	}
	t.Log("SSE2SLOW Support:", got)
}

// TestSSE3Slow tests SSE3slow() function
func TestSSE3Slow(t *testing.T) {
	got := CPU.SSE3Slow()
	expected := CPU.Features&SSE3SLOW == SSE3SLOW
	if got != expected {
		t.Fatalf("SSE3slow: expected %v, got %v", expected, got)
	}
	t.Log("SSE3SLOW Support:", got)
}

// TestAtom tests Atom() function
func TestAtom(t *testing.T) {
	got := CPU.Atom()
	expected := CPU.Features&ATOM == ATOM
	if got != expected {
		t.Fatalf("Atom: expected %v, got %v", expected, got)
	}
	t.Log("ATOM Support:", got)
}

// TestNX tests NX() function (NX (No-Execute) bit)
func TestNX(t *testing.T) {
	got := CPU.NX()
	expected := CPU.Features&NX == NX
	if got != expected {
		t.Fatalf("NX: expected %v, got %v", expected, got)
	}
	t.Log("NX Support:", got)
}

// TestSSE4A tests SSE4A() function (AMD Barcelona microarchitecture SSE4a instructions)
func TestSSE4A(t *testing.T) {
	got := CPU.SSE4A()
	expected := CPU.Features&SSE4A == SSE4A
	if got != expected {
		t.Fatalf("SSE4A: expected %v, got %v", expected, got)
	}
	t.Log("SSE4A Support:", got)
}

// TestHLE tests HLE() function (Hardware Lock Elision)
func TestHLE(t *testing.T) {
	got := CPU.HLE()
	expected := CPU.Features&HLE == HLE
	if got != expected {
		t.Fatalf("HLE: expected %v, got %v", expected, got)
	}
	t.Log("HLE Support:", got)
}

// TestRTM tests RTM() function (Restricted Transactional Memory)
func TestRTM(t *testing.T) {
	got := CPU.RTM()
	expected := CPU.Features&RTM == RTM
	if got != expected {
		t.Fatalf("RTM: expected %v, got %v", expected, got)
	}
	t.Log("RTM Support:", got)
}

// TestRdrand tests RDRAND() function (RDRAND instruction is available)
func TestRdrand(t *testing.T) {
	got := CPU.Rdrand()
	expected := CPU.Features&RDRAND == RDRAND
	if got != expected {
		t.Fatalf("Rdrand: expected %v, got %v", expected, got)
	}
	t.Log("Rdrand Support:", got)
}

// TestRdseed tests RDSEED() function (RDSEED instruction is available)
func TestRdseed(t *testing.T) {
	got := CPU.Rdseed()
	expected := CPU.Features&RDSEED == RDSEED
	if got != expected {
		t.Fatalf("Rdseed: expected %v, got %v", expected, got)
	}
	t.Log("Rdseed Support:", got)
}

// TestADX tests ADX() function (Intel ADX (Multi-Precision Add-Carry Instruction Extensions))
func TestADX(t *testing.T) {
	got := CPU.ADX()
	expected := CPU.Features&ADX == ADX
	if got != expected {
		t.Fatalf("ADX: expected %v, got %v", expected, got)
	}
	t.Log("ADX Support:", got)
}

// TestSHA tests SHA() function (Intel SHA Extensions)
func TestSHA(t *testing.T) {
	got := CPU.SHA()
	expected := CPU.Features&SHA == SHA
	if got != expected {
		t.Fatalf("SHA: expected %v, got %v", expected, got)
	}
	t.Log("SHA Support:", got)
}

// TestAVX512F tests AVX512F() function (AVX-512 Foundation)
func TestAVX512F(t *testing.T) {
	got := CPU.AVX512F()
	expected := CPU.Features&AVX512F == AVX512F
	if got != expected {
		t.Fatalf("AVX512F: expected %v, got %v", expected, got)
	}
	t.Log("AVX512F Support:", got)
}

// TestAVX512DQ tests AVX512DQ() function (AVX-512 Doubleword and Quadword Instructions)
func TestAVX512DQ(t *testing.T) {
	got := CPU.AVX512DQ()
	expected := CPU.Features&AVX512DQ == AVX512DQ
	if got != expected {
		t.Fatalf("AVX512DQ: expected %v, got %v", expected, got)
	}
	t.Log("AVX512DQ Support:", got)
}

// TestAVX512IFMA tests AVX512IFMA() function (AVX-512 Integer Fused Multiply-Add Instructions)
func TestAVX512IFMA(t *testing.T) {
	got := CPU.AVX512IFMA()
	expected := CPU.Features&AVX512IFMA == AVX512IFMA
	if got != expected {
		t.Fatalf("AVX512IFMA: expected %v, got %v", expected, got)
	}
	t.Log("AVX512IFMA Support:", got)
}

// TestAVX512PF tests AVX512PF() function (AVX-512 Prefetch Instructions)
func TestAVX512PF(t *testing.T) {
	got := CPU.AVX512PF()
	expected := CPU.Features&AVX512PF == AVX512PF
	if got != expected {
		t.Fatalf("AVX512PF: expected %v, got %v", expected, got)
	}
	t.Log("AVX512PF Support:", got)
}

// TestAVX512ER tests AVX512ER() function (AVX-512 Exponential and Reciprocal Instructions)
func TestAVX512ER(t *testing.T) {
	got := CPU.AVX512ER()
	expected := CPU.Features&AVX512ER == AVX512ER
	if got != expected {
		t.Fatalf("AVX512ER: expected %v, got %v", expected, got)
	}
	t.Log("AVX512ER Support:", got)
}

// TestAVX512CD tests AVX512CD() function (AVX-512 Conflict Detection Instructions)
func TestAVX512CD(t *testing.T) {
	got := CPU.AVX512CD()
	expected := CPU.Features&AVX512CD == AVX512CD
	if got != expected {
		t.Fatalf("AVX512CD: expected %v, got %v", expected, got)
	}
	t.Log("AVX512CD Support:", got)
}

// TestAVX512BW tests AVX512BW() function (AVX-512 Byte and Word Instructions)
func TestAVX512BW(t *testing.T) {
	got := CPU.AVX512BW()
	expected := CPU.Features&AVX512BW == AVX512BW
	if got != expected {
		t.Fatalf("AVX512BW: expected %v, got %v", expected, got)
	}
	t.Log("AVX512BW Support:", got)
}

// TestAVX512VL tests AVX512VL() function (AVX-512 Vector Length Extensions)
func TestAVX512VL(t *testing.T) {
	got := CPU.AVX512VL()
	expected := CPU.Features&AVX512VL == AVX512VL
	if got != expected {
		t.Fatalf("AVX512VL: expected %v, got %v", expected, got)
	}
	t.Log("AVX512VL Support:", got)
}

// TestAVX512VL tests AVX512VBMI() function (AVX-512 Vector Bit Manipulation Instructions)
func TestAVX512VBMI(t *testing.T) {
	got := CPU.AVX512VBMI()
	expected := CPU.Features&AVX512VBMI == AVX512VBMI
	if got != expected {
		t.Fatalf("AVX512VBMI: expected %v, got %v", expected, got)
	}
	t.Log("AVX512VBMI Support:", got)
}

// TestMPX tests MPX() function (Intel MPX (Memory Protection Extensions))
func TestMPX(t *testing.T) {
	got := CPU.MPX()
	expected := CPU.Features&MPX == MPX
	if got != expected {
		t.Fatalf("MPX: expected %v, got %v", expected, got)
	}
	t.Log("MPX Support:", got)
}

// TestERMS tests ERMS() function (Enhanced REP MOVSB/STOSB)
func TestERMS(t *testing.T) {
	got := CPU.ERMS()
	expected := CPU.Features&ERMS == ERMS
	if got != expected {
		t.Fatalf("ERMS: expected %v, got %v", expected, got)
	}
	t.Log("ERMS Support:", got)
}

// TestVendor writes the detected vendor. Will be 0 if unknown
func TestVendor(t *testing.T) {
	t.Log("Vendor ID:", CPU.VendorID)
}

// Intel returns true if vendor is recognized as Intel
func TestIntel(t *testing.T) {
	got := CPU.Intel()
	expected := CPU.VendorID == Intel
	if got != expected {
		t.Fatalf("TestIntel: expected %v, got %v", expected, got)
	}
	t.Log("TestIntel:", got)
}

// AMD returns true if vendor is recognized as AMD
func TestAMD(t *testing.T) {
	got := CPU.AMD()
	expected := CPU.VendorID == AMD
	if got != expected {
		t.Fatalf("TestAMD: expected %v, got %v", expected, got)
	}
	t.Log("TestAMD:", got)
}

// Transmeta returns true if vendor is recognized as Transmeta
func TestTransmeta(t *testing.T) {
	got := CPU.Transmeta()
	expected := CPU.VendorID == Transmeta
	if got != expected {
		t.Fatalf("TestTransmeta: expected %v, got %v", expected, got)
	}
	t.Log("TestTransmeta:", got)
}

// NSC returns true if vendor is recognized as National Semiconductor
func TestNSC(t *testing.T) {
	got := CPU.NSC()
	expected := CPU.VendorID == NSC
	if got != expected {
		t.Fatalf("TestNSC: expected %v, got %v", expected, got)
	}
	t.Log("TestNSC:", got)
}

// VIA returns true if vendor is recognized as VIA
func TestVIA(t *testing.T) {
	got := CPU.VIA()
	expected := CPU.VendorID == VIA
	if got != expected {
		t.Fatalf("TestVIA: expected %v, got %v", expected, got)
	}
	t.Log("TestVIA:", got)
}

// Test VM function
func TestVM(t *testing.T) {
	t.Log("Vendor ID:", CPU.VM())
}

// Test RTCounter function
func TestRtCounter(t *testing.T) {
	a := CPU.RTCounter()
	b := CPU.RTCounter()
	t.Log("CPU Counter:", a, b, b-a)
}

// Prints the value of Ia32TscAux()
func TestIa32TscAux(t *testing.T) {
	ecx := CPU.Ia32TscAux()
	t.Logf("Ia32TscAux:0x%x\n", ecx)
	if ecx != 0 {
		chip := (ecx & 0xFFF000) >> 12
		core := ecx & 0xFFF
		t.Log("Likely chip, core:", chip, core)
	}
}

func TestThreadsPerCoreNZ(t *testing.T) {
	if CPU.ThreadsPerCore == 0 {
		t.Fatal("threads per core is zero")
	}
}

// Prints the value of LogicalCPU()
func TestLogicalCPU(t *testing.T) {
	t.Log("Currently executing on cpu:", CPU.LogicalCPU())
}

func TestMaxFunction(t *testing.T) {
	expect := maxFunctionID()
	if CPU.maxFunc != expect {
		t.Fatal("Max function does not match, expected", expect, "but got", CPU.maxFunc)
	}
	expect = maxExtendedFunction()
	if CPU.maxExFunc != expect {
		t.Fatal("Max Extended function does not match, expected", expect, "but got", CPU.maxFunc)
	}
}

// This example will calculate the chip/core number on Linux
// Linux encodes numa id (<<12) and core id (8bit) into TSC_AUX.
func ExampleCPUInfo_Ia32TscAux(t *testing.T) {
	ecx := CPU.Ia32TscAux()
	if ecx == 0 {
		fmt.Println("Unknown CPU ID")
		return
	}
	chip := (ecx & 0xFFF000) >> 12
	core := ecx & 0xFFF
	fmt.Println("Chip, Core:", chip, core)
}

/*
func TestPhysical(t *testing.T) {
	var test16 = "CPUID 00000000: 0000000d-756e6547-6c65746e-49656e69 \nCPUID 00000001: 000206d7-03200800-1fbee3ff-bfebfbff   \nCPUID 00000002: 76035a01-00f0b2ff-00000000-00ca0000   \nCPUID 00000003: 00000000-00000000-00000000-00000000   \nCPUID 00000004: 3c004121-01c0003f-0000003f-00000000   \nCPUID 00000004: 3c004122-01c0003f-0000003f-00000000   \nCPUID 00000004: 3c004143-01c0003f-000001ff-00000000   \nCPUID 00000004: 3c07c163-04c0003f-00003fff-00000006   \nCPUID 00000005: 00000040-00000040-00000003-00021120   \nCPUID 00000006: 00000075-00000002-00000009-00000000   \nCPUID 00000007: 00000000-00000000-00000000-00000000   \nCPUID 00000008: 00000000-00000000-00000000-00000000   \nCPUID 00000009: 00000001-00000000-00000000-00000000   \nCPUID 0000000a: 07300403-00000000-00000000-00000603   \nCPUID 0000000b: 00000000-00000000-00000003-00000003   \nCPUID 0000000b: 00000005-00000010-00000201-00000003   \nCPUID 0000000c: 00000000-00000000-00000000-00000000   \nCPUID 0000000d: 00000007-00000340-00000340-00000000   \nCPUID 0000000d: 00000001-00000000-00000000-00000000   \nCPUID 0000000d: 00000100-00000240-00000000-00000000   \nCPUID 80000000: 80000008-00000000-00000000-00000000   \nCPUID 80000001: 00000000-00000000-00000001-2c100800   \nCPUID 80000002: 20202020-49202020-6c65746e-20295228   \nCPUID 80000003: 6e6f6558-20295228-20555043-322d3545   \nCPUID 80000004: 20303636-20402030-30322e32-007a4847   \nCPUID 80000005: 00000000-00000000-00000000-00000000   \nCPUID 80000006: 00000000-00000000-01006040-00000000   \nCPUID 80000007: 00000000-00000000-00000000-00000100   \nCPUID 80000008: 0000302e-00000000-00000000-00000000"
	restore := mockCPU([]byte(test16))
	Detect()
	t.Log("Name:", CPU.BrandName)
	n := maxFunctionID()
	t.Logf("Max Function:0x%x\n", n)
	n = maxExtendedFunction()
	t.Logf("Max Extended Function:0x%x\n", n)
	t.Log("PhysicalCores:", CPU.PhysicalCores)
	t.Log("ThreadsPerCore:", CPU.ThreadsPerCore)
	t.Log("LogicalCores:", CPU.LogicalCores)
	t.Log("Family", CPU.Family, "Model:", CPU.Model)
	t.Log("Features:", CPU.Features)
	t.Log("Cacheline bytes:", CPU.CacheLine)
	t.Log("L1 Instruction Cache:", CPU.Cache.L1I, "bytes")
	t.Log("L1 Data Cache:", CPU.Cache.L1D, "bytes")
	t.Log("L2 Cache:", CPU.Cache.L2, "bytes")
	t.Log("L3 Cache:", CPU.Cache.L3, "bytes")
	if CPU.LogicalCores > 0 && CPU.PhysicalCores > 0 {
		if CPU.LogicalCores != CPU.PhysicalCores*CPU.ThreadsPerCore {
			t.Fatalf("Core count mismatch, LogicalCores (%d) != PhysicalCores (%d) * CPU.ThreadsPerCore (%d)",
				CPU.LogicalCores, CPU.PhysicalCores, CPU.ThreadsPerCore)
		}
	}

	if CPU.ThreadsPerCore > 1 && !CPU.HTT() {
		t.Fatalf("Hyperthreading not detected")
	}
	if CPU.ThreadsPerCore == 1 && CPU.HTT() {
		t.Fatalf("Hyperthreading detected, but only 1 Thread per core")
	}
	restore()
	Detect()
	TestCPUID(t)
}
*/

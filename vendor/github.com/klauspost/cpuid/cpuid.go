// Copyright (c) 2015 Klaus Post, released under MIT License. See LICENSE file.

// Package cpuid provides information about the CPU running the current program.
//
// CPU features are detected on startup, and kept for fast access through the life of the application.
// Currently x86 / x64 (AMD64) is supported.
//
// You can access the CPU information by accessing the shared CPU variable of the cpuid library.
//
// Package home: https://github.com/klauspost/cpuid
package cpuid

import "strings"

// Vendor is a representation of a CPU vendor.
type Vendor int

const (
	Other Vendor = iota
	Intel
	AMD
	VIA
	Transmeta
	NSC
	KVM  // Kernel-based Virtual Machine
	MSVM // Microsoft Hyper-V or Windows Virtual PC
	VMware
	XenHVM
)

const (
	CMOV        = 1 << iota // i686 CMOV
	NX                      // NX (No-Execute) bit
	AMD3DNOW                // AMD 3DNOW
	AMD3DNOWEXT             // AMD 3DNowExt
	MMX                     // standard MMX
	MMXEXT                  // SSE integer functions or AMD MMX ext
	SSE                     // SSE functions
	SSE2                    // P4 SSE functions
	SSE3                    // Prescott SSE3 functions
	SSSE3                   // Conroe SSSE3 functions
	SSE4                    // Penryn SSE4.1 functions
	SSE4A                   // AMD Barcelona microarchitecture SSE4a instructions
	SSE42                   // Nehalem SSE4.2 functions
	AVX                     // AVX functions
	AVX2                    // AVX2 functions
	FMA3                    // Intel FMA 3
	FMA4                    // Bulldozer FMA4 functions
	XOP                     // Bulldozer XOP functions
	F16C                    // Half-precision floating-point conversion
	BMI1                    // Bit Manipulation Instruction Set 1
	BMI2                    // Bit Manipulation Instruction Set 2
	TBM                     // AMD Trailing Bit Manipulation
	LZCNT                   // LZCNT instruction
	POPCNT                  // POPCNT instruction
	AESNI                   // Advanced Encryption Standard New Instructions
	CLMUL                   // Carry-less Multiplication
	HTT                     // Hyperthreading (enabled)
	HLE                     // Hardware Lock Elision
	RTM                     // Restricted Transactional Memory
	RDRAND                  // RDRAND instruction is available
	RDSEED                  // RDSEED instruction is available
	ADX                     // Intel ADX (Multi-Precision Add-Carry Instruction Extensions)
	SHA                     // Intel SHA Extensions
	AVX512F                 // AVX-512 Foundation
	AVX512DQ                // AVX-512 Doubleword and Quadword Instructions
	AVX512IFMA              // AVX-512 Integer Fused Multiply-Add Instructions
	AVX512PF                // AVX-512 Prefetch Instructions
	AVX512ER                // AVX-512 Exponential and Reciprocal Instructions
	AVX512CD                // AVX-512 Conflict Detection Instructions
	AVX512BW                // AVX-512 Byte and Word Instructions
	AVX512VL                // AVX-512 Vector Length Extensions
	AVX512VBMI              // AVX-512 Vector Bit Manipulation Instructions
	MPX                     // Intel MPX (Memory Protection Extensions)
	ERMS                    // Enhanced REP MOVSB/STOSB
	RDTSCP                  // RDTSCP Instruction
	CX16                    // CMPXCHG16B Instruction
	SGX                     // Software Guard Extensions
	IBPB                    // Indirect Branch Restricted Speculation (IBRS) and Indirect Branch Predictor Barrier (IBPB)
	STIBP                   // Single Thread Indirect Branch Predictors

	// Performance indicators
	SSE2SLOW // SSE2 is supported, but usually not faster
	SSE3SLOW // SSE3 is supported, but usually not faster
	ATOM     // Atom processor, some SSSE3 instructions are slower
)

var flagNames = map[Flags]string{
	CMOV:        "CMOV",        // i686 CMOV
	NX:          "NX",          // NX (No-Execute) bit
	AMD3DNOW:    "AMD3DNOW",    // AMD 3DNOW
	AMD3DNOWEXT: "AMD3DNOWEXT", // AMD 3DNowExt
	MMX:         "MMX",         // Standard MMX
	MMXEXT:      "MMXEXT",      // SSE integer functions or AMD MMX ext
	SSE:         "SSE",         // SSE functions
	SSE2:        "SSE2",        // P4 SSE2 functions
	SSE3:        "SSE3",        // Prescott SSE3 functions
	SSSE3:       "SSSE3",       // Conroe SSSE3 functions
	SSE4:        "SSE4.1",      // Penryn SSE4.1 functions
	SSE4A:       "SSE4A",       // AMD Barcelona microarchitecture SSE4a instructions
	SSE42:       "SSE4.2",      // Nehalem SSE4.2 functions
	AVX:         "AVX",         // AVX functions
	AVX2:        "AVX2",        // AVX functions
	FMA3:        "FMA3",        // Intel FMA 3
	FMA4:        "FMA4",        // Bulldozer FMA4 functions
	XOP:         "XOP",         // Bulldozer XOP functions
	F16C:        "F16C",        // Half-precision floating-point conversion
	BMI1:        "BMI1",        // Bit Manipulation Instruction Set 1
	BMI2:        "BMI2",        // Bit Manipulation Instruction Set 2
	TBM:         "TBM",         // AMD Trailing Bit Manipulation
	LZCNT:       "LZCNT",       // LZCNT instruction
	POPCNT:      "POPCNT",      // POPCNT instruction
	AESNI:       "AESNI",       // Advanced Encryption Standard New Instructions
	CLMUL:       "CLMUL",       // Carry-less Multiplication
	HTT:         "HTT",         // Hyperthreading (enabled)
	HLE:         "HLE",         // Hardware Lock Elision
	RTM:         "RTM",         // Restricted Transactional Memory
	RDRAND:      "RDRAND",      // RDRAND instruction is available
	RDSEED:      "RDSEED",      // RDSEED instruction is available
	ADX:         "ADX",         // Intel ADX (Multi-Precision Add-Carry Instruction Extensions)
	SHA:         "SHA",         // Intel SHA Extensions
	AVX512F:     "AVX512F",     // AVX-512 Foundation
	AVX512DQ:    "AVX512DQ",    // AVX-512 Doubleword and Quadword Instructions
	AVX512IFMA:  "AVX512IFMA",  // AVX-512 Integer Fused Multiply-Add Instructions
	AVX512PF:    "AVX512PF",    // AVX-512 Prefetch Instructions
	AVX512ER:    "AVX512ER",    // AVX-512 Exponential and Reciprocal Instructions
	AVX512CD:    "AVX512CD",    // AVX-512 Conflict Detection Instructions
	AVX512BW:    "AVX512BW",    // AVX-512 Byte and Word Instructions
	AVX512VL:    "AVX512VL",    // AVX-512 Vector Length Extensions
	AVX512VBMI:  "AVX512VBMI",  // AVX-512 Vector Bit Manipulation Instructions
	MPX:         "MPX",         // Intel MPX (Memory Protection Extensions)
	ERMS:        "ERMS",        // Enhanced REP MOVSB/STOSB
	RDTSCP:      "RDTSCP",      // RDTSCP Instruction
	CX16:        "CX16",        // CMPXCHG16B Instruction
	SGX:         "SGX",         // Software Guard Extensions
	IBPB:        "IBPB",        // Indirect Branch Restricted Speculation and Indirect Branch Predictor Barrier
	STIBP:       "STIBP",       // Single Thread Indirect Branch Predictors

	// Performance indicators
	SSE2SLOW: "SSE2SLOW", // SSE2 supported, but usually not faster
	SSE3SLOW: "SSE3SLOW", // SSE3 supported, but usually not faster
	ATOM:     "ATOM",     // Atom processor, some SSSE3 instructions are slower

}

// CPUInfo contains information about the detected system CPU.
type CPUInfo struct {
	BrandName      string // Brand name reported by the CPU
	VendorID       Vendor // Comparable CPU vendor ID
	Features       Flags  // Features of the CPU
	PhysicalCores  int    // Number of physical processor cores in your CPU. Will be 0 if undetectable.
	ThreadsPerCore int    // Number of threads per physical core. Will be 1 if undetectable.
	LogicalCores   int    // Number of physical cores times threads that can run on each core through the use of hyperthreading. Will be 0 if undetectable.
	Family         int    // CPU family number
	Model          int    // CPU model number
	CacheLine      int    // Cache line size in bytes. Will be 0 if undetectable.
	Cache          struct {
		L1I int // L1 Instruction Cache (per core or shared). Will be -1 if undetected
		L1D int // L1 Data Cache (per core or shared). Will be -1 if undetected
		L2  int // L2 Cache (per core or shared). Will be -1 if undetected
		L3  int // L3 Instruction Cache (per core or shared). Will be -1 if undetected
	}
	SGX       SGXSupport
	maxFunc   uint32
	maxExFunc uint32
}

var cpuid func(op uint32) (eax, ebx, ecx, edx uint32)
var cpuidex func(op, op2 uint32) (eax, ebx, ecx, edx uint32)
var xgetbv func(index uint32) (eax, edx uint32)
var rdtscpAsm func() (eax, ebx, ecx, edx uint32)

// CPU contains information about the CPU as detected on startup,
// or when Detect last was called.
//
// Use this as the primary entry point to you data,
// this way queries are
var CPU CPUInfo

func init() {
	initCPU()
	Detect()
}

// Detect will re-detect current CPU info.
// This will replace the content of the exported CPU variable.
//
// Unless you expect the CPU to change while you are running your program
// you should not need to call this function.
// If you call this, you must ensure that no other goroutine is accessing the
// exported CPU variable.
func Detect() {
	CPU.maxFunc = maxFunctionID()
	CPU.maxExFunc = maxExtendedFunction()
	CPU.BrandName = brandName()
	CPU.CacheLine = cacheLine()
	CPU.Family, CPU.Model = familyModel()
	CPU.Features = support()
	CPU.SGX = hasSGX(CPU.Features&SGX != 0)
	CPU.ThreadsPerCore = threadsPerCore()
	CPU.LogicalCores = logicalCores()
	CPU.PhysicalCores = physicalCores()
	CPU.VendorID = vendorID()
	CPU.cacheSize()
}

// Generated here: http://play.golang.org/p/BxFH2Gdc0G

// Cmov indicates support of CMOV instructions
func (c CPUInfo) Cmov() bool {
	return c.Features&CMOV != 0
}

// Amd3dnow indicates support of AMD 3DNOW! instructions
func (c CPUInfo) Amd3dnow() bool {
	return c.Features&AMD3DNOW != 0
}

// Amd3dnowExt indicates support of AMD 3DNOW! Extended instructions
func (c CPUInfo) Amd3dnowExt() bool {
	return c.Features&AMD3DNOWEXT != 0
}

// MMX indicates support of MMX instructions
func (c CPUInfo) MMX() bool {
	return c.Features&MMX != 0
}

// MMXExt indicates support of MMXEXT instructions
// (SSE integer functions or AMD MMX ext)
func (c CPUInfo) MMXExt() bool {
	return c.Features&MMXEXT != 0
}

// SSE indicates support of SSE instructions
func (c CPUInfo) SSE() bool {
	return c.Features&SSE != 0
}

// SSE2 indicates support of SSE 2 instructions
func (c CPUInfo) SSE2() bool {
	return c.Features&SSE2 != 0
}

// SSE3 indicates support of SSE 3 instructions
func (c CPUInfo) SSE3() bool {
	return c.Features&SSE3 != 0
}

// SSSE3 indicates support of SSSE 3 instructions
func (c CPUInfo) SSSE3() bool {
	return c.Features&SSSE3 != 0
}

// SSE4 indicates support of SSE 4 (also called SSE 4.1) instructions
func (c CPUInfo) SSE4() bool {
	return c.Features&SSE4 != 0
}

// SSE42 indicates support of SSE4.2 instructions
func (c CPUInfo) SSE42() bool {
	return c.Features&SSE42 != 0
}

// AVX indicates support of AVX instructions
// and operating system support of AVX instructions
func (c CPUInfo) AVX() bool {
	return c.Features&AVX != 0
}

// AVX2 indicates support of AVX2 instructions
func (c CPUInfo) AVX2() bool {
	return c.Features&AVX2 != 0
}

// FMA3 indicates support of FMA3 instructions
func (c CPUInfo) FMA3() bool {
	return c.Features&FMA3 != 0
}

// FMA4 indicates support of FMA4 instructions
func (c CPUInfo) FMA4() bool {
	return c.Features&FMA4 != 0
}

// XOP indicates support of XOP instructions
func (c CPUInfo) XOP() bool {
	return c.Features&XOP != 0
}

// F16C indicates support of F16C instructions
func (c CPUInfo) F16C() bool {
	return c.Features&F16C != 0
}

// BMI1 indicates support of BMI1 instructions
func (c CPUInfo) BMI1() bool {
	return c.Features&BMI1 != 0
}

// BMI2 indicates support of BMI2 instructions
func (c CPUInfo) BMI2() bool {
	return c.Features&BMI2 != 0
}

// TBM indicates support of TBM instructions
// (AMD Trailing Bit Manipulation)
func (c CPUInfo) TBM() bool {
	return c.Features&TBM != 0
}

// Lzcnt indicates support of LZCNT instruction
func (c CPUInfo) Lzcnt() bool {
	return c.Features&LZCNT != 0
}

// Popcnt indicates support of POPCNT instruction
func (c CPUInfo) Popcnt() bool {
	return c.Features&POPCNT != 0
}

// HTT indicates the processor has Hyperthreading enabled
func (c CPUInfo) HTT() bool {
	return c.Features&HTT != 0
}

// SSE2Slow indicates that SSE2 may be slow on this processor
func (c CPUInfo) SSE2Slow() bool {
	return c.Features&SSE2SLOW != 0
}

// SSE3Slow indicates that SSE3 may be slow on this processor
func (c CPUInfo) SSE3Slow() bool {
	return c.Features&SSE3SLOW != 0
}

// AesNi indicates support of AES-NI instructions
// (Advanced Encryption Standard New Instructions)
func (c CPUInfo) AesNi() bool {
	return c.Features&AESNI != 0
}

// Clmul indicates support of CLMUL instructions
// (Carry-less Multiplication)
func (c CPUInfo) Clmul() bool {
	return c.Features&CLMUL != 0
}

// NX indicates support of NX (No-Execute) bit
func (c CPUInfo) NX() bool {
	return c.Features&NX != 0
}

// SSE4A indicates support of AMD Barcelona microarchitecture SSE4a instructions
func (c CPUInfo) SSE4A() bool {
	return c.Features&SSE4A != 0
}

// HLE indicates support of Hardware Lock Elision
func (c CPUInfo) HLE() bool {
	return c.Features&HLE != 0
}

// RTM indicates support of Restricted Transactional Memory
func (c CPUInfo) RTM() bool {
	return c.Features&RTM != 0
}

// Rdrand indicates support of RDRAND instruction is available
func (c CPUInfo) Rdrand() bool {
	return c.Features&RDRAND != 0
}

// Rdseed indicates support of RDSEED instruction is available
func (c CPUInfo) Rdseed() bool {
	return c.Features&RDSEED != 0
}

// ADX indicates support of Intel ADX (Multi-Precision Add-Carry Instruction Extensions)
func (c CPUInfo) ADX() bool {
	return c.Features&ADX != 0
}

// SHA indicates support of Intel SHA Extensions
func (c CPUInfo) SHA() bool {
	return c.Features&SHA != 0
}

// AVX512F indicates support of AVX-512 Foundation
func (c CPUInfo) AVX512F() bool {
	return c.Features&AVX512F != 0
}

// AVX512DQ indicates support of AVX-512 Doubleword and Quadword Instructions
func (c CPUInfo) AVX512DQ() bool {
	return c.Features&AVX512DQ != 0
}

// AVX512IFMA indicates support of AVX-512 Integer Fused Multiply-Add Instructions
func (c CPUInfo) AVX512IFMA() bool {
	return c.Features&AVX512IFMA != 0
}

// AVX512PF indicates support of AVX-512 Prefetch Instructions
func (c CPUInfo) AVX512PF() bool {
	return c.Features&AVX512PF != 0
}

// AVX512ER indicates support of AVX-512 Exponential and Reciprocal Instructions
func (c CPUInfo) AVX512ER() bool {
	return c.Features&AVX512ER != 0
}

// AVX512CD indicates support of AVX-512 Conflict Detection Instructions
func (c CPUInfo) AVX512CD() bool {
	return c.Features&AVX512CD != 0
}

// AVX512BW indicates support of AVX-512 Byte and Word Instructions
func (c CPUInfo) AVX512BW() bool {
	return c.Features&AVX512BW != 0
}

// AVX512VL indicates support of AVX-512 Vector Length Extensions
func (c CPUInfo) AVX512VL() bool {
	return c.Features&AVX512VL != 0
}

// AVX512VBMI indicates support of AVX-512 Vector Bit Manipulation Instructions
func (c CPUInfo) AVX512VBMI() bool {
	return c.Features&AVX512VBMI != 0
}

// MPX indicates support of Intel MPX (Memory Protection Extensions)
func (c CPUInfo) MPX() bool {
	return c.Features&MPX != 0
}

// ERMS indicates support of Enhanced REP MOVSB/STOSB
func (c CPUInfo) ERMS() bool {
	return c.Features&ERMS != 0
}

// RDTSCP Instruction is available.
func (c CPUInfo) RDTSCP() bool {
	return c.Features&RDTSCP != 0
}

// CX16 indicates if CMPXCHG16B instruction is available.
func (c CPUInfo) CX16() bool {
	return c.Features&CX16 != 0
}

// TSX is split into HLE (Hardware Lock Elision) and RTM (Restricted Transactional Memory) detection.
// So TSX simply checks that.
func (c CPUInfo) TSX() bool {
	return c.Features&(HLE|RTM) == HLE|RTM
}

// Atom indicates an Atom processor
func (c CPUInfo) Atom() bool {
	return c.Features&ATOM != 0
}

// Intel returns true if vendor is recognized as Intel
func (c CPUInfo) Intel() bool {
	return c.VendorID == Intel
}

// AMD returns true if vendor is recognized as AMD
func (c CPUInfo) AMD() bool {
	return c.VendorID == AMD
}

// Transmeta returns true if vendor is recognized as Transmeta
func (c CPUInfo) Transmeta() bool {
	return c.VendorID == Transmeta
}

// NSC returns true if vendor is recognized as National Semiconductor
func (c CPUInfo) NSC() bool {
	return c.VendorID == NSC
}

// VIA returns true if vendor is recognized as VIA
func (c CPUInfo) VIA() bool {
	return c.VendorID == VIA
}

// RTCounter returns the 64-bit time-stamp counter
// Uses the RDTSCP instruction. The value 0 is returned
// if the CPU does not support the instruction.
func (c CPUInfo) RTCounter() uint64 {
	if !c.RDTSCP() {
		return 0
	}
	a, _, _, d := rdtscpAsm()
	return uint64(a) | (uint64(d) << 32)
}

// Ia32TscAux returns the IA32_TSC_AUX part of the RDTSCP.
// This variable is OS dependent, but on Linux contains information
// about the current cpu/core the code is running on.
// If the RDTSCP instruction isn't supported on the CPU, the value 0 is returned.
func (c CPUInfo) Ia32TscAux() uint32 {
	if !c.RDTSCP() {
		return 0
	}
	_, _, ecx, _ := rdtscpAsm()
	return ecx
}

// LogicalCPU will return the Logical CPU the code is currently executing on.
// This is likely to change when the OS re-schedules the running thread
// to another CPU.
// If the current core cannot be detected, -1 will be returned.
func (c CPUInfo) LogicalCPU() int {
	if c.maxFunc < 1 {
		return -1
	}
	_, ebx, _, _ := cpuid(1)
	return int(ebx >> 24)
}

// VM Will return true if the cpu id indicates we are in
// a virtual machine. This is only a hint, and will very likely
// have many false negatives.
func (c CPUInfo) VM() bool {
	switch c.VendorID {
	case MSVM, KVM, VMware, XenHVM:
		return true
	}
	return false
}

// Flags contains detected cpu features and caracteristics
type Flags uint64

// String returns a string representation of the detected
// CPU features.
func (f Flags) String() string {
	return strings.Join(f.Strings(), ",")
}

// Strings returns and array of the detected features.
func (f Flags) Strings() []string {
	s := support()
	r := make([]string, 0, 20)
	for i := uint(0); i < 64; i++ {
		key := Flags(1 << i)
		val := flagNames[key]
		if s&key != 0 {
			r = append(r, val)
		}
	}
	return r
}

func maxExtendedFunction() uint32 {
	eax, _, _, _ := cpuid(0x80000000)
	return eax
}

func maxFunctionID() uint32 {
	a, _, _, _ := cpuid(0)
	return a
}

func brandName() string {
	if maxExtendedFunction() >= 0x80000004 {
		v := make([]uint32, 0, 48)
		for i := uint32(0); i < 3; i++ {
			a, b, c, d := cpuid(0x80000002 + i)
			v = append(v, a, b, c, d)
		}
		return strings.Trim(string(valAsString(v...)), " ")
	}
	return "unknown"
}

func threadsPerCore() int {
	mfi := maxFunctionID()
	if mfi < 0x4 || vendorID() != Intel {
		return 1
	}

	if mfi < 0xb {
		_, b, _, d := cpuid(1)
		if (d & (1 << 28)) != 0 {
			// v will contain logical core count
			v := (b >> 16) & 255
			if v > 1 {
				a4, _, _, _ := cpuid(4)
				// physical cores
				v2 := (a4 >> 26) + 1
				if v2 > 0 {
					return int(v) / int(v2)
				}
			}
		}
		return 1
	}
	_, b, _, _ := cpuidex(0xb, 0)
	if b&0xffff == 0 {
		return 1
	}
	return int(b & 0xffff)
}

func logicalCores() int {
	mfi := maxFunctionID()
	switch vendorID() {
	case Intel:
		// Use this on old Intel processors
		if mfi < 0xb {
			if mfi < 1 {
				return 0
			}
			// CPUID.1:EBX[23:16] represents the maximum number of addressable IDs (initial APIC ID)
			// that can be assigned to logical processors in a physical package.
			// The value may not be the same as the number of logical processors that are present in the hardware of a physical package.
			_, ebx, _, _ := cpuid(1)
			logical := (ebx >> 16) & 0xff
			return int(logical)
		}
		_, b, _, _ := cpuidex(0xb, 1)
		return int(b & 0xffff)
	case AMD:
		_, b, _, _ := cpuid(1)
		return int((b >> 16) & 0xff)
	default:
		return 0
	}
}

func familyModel() (int, int) {
	if maxFunctionID() < 0x1 {
		return 0, 0
	}
	eax, _, _, _ := cpuid(1)
	family := ((eax >> 8) & 0xf) + ((eax >> 20) & 0xff)
	model := ((eax >> 4) & 0xf) + ((eax >> 12) & 0xf0)
	return int(family), int(model)
}

func physicalCores() int {
	switch vendorID() {
	case Intel:
		return logicalCores() / threadsPerCore()
	case AMD:
		if maxExtendedFunction() >= 0x80000008 {
			_, _, c, _ := cpuid(0x80000008)
			return int(c&0xff) + 1
		}
	}
	return 0
}

// Except from http://en.wikipedia.org/wiki/CPUID#EAX.3D0:_Get_vendor_ID
var vendorMapping = map[string]Vendor{
	"AMDisbetter!": AMD,
	"AuthenticAMD": AMD,
	"CentaurHauls": VIA,
	"GenuineIntel": Intel,
	"TransmetaCPU": Transmeta,
	"GenuineTMx86": Transmeta,
	"Geode by NSC": NSC,
	"VIA VIA VIA ": VIA,
	"KVMKVMKVMKVM": KVM,
	"Microsoft Hv": MSVM,
	"VMwareVMware": VMware,
	"XenVMMXenVMM": XenHVM,
}

func vendorID() Vendor {
	_, b, c, d := cpuid(0)
	v := valAsString(b, d, c)
	vend, ok := vendorMapping[string(v)]
	if !ok {
		return Other
	}
	return vend
}

func cacheLine() int {
	if maxFunctionID() < 0x1 {
		return 0
	}

	_, ebx, _, _ := cpuid(1)
	cache := (ebx & 0xff00) >> 5 // cflush size
	if cache == 0 && maxExtendedFunction() >= 0x80000006 {
		_, _, ecx, _ := cpuid(0x80000006)
		cache = ecx & 0xff // cacheline size
	}
	// TODO: Read from Cache and TLB Information
	return int(cache)
}

func (c *CPUInfo) cacheSize() {
	c.Cache.L1D = -1
	c.Cache.L1I = -1
	c.Cache.L2 = -1
	c.Cache.L3 = -1
	vendor := vendorID()
	switch vendor {
	case Intel:
		if maxFunctionID() < 4 {
			return
		}
		for i := uint32(0); ; i++ {
			eax, ebx, ecx, _ := cpuidex(4, i)
			cacheType := eax & 15
			if cacheType == 0 {
				break
			}
			cacheLevel := (eax >> 5) & 7
			coherency := int(ebx&0xfff) + 1
			partitions := int((ebx>>12)&0x3ff) + 1
			associativity := int((ebx>>22)&0x3ff) + 1
			sets := int(ecx) + 1
			size := associativity * partitions * coherency * sets
			switch cacheLevel {
			case 1:
				if cacheType == 1 {
					// 1 = Data Cache
					c.Cache.L1D = size
				} else if cacheType == 2 {
					// 2 = Instruction Cache
					c.Cache.L1I = size
				} else {
					if c.Cache.L1D < 0 {
						c.Cache.L1I = size
					}
					if c.Cache.L1I < 0 {
						c.Cache.L1I = size
					}
				}
			case 2:
				c.Cache.L2 = size
			case 3:
				c.Cache.L3 = size
			}
		}
	case AMD:
		// Untested.
		if maxExtendedFunction() < 0x80000005 {
			return
		}
		_, _, ecx, edx := cpuid(0x80000005)
		c.Cache.L1D = int(((ecx >> 24) & 0xFF) * 1024)
		c.Cache.L1I = int(((edx >> 24) & 0xFF) * 1024)

		if maxExtendedFunction() < 0x80000006 {
			return
		}
		_, _, ecx, _ = cpuid(0x80000006)
		c.Cache.L2 = int(((ecx >> 16) & 0xFFFF) * 1024)
	}

	return
}

type SGXSupport struct {
	Available           bool
	SGX1Supported       bool
	SGX2Supported       bool
	MaxEnclaveSizeNot64 int64
	MaxEnclaveSize64    int64
}

func hasSGX(available bool) (rval SGXSupport) {
	rval.Available = available

	if !available {
		return
	}

	a, _, _, d := cpuidex(0x12, 0)
	rval.SGX1Supported = a&0x01 != 0
	rval.SGX2Supported = a&0x02 != 0
	rval.MaxEnclaveSizeNot64 = 1 << (d & 0xFF)     // pow 2
	rval.MaxEnclaveSize64 = 1 << ((d >> 8) & 0xFF) // pow 2

	return
}

func support() Flags {
	mfi := maxFunctionID()
	vend := vendorID()
	if mfi < 0x1 {
		return 0
	}
	rval := uint64(0)
	_, _, c, d := cpuid(1)
	if (d & (1 << 15)) != 0 {
		rval |= CMOV
	}
	if (d & (1 << 23)) != 0 {
		rval |= MMX
	}
	if (d & (1 << 25)) != 0 {
		rval |= MMXEXT
	}
	if (d & (1 << 25)) != 0 {
		rval |= SSE
	}
	if (d & (1 << 26)) != 0 {
		rval |= SSE2
	}
	if (c & 1) != 0 {
		rval |= SSE3
	}
	if (c & 0x00000200) != 0 {
		rval |= SSSE3
	}
	if (c & 0x00080000) != 0 {
		rval |= SSE4
	}
	if (c & 0x00100000) != 0 {
		rval |= SSE42
	}
	if (c & (1 << 25)) != 0 {
		rval |= AESNI
	}
	if (c & (1 << 1)) != 0 {
		rval |= CLMUL
	}
	if c&(1<<23) != 0 {
		rval |= POPCNT
	}
	if c&(1<<30) != 0 {
		rval |= RDRAND
	}
	if c&(1<<29) != 0 {
		rval |= F16C
	}
	if c&(1<<13) != 0 {
		rval |= CX16
	}
	if vend == Intel && (d&(1<<28)) != 0 && mfi >= 4 {
		if threadsPerCore() > 1 {
			rval |= HTT
		}
	}

	// Check XGETBV, OXSAVE and AVX bits
	if c&(1<<26) != 0 && c&(1<<27) != 0 && c&(1<<28) != 0 {
		// Check for OS support
		eax, _ := xgetbv(0)
		if (eax & 0x6) == 0x6 {
			rval |= AVX
			if (c & 0x00001000) != 0 {
				rval |= FMA3
			}
		}
	}

	// Check AVX2, AVX2 requires OS support, but BMI1/2 don't.
	if mfi >= 7 {
		_, ebx, ecx, edx := cpuidex(7, 0)
		if (rval&AVX) != 0 && (ebx&0x00000020) != 0 {
			rval |= AVX2
		}
		if (ebx & 0x00000008) != 0 {
			rval |= BMI1
			if (ebx & 0x00000100) != 0 {
				rval |= BMI2
			}
		}
		if ebx&(1<<2) != 0 {
			rval |= SGX
		}
		if ebx&(1<<4) != 0 {
			rval |= HLE
		}
		if ebx&(1<<9) != 0 {
			rval |= ERMS
		}
		if ebx&(1<<11) != 0 {
			rval |= RTM
		}
		if ebx&(1<<14) != 0 {
			rval |= MPX
		}
		if ebx&(1<<18) != 0 {
			rval |= RDSEED
		}
		if ebx&(1<<19) != 0 {
			rval |= ADX
		}
		if ebx&(1<<29) != 0 {
			rval |= SHA
		}
		if edx&(1<<26) != 0 {
			rval |= IBPB
		}
		if edx&(1<<27) != 0 {
			rval |= STIBP
		}

		// Only detect AVX-512 features if XGETBV is supported
		if c&((1<<26)|(1<<27)) == (1<<26)|(1<<27) {
			// Check for OS support
			eax, _ := xgetbv(0)

			// Verify that XCR0[7:5] = ‘111b’ (OPMASK state, upper 256-bit of ZMM0-ZMM15 and
			// ZMM16-ZMM31 state are enabled by OS)
			/// and that XCR0[2:1] = ‘11b’ (XMM state and YMM state are enabled by OS).
			if (eax>>5)&7 == 7 && (eax>>1)&3 == 3 {
				if ebx&(1<<16) != 0 {
					rval |= AVX512F
				}
				if ebx&(1<<17) != 0 {
					rval |= AVX512DQ
				}
				if ebx&(1<<21) != 0 {
					rval |= AVX512IFMA
				}
				if ebx&(1<<26) != 0 {
					rval |= AVX512PF
				}
				if ebx&(1<<27) != 0 {
					rval |= AVX512ER
				}
				if ebx&(1<<28) != 0 {
					rval |= AVX512CD
				}
				if ebx&(1<<30) != 0 {
					rval |= AVX512BW
				}
				if ebx&(1<<31) != 0 {
					rval |= AVX512VL
				}
				// ecx
				if ecx&(1<<1) != 0 {
					rval |= AVX512VBMI
				}
			}
		}
	}

	if maxExtendedFunction() >= 0x80000001 {
		_, _, c, d := cpuid(0x80000001)
		if (c & (1 << 5)) != 0 {
			rval |= LZCNT
			rval |= POPCNT
		}
		if (d & (1 << 31)) != 0 {
			rval |= AMD3DNOW
		}
		if (d & (1 << 30)) != 0 {
			rval |= AMD3DNOWEXT
		}
		if (d & (1 << 23)) != 0 {
			rval |= MMX
		}
		if (d & (1 << 22)) != 0 {
			rval |= MMXEXT
		}
		if (c & (1 << 6)) != 0 {
			rval |= SSE4A
		}
		if d&(1<<20) != 0 {
			rval |= NX
		}
		if d&(1<<27) != 0 {
			rval |= RDTSCP
		}

		/* Allow for selectively disabling SSE2 functions on AMD processors
		   with SSE2 support but not SSE4a. This includes Athlon64, some
		   Opteron, and some Sempron processors. MMX, SSE, or 3DNow! are faster
		   than SSE2 often enough to utilize this special-case flag.
		   AV_CPU_FLAG_SSE2 and AV_CPU_FLAG_SSE2SLOW are both set in this case
		   so that SSE2 is used unless explicitly disabled by checking
		   AV_CPU_FLAG_SSE2SLOW. */
		if vendorID() != Intel &&
			rval&SSE2 != 0 && (c&0x00000040) == 0 {
			rval |= SSE2SLOW
		}

		/* XOP and FMA4 use the AVX instruction coding scheme, so they can't be
		 * used unless the OS has AVX support. */
		if (rval & AVX) != 0 {
			if (c & 0x00000800) != 0 {
				rval |= XOP
			}
			if (c & 0x00010000) != 0 {
				rval |= FMA4
			}
		}

		if vendorID() == Intel {
			family, model := familyModel()
			if family == 6 && (model == 9 || model == 13 || model == 14) {
				/* 6/9 (pentium-m "banias"), 6/13 (pentium-m "dothan"), and
				 * 6/14 (core1 "yonah") theoretically support sse2, but it's
				 * usually slower than mmx. */
				if (rval & SSE2) != 0 {
					rval |= SSE2SLOW
				}
				if (rval & SSE3) != 0 {
					rval |= SSE3SLOW
				}
			}
			/* The Atom processor has SSSE3 support, which is useful in many cases,
			 * but sometimes the SSSE3 version is slower than the SSE2 equivalent
			 * on the Atom, but is generally faster on other processors supporting
			 * SSSE3. This flag allows for selectively disabling certain SSSE3
			 * functions on the Atom. */
			if family == 6 && model == 28 {
				rval |= ATOM
			}
		}
	}
	return Flags(rval)
}

func valAsString(values ...uint32) []byte {
	r := make([]byte, 4*len(values))
	for i, v := range values {
		dst := r[i*4:]
		dst[0] = byte(v & 0xff)
		dst[1] = byte((v >> 8) & 0xff)
		dst[2] = byte((v >> 16) & 0xff)
		dst[3] = byte((v >> 24) & 0xff)
		switch {
		case dst[0] == 0:
			return r[:i*4]
		case dst[1] == 0:
			return r[:i*4+1]
		case dst[2] == 0:
			return r[:i*4+2]
		case dst[3] == 0:
			return r[:i*4+3]
		}
	}
	return r
}

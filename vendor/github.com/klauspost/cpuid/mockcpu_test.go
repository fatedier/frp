package cpuid

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"testing"
)

type fakecpuid map[uint32][][]uint32

type idfuncs struct {
	cpuid   func(op uint32) (eax, ebx, ecx, edx uint32)
	cpuidex func(op, op2 uint32) (eax, ebx, ecx, edx uint32)
	xgetbv  func(index uint32) (eax, edx uint32)
}

func (f fakecpuid) String() string {
	var out = make([]string, 0, len(f))
	for key, val := range f {
		for _, v := range val {
			out = append(out, fmt.Sprintf("CPUID %08x: [%08x, %08x, %08x, %08x]", key, v[0], v[1], v[2], v[3]))
		}
	}
	sorter := sort.StringSlice(out)
	sort.Sort(&sorter)
	return strings.Join(sorter, "\n")
}

func mockCPU(def []byte) func() {
	lines := strings.Split(string(def), "\n")
	anyfound := false
	fakeID := make(fakecpuid)
	for _, line := range lines {
		line = strings.Trim(line, "\r\t ")
		if !strings.HasPrefix(line, "CPUID") {
			continue
		}
		// Only collect for first cpu
		if strings.HasPrefix(line, "CPUID 00000000") {
			if anyfound {
				break
			}
		}
		if !strings.Contains(line, "-") {
			//continue
		}
		items := strings.Split(line, ":")
		if len(items) < 2 {
			if len(line) == 51 || len(line) == 50 {
				items = []string{line[0:14], line[15:]}
			} else {
				items = strings.Split(line, "\t")
				if len(items) != 2 {
					//fmt.Println("not found:", line, "len:", len(line))
					continue
				}
			}
		}
		items = items[0:2]
		vals := strings.Trim(items[1], "\r\n ")

		var idV uint32
		n, err := fmt.Sscanf(items[0], "CPUID %x", &idV)
		if err != nil || n != 1 {
			continue
		}
		existing, ok := fakeID[idV]
		if !ok {
			existing = make([][]uint32, 0)
		}

		values := make([]uint32, 4)
		n, err = fmt.Sscanf(vals, "%x-%x-%x-%x", &values[0], &values[1], &values[2], &values[3])
		if n != 4 || err != nil {
			n, err = fmt.Sscanf(vals, "%x %x %x %x", &values[0], &values[1], &values[2], &values[3])
			if n != 4 || err != nil {
				//fmt.Println("scanned", vals, "got", n, "Err:", err)
				continue
			}
		}

		existing = append(existing, values)
		fakeID[idV] = existing
		anyfound = true
	}

	restorer := func(f idfuncs) func() {
		return func() {
			cpuid = f.cpuid
			cpuidex = f.cpuidex
			xgetbv = f.xgetbv
		}
	}(idfuncs{cpuid: cpuid, cpuidex: cpuidex, xgetbv: xgetbv})

	cpuid = func(op uint32) (eax, ebx, ecx, edx uint32) {
		if op == 0x80000000 || op == 0 {
			var ok bool
			_, ok = fakeID[op]
			if !ok {
				return 0, 0, 0, 0
			}
		}
		first, ok := fakeID[op]
		if !ok {
			if op > maxFunctionID() {
				panic(fmt.Sprintf("Base not found: %v, request:%#v\n", fakeID, op))
			} else {
				// we have some entries missing
				return 0, 0, 0, 0
			}
		}
		theid := first[0]
		return theid[0], theid[1], theid[2], theid[3]
	}
	cpuidex = func(op, op2 uint32) (eax, ebx, ecx, edx uint32) {
		if op == 0x80000000 {
			var ok bool
			_, ok = fakeID[op]
			if !ok {
				return 0, 0, 0, 0
			}
		}
		first, ok := fakeID[op]
		if !ok {
			if op > maxExtendedFunction() {
				panic(fmt.Sprintf("Extended not found Info: %v, request:%#v, %#v\n", fakeID, op, op2))
			} else {
				// we have some entries missing
				return 0, 0, 0, 0
			}
		}
		if int(op2) >= len(first) {
			//fmt.Printf("Extended not found Info: %v, request:%#v, %#v\n", fakeID, op, op2)
			return 0, 0, 0, 0
		}
		theid := first[op2]
		return theid[0], theid[1], theid[2], theid[3]
	}
	xgetbv = func(index uint32) (eax, edx uint32) {
		first, ok := fakeID[1]
		if !ok {
			panic(fmt.Sprintf("XGETBV not supported %v", fakeID))
		}
		second := first[0]
		// ECX bit 26 must be set
		if (second[2] & 1 << 26) == 0 {
			panic(fmt.Sprintf("XGETBV not supported %v", fakeID))
		}
		// We don't have any data to return, unfortunately
		return 0, 0
	}
	return restorer
}

func TestMocks(t *testing.T) {
	zr, err := zip.OpenReader("testdata/cpuid_data.zip")
	if err != nil {
		t.Skip("No testdata:", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := ioutil.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		rc.Close()
		t.Log("Opening", f.FileInfo().Name())
		restore := mockCPU(content)
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
	}
	Detect()

}

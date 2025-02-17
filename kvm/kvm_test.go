package kvm_test

import (
	"errors"
	"math"
	"os"
	"syscall"
	"testing"
	"unsafe"

	"github.com/bobuhiro11/gokvm/kvm"
)

func TestGetAPIVersion(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	defer devKVM.Close()

	_, err = kvm.GetAPIVersion(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateVM(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	vmFd, err := kvm.CreateVM(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}

	if err := kvm.SetTSSAddr(vmFd); err != nil {
		t.Fatal(err)
	}

	if err := kvm.SetIdentityMapAddr(vmFd); err != nil {
		t.Fatal(err)
	}

	vcpuFd, err := kvm.CreateVCPU(vmFd, 0)
	if err != nil {
		t.Fatal(err)
	}

	CPUID := kvm.CPUID{}
	CPUID.Nent = 100

	if err := kvm.GetSupportedCPUID(devKVM.Fd(), &CPUID); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		CPUID.Entries[i].Eax = kvm.CPUIDFeatures
		CPUID.Entries[i].Ebx = 0x4b4d564b // KVMK
		CPUID.Entries[i].Ecx = 0x564b4d56 // VMKV
		CPUID.Entries[i].Edx = 0x4d       // M
	}

	if err := kvm.SetCPUID2(vcpuFd, &CPUID); err != nil {
		t.Fatal(err)
	}
}

func TestCPUID(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	defer devKVM.Close()

	_, err = kvm.CreateVM(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateVCPU(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	defer devKVM.Close()

	vmFd, err := kvm.CreateVM(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}

	err = kvm.CreateIRQChip(vmFd)
	if err != nil {
		t.Fatal(err)
	}

	err = kvm.CreatePIT2(vmFd)
	if err != nil {
		t.Fatal(err)
	}

	vcpuFd, err := kvm.CreateVCPU(vmFd, 0)
	if err != nil {
		t.Fatal(err)
	}

	sregs, err := kvm.GetSregs(vcpuFd)
	if err != nil {
		t.Fatal(err)
	}

	err = kvm.SetSregs(vcpuFd, sregs)
	if err != nil {
		t.Fatal(err)
	}

	regs, err := kvm.GetRegs(vcpuFd)
	if err != nil {
		t.Fatal(err)
	}

	err = kvm.SetRegs(vcpuFd, regs)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetVCPUMMapSize(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	defer devKVM.Close()

	_, err = kvm.GetVCPUMMmapSize(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateVCPUWithNoVmFd(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = kvm.CreateVCPU(devKVM.Fd(), 0)
	if err == nil {
		t.Fatal(err)
	}
}

// mirror from https://lwn.net/Articles/658512/
func TestAddNum(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	defer devKVM.Close()

	vmFd, err := kvm.CreateVM(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}

	mem, err := syscall.Mmap(-1, 0, 0x1000, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED|syscall.MAP_ANONYMOUS)
	if err != nil {
		t.Fatal(err)
	}

	code := []byte{0xba, 0xf8, 0x03, 0x00, 0xd8, 0x04, '0', 0xee, 0xb0, '\n', 0xee, 0xf4}
	copy(mem, code)

	if err = kvm.SetUserMemoryRegion(vmFd, &kvm.UserspaceMemoryRegion{
		Slot:          0,
		Flags:         0,
		GuestPhysAddr: 0x1000,
		MemorySize:    0x1000,
		UserspaceAddr: uint64(uintptr(unsafe.Pointer(&mem[0]))),
	}); err != nil {
		t.Fatal(err)
	}

	vcpuFd, err := kvm.CreateVCPU(vmFd, 0)
	if err != nil {
		t.Fatal(err)
	}

	mmapSize, err := kvm.GetVCPUMMmapSize(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}

	r, err := syscall.Mmap(int(vcpuFd), 0, int(mmapSize), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		t.Fatal(err)
	}

	run := (*kvm.RunData)(unsafe.Pointer(&r[0]))

	sregs, err := kvm.GetSregs(vcpuFd)
	if err != nil {
		t.Fatal(err)
	}

	sregs.CS.Base, sregs.CS.Selector = 0, 0

	if err = kvm.SetSregs(vcpuFd, sregs); err != nil {
		t.Fatal(err)
	}

	if err = kvm.SetRegs(vcpuFd, kvm.Regs{
		RIP: 0x1000, RAX: 2, RBX: 2, RFLAGS: 0x2, RCX: 0, RDX: 0, RSI: 0,
		RDI: 0, RSP: 0, RBP: 0, R8: 0, R9: 0, R10: 0, R11: 0, R12: 0,
		R13: 0, R14: 0, R15: 0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := kvm.SingleStep(vcpuFd, true); err != nil {
		t.Logf("kvm.SingleStep(%d, true): got %v, want nil", vcpuFd, err)
	}

	if err := kvm.SingleStep(vcpuFd, false); err != nil {
		t.Logf("kvm.SingleStep(%d, false): got %v, want nil", vcpuFd, err)
	}

	if err := kvm.SingleStep(uintptr(math.MaxUint), false); !errors.Is(err, syscall.EBADF) {
		t.Errorf("fixme:kvm.SingleStep(%d, false): got %v, want %v", vcpuFd, err, syscall.EBADF)
	}

	if err := kvm.SingleStep(vcpuFd, true); err != nil {
		t.Logf("kvm.SingleStep(%d, true): got %v, want nil", vcpuFd, err)
	}

	// While we are running the test, run it with SingleStep enabled,
	// so we can test that too.
	var singleStepOK bool

	for {
		if err = kvm.Run(vcpuFd); err != nil {
			t.Logf("kvm.Run(%d) returns with %v", vcpuFd, err)
		}

		switch kvm.ExitType(run.ExitReason) {
		case kvm.EXITHLT:
			if !singleStepOK {
				t.Errorf("singleStepOK: got false, want true; single step is not working")
			}

			return

		case kvm.EXITIO:
			direction, size, port, count, offset := run.IO()
			if direction == uint64(kvm.EXITIOOUT) && size == 1 && port == 0x3f8 && count == 1 {
				p := uintptr(unsafe.Pointer(run))
				c := *(*byte)(unsafe.Pointer(p + uintptr(offset)))
				t.Logf("output from IO port: \"%c\"\n", c)

				if c != '4' && c != '\n' {
					t.Fatal("Unexpected Output")
				}
			} else {
				t.Fatal("Unexpected KVM_EXIT_IO")
			}
		case kvm.EXITDEBUG:
			singleStepOK = true
		case kvm.EXITDCR,
			kvm.EXITINTR,
			kvm.EXITEXCEPTION,
			kvm.EXITFAILENTRY,
			kvm.EXITHYPERCALL,
			kvm.EXITINTERNALERROR,
			kvm.EXITIRQWINDOWOPEN,
			kvm.EXITMMIO,
			kvm.EXITNMI,
			kvm.EXITS390RESET,
			kvm.EXITS390SIEIC,
			kvm.EXITSETTPR,
			kvm.EXITSHUTDOWN,
			kvm.EXITTPRACCESS,
			kvm.EXITUNKNOWN:
			t.Fatalf("Unexpected EXIT REASON = %s\n", kvm.ExitType(run.ExitReason).String())
		default:
			t.Fatalf("Unexpected EXIT REASON = %s\n", kvm.ExitType(run.ExitReason).String())
		}
	}
}

func TestSetMemLogDirtyPages(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	u := kvm.UserspaceMemoryRegion{}
	u.SetMemLogDirtyPages()
	u.SetMemReadonly()

	if u.Flags != 0x3 {
		t.Fatal("unexpected flags")
	}
}

func TestIRQLine(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("Skipping test since we are not root")
	}

	t.Parallel()

	devKVM, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	vmFd, err := kvm.CreateVM(devKVM.Fd())
	if err != nil {
		t.Fatal(err)
	}

	if err := kvm.CreateIRQChip(vmFd); err != nil {
		t.Fatal(err)
	}

	if err := kvm.IRQLine(vmFd, 4, 0); err != nil {
		t.Fatal(err)
	}
}

func TestIoctlStringer(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		val  kvm.ExitType
		want string
	}{
		{name: "First error", val: kvm.EXITUNKNOWN, want: "EXITUNKNOWN"},
		{name: "Middle error", val: kvm.EXITIO, want: "EXITIO"},
		{name: "Last error", val: kvm.EXITINTERNALERROR, want: "EXITINTERNALERROR"},
		{name: "Out of range error", val: kvm.ExitType(1024), want: "ExitType(1024)"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := test.val.String()
			if got != test.want {
				t.Errorf("%s:%s != %s", test.name, test.want, got)
			}
		})
	}
}

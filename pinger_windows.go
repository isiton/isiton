package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"
)

type WinPdPinger struct {
	handle uintptr
}

func NewPinger() PdPinger {
	return &WinPdPinger{IcmpCreateFile()}
}

func (p *WinPdPinger) RunPing(hostname string) PingResult {
	result := PingResult{}
	result.Latency.Min = 1000.0

	var latencySum float64

	pings := 4
	errors := 0
	for i := 0; i < pings; i++ {
		if e, ok := IcmpSendEcho(p.handle, net.ParseIP(hostname)); ok {
			latencyValue := float64(e.RoundTripTime)
			latencySum += latencyValue
			if latencyValue > result.Latency.Max {
				result.Latency.Max = latencyValue
			}
			if latencyValue < result.Latency.Min {
				result.Latency.Min = latencyValue
			}
			time.Sleep(time.Second)
		} else {
			errors++
		}
	}
	result.PacketLoss = int64(errors) * 100 / int64(pings)
	result.Latency.Avg = latencySum / float64(pings)
	result.Online = true
	if result.PacketLoss == 100 {
		result.Online = false
	}
	return result
}

var (
	iphlpapi, _        = syscall.LoadLibrary("iphlpapi.dll")
	icmpCreateFile, _  = syscall.GetProcAddress(iphlpapi, "IcmpCreateFile")
	icmpCloseHandle, _ = syscall.GetProcAddress(iphlpapi, "IcmpCloseHandle")
	icmpSendEcho, _    = syscall.GetProcAddress(iphlpapi, "IcmpSendEcho")
)

func abort(funcname string, err error) {
	panic(fmt.Sprintf("%s failed: %v", funcname, err))
}

func IcmpCreateFile() (handle uintptr) {
	var nargs uintptr = 0
	if ret, _, callErr := syscall.Syscall(uintptr(icmpCreateFile), nargs, 0, 0, 0); callErr != 0 {
		abort("Call IcmpCreateFile", callErr)
	} else {
		handle = ret
	}
	return
}

func IcmpCloseHandle(handle uintptr) bool {
	var nargs uintptr = 1
	if ret, _, callErr := syscall.Syscall(uintptr(icmpCloseHandle), nargs, handle, 0, 0); callErr != 0 {
		abort("Call IcmpCloseHandle", callErr)
	} else {
		return ret == 1
	}
	return false
}

type ICMP_ECHO_REPLY32 struct {
	Address        [4]byte
	Status         uint32
	RoundTripTime  uint32
	DataSize       uint16
	Reserved       uint16
	DataPtr        uint64
	Ttl            uint8
	Tos            uint8
	Flags          uint8
	OptionsSize    uint8
	Unknown        uint32
	OptionsDataPtr uint64
}

func IcmpSendEcho(handle uintptr, ip net.IP) (ICMP_ECHO_REPLY32, bool) {
	var nargs uintptr = 8
	reqData := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	repData := make([]byte, 100)
	if ret, _, callErr := syscall.Syscall9(
		uintptr(icmpSendEcho),
		nargs,
		handle,
		uintptr(binary.LittleEndian.Uint32(ip.To4())),
		uintptr(unsafe.Pointer(&reqData[0])),
		uintptr(len(reqData)),
		0,
		uintptr(unsafe.Pointer(&repData[0])),
		uintptr(len(repData)),
		1000,
		0); ret == 0 && callErr != 0 && callErr != 11010 {
		//fmt.Printf("ERR %#v %d %d\n", callErr, callErr, ret)
		//abort("Call IcmpSendEcho", callErr)
		return ICMP_ECHO_REPLY32{}, false
	} else {
		//fmt.Printf("%s ret %d base %#v buf %#v\n", ip, ret, unsafe.Pointer(&repData[0]), repData)
		er := ICMP_ECHO_REPLY32{}
		if err := binary.Read(bytes.NewReader(repData), binary.LittleEndian, &er); err != nil {
			return ICMP_ECHO_REPLY32{}, false
		}
		//fmt.Printf("%#v\n%s\n", er, err)
		//fmt.Printf("TEST %s\n", net.IP(er.Address[:]))
		return er, true
	}
}

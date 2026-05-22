package misc_utils

import (
	"net"
	"strconv"
	"strings"
)

// IPAddrSplit convert string whose form is similar to `addr:port` into (addr, port)
//
//	return "", 0 if there is any error
func IPAddrSplit(ipPort string) (string, uint16) {
	resIp, strPort, err := net.SplitHostPort(ipPort)
	if err == nil {
		resPort, err := strconv.ParseUint(strPort, 10, 16)
		if err != nil {
			return "", 0
		}
		flag := net.ParseIP(resIp)
		if flag == nil {
			return "", 0
		}
		return resIp, uint16(resPort & 0xFFFF)
	}
	var (
		i         int
		tmpPort   uint64
		portDigit = map[string]struct{}{
			"0": {}, "1": {}, "2": {}, "3": {},
			"4": {}, "5": {}, "6": {}, "7": {},
			"8": {}, "9": {},
		}
		tmpIp string
	)
	for i = len(ipPort) - 1; i >= 0; i-- {
		_, ok := portDigit[string(ipPort[i])]
		if !ok {
			break
		}
	}
	if i >= 0 {
		tmpIp = ipPort[:i]
	}
	tmpPort, err = strconv.ParseUint(
		ipPort[min(i+1, max(len(ipPort)-1, 0)):],
		10, 16,
	)
	if err != nil {
		return "", 0
	}
	if len(tmpIp) > 0 && tmpIp[0] == '[' && tmpIp[len(tmpIp)-1] == ']' {
		tmpIp = tmpIp[1 : len(tmpIp)-1]
	}
	// check whether having zone
	if strings.Contains(tmpIp, "%") {
		slicer := strings.Split(tmpIp, "%")
		if len(slicer) != 2 {
			return "", 0
		}
		tmpIp, _ = slicer[0], slicer[1]
	}
	flag := net.ParseIP(tmpIp)
	if flag == nil {
		return "", 0
	}
	return tmpIp, uint16(tmpPort & 0xFFFF)
}

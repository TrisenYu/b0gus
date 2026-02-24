package misc_utils

import (
	"net"
	"strconv"
	"strings"
)

/*
addr:port => (addr, port)

	return "", 0 if any error emerges
*/
func IPaddrSplit(ip_port string) (string, uint16) {
	var res_ip string = ""
	res_ip, str_port, err := net.SplitHostPort(ip_port)
	if err == nil {
		res_port, err := strconv.ParseUint(str_port, 10, 16)
		if err != nil {
			return "", 0
		}
		flag := net.ParseIP(res_ip)
		if flag == nil {
			return "", 0
		}
		return res_ip, uint16(res_port & 0xFFFF)
	}
	var (
		i          int
		tmp_port   uint64
		port_digit = map[string]struct{}{
			"0": {}, "1": {}, "2": {}, "3": {},
			"4": {}, "5": {}, "6": {}, "7": {},
			"8": {}, "9": {},
		}
		tmp_ip string
	)
	for i = len(ip_port) - 1; i >= 0; i-- {
		_, ok := port_digit[string(ip_port[i])]
		if !ok {
			break
		}
	}
	if i >= 0 {
		tmp_ip = ip_port[:i]
	}
	tmp_port, err = strconv.ParseUint(
		ip_port[min(i+1, max(len(ip_port)-1, 0)):],
		10, 16,
	)
	if err != nil {
		return "", 0
	}
	if len(tmp_ip) > 0 && tmp_ip[0] == '[' && tmp_ip[len(tmp_ip)-1] == ']' {
		tmp_ip = tmp_ip[1 : len(tmp_ip)-1]
	}
	// check whether having zone
	if strings.Contains(tmp_ip, "%") {
		slicer := strings.Split(tmp_ip, "%")
		if len(slicer) != 2 {
			return "", 0
		}
		tmp_ip, _ = slicer[0], slicer[1]
	}
	flag := net.ParseIP(tmp_ip)
	if flag == nil {
		return "", 0
	}
	return tmp_ip, uint16(tmp_port & 0xFFFF)
}

package misc_utils

import (
	"net"
	"strconv"

	b0gus_config "b0gus/configs"
)

/*
addr:port => (addr, port)

	return "", 0 if any error emerges
*/
func IPaddrSplit(ip_port string) (string, uint16) {
	var (
		res_ip string
		i      int
		digits = map[string]any{
			"0": nil, "1": nil, "2": nil, "3": nil, "4": nil,
			"5": nil, "6": nil, "7": nil, "8": nil, "9": nil,
		}
	)
	for i = len(ip_port) - 1; i >= 0; i-- {
		_, ok := digits[string(ip_port[i])]
		if !ok {
			break
		}
	}
	res_ip = ip_port[:i]
	res_port, err := strconv.ParseUint(ip_port[min(i+1, len(ip_port)-1):], 10, 16)
	if err != nil {
		b0gus_config.Logger.Error(err, i, ip_port)
		return "", 0
	}
	flag := net.ParseIP(res_ip)
	if flag == nil {
		b0gus_config.Logger.Error(err, i, ip_port)
		return "", 0
	}
	return res_ip, uint16(res_port & 0xFFFF)
}

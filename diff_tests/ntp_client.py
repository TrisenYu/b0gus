#!/usr/bin/env python3
# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
# Last modified at 2026/03/02 星期一 15:55:26

# Currently don't need virtual environment
# because imported packages are standard.
from socket import  AF_INET6, SOCK_DGRAM, socket
from concurrent.futures import ThreadPoolExecutor
import struct, time

"""
	li			  2 bits, leap indicator for leap second
	vn			  3 bits, version number
	mode		  3 bits, communication mode

	stratum		 8 bits, stratum level of local clock
	poll			8 bits, maximum interval between successive message
	precision	   8 bits, precision of local clock

	root_delay	  32 bits, total round trip delay time
	root_dispersion 32 bits, max error aloud from primary clock source
	ref_ID		  32 bits, ref clock id

	ref_tm_s		32 bits, ref time-stamp second
	ref_tm_f		32 bits, ref time-stamp fraction
		if the local clock has never been synchronized, the reference timestamp is zero.

	ori_tm_s		32 bits, originate time-stamp second
	ori_tm_f		32 bits, originate time-stamp fraction
		specifying the local time at which the request departed for the service host.
		it will always have a nonzero value.

	rx_tm_s		32 bits, received time-stamp second
	rx_tm_f		32 bits, received time-stamp fraction

	tx_tm_s		32 bits, transmit time-stamp second
	tx_tm_f		32 bits, transmit time-stamp fraction
"""

addr = (
	# This domain represents a ntp server running in China
	# "ntp5.aliyun.com", # or replace with `localhost` and corresponding port number
	# 123
	"localhost",
	1234
)
FROM_1900_TO_1970 = 2208988800
get_LI = lambda x: x & 0b11
get_VN = lambda x: (x >> 2) & 0b111
get_MODE = lambda x: (x >> 5) & 0b111
set_LI = lambda x: x & 0b11
set_VN = lambda x: (x & 0b111) << 2
set_MODE = lambda x: (x & 0b111) << 5
# 011_011_00 ==> big-endian 00_011_011
"""
	mode value
	1: read status
	2: read variable
	3: write variable
	6: Set Trap Address/Port
	7: Trap Response
	9: Save Configuration
	10: Read MRU
	11: Read ordered list
	12: Request Nonce
	31: Unset Trap
	13-31: Reserved
"""
LI_VN_MODE = set_MODE(3) | set_VN(3) | set_LI(0)
# TODO: send fuzzed/malform packet
data = LI_VN_MODE.to_bytes(byteorder="big") + 47 * b"\x00" # big endian.
client = socket(AF_INET6, SOCK_DGRAM) # AF_INET leads to unexpected block

def sender(inp: bytes, remote: tuple[str, int]):
	global client
	client.sendto(inp, remote)

def receiver():
	global client
	try:
		resp, remote_ip = client.recvfrom(512)
		t = struct.unpack("!12I", resp[:min(48, len(data))])[10] - FROM_1900_TO_1970
		print(f"{time.ctime(t)} {t} from {remote_ip}")
	# Like this: `Thu Nov 27 20:19:48 2025`
	# Literally, t is a unix timestamp in the form of integer
	except struct.error:
		print("can not properly convert to 12 bytes int!")
		print(data)


# TODO: NTP-DDoS/relay/sniff.
with ThreadPoolExecutor(max_workers=2) as per_mission:
	per_mission.submit(sender, data, addr)
	per_mission.submit(receiver)

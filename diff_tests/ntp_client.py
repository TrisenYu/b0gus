#!/usr/bin/env python3
# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
from socket import AF_INET, SOCK_DGRAM, socket
import struct, time
"""
    li              2 bits, leap indicator for leap second
    vn              3 bits, version number
    mode            3 bits, communication mode

    stratum         8 bits, stratum level of local clock
    poll            8 bits, maximum interval between successive message
    precision       8 bits, precision of local clock

    root_delay      32 bits, total round trip delay time
    root_dispersion 32 bits, max error aloud from primary clock source
    ref_ID          32 bits, ref clock id

    ref_tm_s        32 bits, ref time-stamp second
    ref_tm_f        32 bits, ref time-stamp fraction
        if the local clock has never been synchronized, the reference timestamp is zero.

    ori_tm_s        32 bits, originate time-stamp second
    ori_tm_f        32 bits, originate time-stamp fraction
        specifying the local time at which the request departed for the service host.
        it will always have a nonzero value.

    rx_tm_s        32 bits, received time-stamp second
    rx_tm_f        32 bits, received time-stamp fraction

    tx_tm_s        32 bits, transmit time-stamp second
    tx_tm_f        32 bits, transmit time-stamp fraction
"""

addr = (
    # This domain represents a ntp server running in China
    "ntp5.aliyun.com", # or replace with `localhost` and corresponding port number
    123
) 
FROM_1900_TO_1970 = 2208988800
"""
    get_LI = lambda x: (x >> 6) & 0b11
    get_VN = lambda x: (x >> 3) & 0b111
    get_MODE = lambda x: (x & 0b111)
"""
li_vn_mode = 0b00_011_011

data = bytes.fromhex(hex(li_vn_mode)[2:])+47*b"\x00"
client = socket(AF_INET, SOCK_DGRAM)
client.sendto(data, addr)
data, addr = client.recvfrom(512)

try:
    t = struct.unpack("!12I", data[:min(48, len(data))])[10] - FROM_1900_TO_1970
    print(f"{time.ctime(t)} {t}")
    # Like this: `Thu Nov 27 20:19:48 2025`
    # Literally, t is a unix timestamp in the form of integer
except struct.error:
    print("can not properly convert to 12 bytes int!")
    print(data)


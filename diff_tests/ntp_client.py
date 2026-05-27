#!/usr/bin/env python3
# SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
# Last modified at 2026/05/23 星期六 23:36:46

# Currently don't need virtual environment
# because imported packages are standard.
import struct
import time
from concurrent.futures import ThreadPoolExecutor
from socket import AF_INET6, SOCK_DGRAM, socket
from threading import Semaphore

"""
    li              2 bits, leap indicator for leap second
    vn              3 bits, version number
    mode          3 bits, communication mode

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
    # "ntp5.aliyun.com", # or replace with `localhost` and corresponding port number
    # 123
    "localhost",
    1234
)
FROM_1900_TO_1970 = 2208988800
GET_LI = lambda x: x & 0b11
GET_VN = lambda x: (x >> 2) & 0b111
GET_MODE = lambda x: (x >> 5) & 0b111
SET_LI = lambda x: x & 0b11
SET_VN = lambda x: (x & 0b111) << 2
SET_MODE = lambda x: (x & 0b111) << 5
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
LI_VN_MODE = SET_MODE(3) | SET_VN(3) | SET_LI(0)
# TODO: send fuzzed/malform packet
data = LI_VN_MODE.to_bytes(byteorder="big") + 47 * b"\x00" # big endian.

class SockWrapper:
    def __init__(self):
        self.sock = socket(AF_INET6, SOCK_DGRAM)
        self.lock = Semaphore(1)

    def sender(self, inp: bytes, remote: tuple[str, int]):
        self.lock.acquire()
        self.sock.sendto(inp, remote)
        self.lock.release()

    def receiver(self):
        try:
            self.lock.acquire()
            resp, remote_ip = self.sock.recvfrom(512)
            self.lock.release()
            t = struct.unpack("!12I", resp[:min(48, len(data), len(resp))])[10] - FROM_1900_TO_1970
            print(f"{time.ctime(t)} {t} from {remote_ip}")
        # Like this: `Thu Nov 27 20:19:48 2025`
        # Literally, t is a unix timestamp in the form of integer
        except Exception as e:
            print("can not properly convert to 12 bytes int!", len(data), e)


# TODO: NTP-DDoS/relay/sniff.
sw = SockWrapper()
with ThreadPoolExecutor(max_workers=2) as per_mission:
    per_mission.submit(sw.sender, data, addr)
    per_mission.submit(sw.receiver)

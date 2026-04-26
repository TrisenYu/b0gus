Before starting the service, you must ensure that all ports required by the service are in an open state. This is a critical prerequisite to ensure the normal operation of the service and avoid connection failures caused by blocked ports.

Generally speaking, the port checking method follows the sequence from hardware to software. Access the control panel provided by your host or virtual machine provider, and check whether the port numbers you need are open. This step ensures that the external network can normally access the ports of the host or virtual machine.

After confirming the hardware-level port status, you need to check the software side, which is represented by the operating system. A commonly used tool on the software side is `firewalld`, which is written in Python and is widely used for managing system firewall rules and port status in *root privileges*.

Typically, `systemctl` can inspect and manage the status of `firewalld`.

```shell
# under root privileges as well.
systemctl status firewalld # check status
systemctl enable firewalld # enable firewalld after booting
systemctl start firewalld  # start up firewalld
systemctl reload firewalld # reload firewalld
systemctl stop firewalld   # stop firewalld
```

Run the command `firewall-cmd --list-ports` under root privileges to view all currently open ports. To check a specific port (e.g., port 8080), use `firewall-cmd --query-port=8080/tcp` (replace 8080/tcp with the required port and protocol). 

For persistently opening the port, execute these commands below.

```shell
firewall-cmd --add-port=8080/tcp --permanent
firewall-cmd --reload
```

For port forwarding, enable IP forwarding by command: `echo 1 > /proc/sys/net/ipv4/ip_forward`.
To make it persistent, modify `/etc/sysctl.conf` and set `net.ipv4.ip_forward=1`, then run `sysctl -p`.

Then execute the following commands to set port forwarding rules.

```shell
## for ssh
iptables -t nat -A PREROUTING -p tcp --dport 22 -j REDIRECT --to-port 2222
## for ntp
iptables -t nat -A PREROUTING -p udp --dport 123 -j REDIRECT --to-port 1234
## for ftp
iptables -t nat -A PREROUTING -p tcp --dport 21 -j REDIRECT --to-port 2121
## for smtp
iptables -t nat -A PREROUTING -p tcp --dport 25 -j REDIRECT --to-port 2525
## for dns
iptables -t nat -A PREROUTING -p tcp --dport 53 -j REDIRECT --to-port 5353
## for http
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
```

Note: 
1. All port forwarding operations above require root privileges; 
2. Do not use firewalld and iptables at the same time to avoid rule conflicts; 
3. The non-privileged ports selected (2222, 2121, etc.) can be modified according to actual demands, but ensure that they are not occupied by other services.

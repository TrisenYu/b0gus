开启服务前，必须要确保所有服务所需的端口处于**开启状态**。不管是主机内防火墙一侧或者是主机外的防火墙一侧。

通常确认方法是从硬件侧到软件侧。对于硬件侧，访问主机服务提供商的控制面板，检查所需端口号有无放行。
在确认无误后，检查软件侧防火墙的放行情况，而这视操作系统而异。
在Linux下，通常可以用`firewalld`来检查。而有没有`firewalld`又可以用`systemctl`来检查。

```shell
# # 这里的操作涉及root权限。
# systemctl status firewalld # 状态
# systemctl enable firewalld # 防火墙使能情况
# systemctl start firewalld  # 开启防火墙服务
# systemctl reload firewalld # 重载防火墙服务
# systemctl stop firewalld   # 停止防火墙
```

防火墙正常运行时，可以在root权限下通过`firewall-cmd --list-ports`来看当前所有暴露于公网的端口。
如果要检查某个特定端口，比如8080，可以用`firewall-cmd --query-port=8080/tcp`来检查。

而想持久化某个端口状态时，可以执行下面的命令。
```shell
# firewall-cmd --add-port=8080/tcp --permanent
# firewall-cmd --reload
```
对于端口转发，使能命令是：`echo 1 > /proc/sys/net/ipv4/ip_forward`。
ipv6则是：`echo 1 > /proc/sys/net/ipv6/conf/all/forwarding`。
持久化方式则是修改`/etc/sysctl.conf`并设置`net.ipv4.ip_forward=1`，然后运行`sysctl -p`。
> 持久化ipv6则是要设置`net.ipv6.conf.all.forwarding=1`。

而下列的命令就是针对服务来启用的具体转发规则。

```shell
## for ssh
# iptables -t nat -A PREROUTING -p tcp --dport 22 -j REDIRECT --to-port 2222
## for ntp
# iptables -t nat -A PREROUTING -p udp --dport 123 -j REDIRECT --to-port 1234
## for ftp
# iptables -t nat -A PREROUTING -p tcp --dport 21 -j REDIRECT --to-port 2121
## for smtp
# iptables -t nat -A PREROUTING -p tcp --dport 25 -j REDIRECT --to-port 2525
## for dns
# iptables -t nat -A PREROUTING -p tcp --dport 53 -j REDIRECT --to-port 5353
## for http
# iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
```

也可以用firewalld来完成这一操作。不过，firewalld端口转发必须开启NAT伪装。
```shell
# firewall-cmd --permanent --add-masquerade
```
转发的写法类似于
```shell
## 添加 TCP 端口转发规则
# firewall-cmd --permanent --add-forward-port=port=10001:proto=tcp:toaddr=1.2.3.4:toport=30553
## 添加 UDP 端口转发规则
# firewall-cmd --permanent --add-forward-port=port=10001:proto=udp:toaddr=5.6.7.8:toport=30553
```

注意： 
1. 上面的操作都要在root权限下完成。
2. 不应该同时用iptable和firewalld配置端口转发。 
3. 可以按实际需求，改选你想要的非特权端口，但要确保这些端口没有被其他原有的服务所占用。

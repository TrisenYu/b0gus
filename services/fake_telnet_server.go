// just a demo for references, have not integrated with database operations yet
package services

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type TelnetClient struct {
	IP               string
	Port             string
	IsAuthentication bool
	UserName         string
	Password         string
}

const (
	TIME_DELAY_AFTER_WRITE = 500 // 500ms
)

var g_WriteChan chan string

func main() {
	g_WriteChan = make(chan string)
	telnetClientObj := new(TelnetClient)
	telnetClientObj.IP = "192.168.1.104"
	telnetClientObj.Port = "23"
	telnetClientObj.IsAuthentication = false
	// telnetClientObj.UserName = "userOne"
	// telnetClientObj.Password = "123456"
	// fmt.Println(telnetClientObj.PortIsOpen(5))
	go telnetClientObj.Telnet(20)

	for {
		line := readLine()
		g_WriteChan <- string(line)
	}
}

func readLine() string {
	// fmt.Print("> ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	return strings.TrimSpace(line)
}

func (this *TelnetClient) PortIsOpen(timeout int) bool {
	raddr := this.IP + ":" + this.Port
	conn, err := net.DialTimeout("tcp", raddr, time.Duration(timeout)*time.Second)
	if nil != err {
		log.Println("pkg: model, func: PortIsOpen, method: net.DialTimeout, errInfo:", err)
		return false
	}
	defer conn.Close()
	return true
}

func (this *TelnetClient) Telnet(timeout int) (err error) {
	raddr := this.IP + ":" + this.Port
	conn, err := net.DialTimeout("tcp", raddr, time.Duration(timeout)*time.Second)
	if nil != err {
		log.Println("pkg: model, func: Telnet, method: net.DialTimeout, errInfo:", err)
		return
	}
	defer conn.Close()
	if false == this.telnetProtocolHandshake(conn) {
		log.Println("pkg: model, func: Telnet, method: this.telnetProtocolHandshake, errInfo: telnet protocol handshake failed!!!")
		return
	}
	go func() {
		for {
			data := make([]byte, 1024)
			_, err := conn.Read(data)
			if err != nil {
				fmt.Println(err)
				break
			}
			// strData := string(data)
			fmt.Printf("%s", data)
		}
	}()

	//	conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	for {
		select {
		case cmd, _ := <-g_WriteChan:
			_, err = conn.Write([]byte(cmd + "\n"))
			if nil != err {
				log.Println("pkg: model, func: Telnet, method: conn.Write, errInfo:", err)
				return
			}
			fmt.Println("Quit telnet!!!!!!")
			return
		default:
			time.Sleep(100 * time.Millisecond)
			continue
		}
	}
}

func (this *TelnetClient) telnetProtocolHandshake(conn net.Conn) bool {
	var buf [4096]byte
	n, err := conn.Read(buf[0:])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Read, errInfo:", err)
		return false
	}
	// fmt.Println(string(buf[0:n]))
	// fmt.Println((buf[0:n]))

	buf[1] = 252
	buf[4] = 252
	buf[7] = 252
	buf[10] = 252
	// fmt.Println((buf[0:n]))
	n, err = conn.Write(buf[0:n])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Write, errInfo:", err)
		return false
	}

	n, err = conn.Read(buf[0:])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Read, errInfo:", err)
		return false
	}
	// fmt.Println(string(buf[0:n]))
	// fmt.Println((buf[0:n]))

	buf[1] = 252
	buf[4] = 251
	buf[7] = 252
	buf[10] = 254
	buf[13] = 252
	fmt.Println((buf[0:n]))
	n, err = conn.Write(buf[0:n])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Write, errInfo:", err)
		return false
	}

	n, err = conn.Read(buf[0:])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Read, errInfo:", err)
		return false
	}
	// fmt.Println(string(buf[0:n]))
	// fmt.Println((buf[0:n]))

	buf[1] = 252
	buf[4] = 252
	// fmt.Println((buf[0:n]))
	n, err = conn.Write(buf[0:n])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Write, errInfo:", err)
		return false
	}

	n, err = conn.Read(buf[0:])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Read, errInfo:", err)
		return false
	}
	// fmt.Println(string(buf[0:n]))
	// fmt.Println((buf[0:n]))

	if false == this.IsAuthentication {
		return true
	}

	n, err = conn.Write([]byte(this.UserName + "\n"))
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Write, errInfo:", err)
		return false
	}
	time.Sleep(time.Millisecond * TIME_DELAY_AFTER_WRITE)

	n, err = conn.Read(buf[0:])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Read, errInfo:", err)
		return false
	}
	// fmt.Println(string(buf[0:n]))

	n, err = conn.Write([]byte(this.Password + "\n"))
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Write, errInfo:", err)
		return false
	}
	time.Sleep(time.Millisecond * TIME_DELAY_AFTER_WRITE)

	n, err = conn.Read(buf[0:])
	if nil != err {
		log.Println("pkg: model, func: telnetProtocolHandshake, method: conn.Read, errInfo:", err)
		return false
	}
	// fmt.Println(string(buf[0:n]))
	return true
}

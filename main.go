package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

var chanIP = make(chan string, 200)
var chanSCan = make(chan int, 200)
var chanWait = make(chan int, 1)

func main() {
	startIP := flag.String("i", "127.0.0.1", "扫描的起始IP地址,默认为本机地址")
	endIP := flag.String("e", "", "结束扫描的IP地址,可为空, 指定此项,-n无效")
	scanNumber := flag.Uint("n", 1, "扫描的ip数,默认为1")
	startPort := flag.Uint("p", 1, "指定扫描的起始端口,默认为值1")
	endPort := flag.Uint("f", 65535, "指定扫描的结束端口, 默认值为65535")
	flag.Parse()
	//检测输入的IP是否正确
	si := net.ParseIP(*startIP)
	ei := si
	if si == nil {
		log.Fatal("输入的起始IP不正确,请重新输入")
	}
	if *endIP != "" {
		ei = net.ParseIP(*endIP)
		if ei == nil {
			log.Fatal("输入的结束IP不正确")
		}
	}
	if *endPort > 65535 {
		*endPort = 65535
	}
	if *startPort < 1 {
		*startPort = 1
	}
	if *startPort > *endPort {
		log.Fatal("扫描起始端口大于结束端口,请重新输入")
	}

	fmt.Println(*scanNumber, si, ei)
	//启动解析ip进程
	go calculateIP(si, ei)
	//启动扫描进程
	go scanIP(*startPort, *endPort)
	<-chanWait
}

//扫描进程
func scanIP(startPort, endPort uint) {
	for IP := range chanIP {
		var result = make([]string, 0)
		var buffer bytes.Buffer
		buffer.WriteString(IP)
		buffer.WriteString(":")
		for index := startPort; index <= endPort; index++ {
			chanSCan <- 1
			func(newBuffer bytes.Buffer, newIndex uint) {
				defer func() {
					<-chanSCan
				}()
				newBuffer.WriteString(strconv.Itoa(int(newIndex)))
				fmt.Println(">>>>>>>>", newBuffer.String())
				_, err := net.Dial("tcp", "120.78.76.139:1234")
				if err == nil {
					result = append(result, newBuffer.String())
				}
			}(buffer, index)
		}
		fmt.Println(result)
	}
	close(chanWait)
}

//解析IP进程
func calculateIP(si, ei net.IP) {
	siString := si.String()
	eiString := ei.String()
	if siString > eiString {
		siString, eiString = eiString, siString
	}
	for ; siString != eiString; siString = nextIP(siString) {
		if si = net.ParseIP(siString); si == nil {
			continue
		}
		chanIP <- siString
	}
	chanIP <- siString
	close(chanIP)
}

//获取下一个IP
func nextIP(ip string) string {
	ips := strings.Split(ip, ".")
	var i int
	for i = len(ips) - 1; i >= 0; i-- {
		n, _ := strconv.Atoi(ips[i])
		if n >= 255 {
			//进位
			ips[i] = "1"
		} else {
			//+1
			n++
			ips[i] = strconv.Itoa(n)
			break
		}
	}
	ip = ""
	leng := len(ips)
	for i := 0; i < leng; i++ {
		if i == leng-1 {
			ip += ips[i]
		} else {
			ip += ips[i] + "."
		}
	}
	return ip
}

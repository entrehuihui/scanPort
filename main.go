package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var chanIP = make(chan string, 200)

//限制线程数
var chanSCan = make(chan int, 200)
var chanWait = make(chan int, 1)
var chanResult = make(chan []string, 20)

func main() {
	startIP := flag.String("i", "127.0.0.1", "扫描的起始IP地址,默认为本机地址")
	endIP := flag.String("e", "", "结束扫描的IP地址,可为空, 指定此项,-n无效")
	scanNumber := flag.Uint("n", 1, "扫描的ip数,默认为1")
	startPort := flag.Uint("p", 1, "指定扫描的起始端口,默认为值1")
	endPort := flag.Uint("f", 65535, "指定扫描的结束端口, 默认值为65535")
	file := flag.String("l", "resulkt.log", "默认日志记录文档")
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

	fmt.Println(*scanNumber)
	//启动解析ip进程
	go calculateIP(si, ei)
	//启动扫描进程
	go scanIP(*startPort, *endPort)
	//启动日志文件
	go WriteLog(*file)
	<-chanWait
}

//扫描进程
func scanIP(startPort, endPort uint) {
	var wg sync.WaitGroup
	for IP := range chanIP {
		var result = make([]string, 0)
		var buffer bytes.Buffer
		buffer.WriteString(IP)
		buffer.WriteString(":")
		for index := startPort; index <= endPort; index++ {
			chanSCan <- 1
			wg.Add(1)
			go func(ip string, newIndex uint) {
				var newBuffer bytes.Buffer
				defer func() {
					wg.Done()
					<-chanSCan
				}()
				newBuffer.WriteString(ip)
				newBuffer.WriteString(strconv.Itoa(int(newIndex)))
				fmt.Println("SCAN IP PORT :", newBuffer.String())
				_, err := net.Dial("tcp4", newBuffer.String())
				if err == nil {
					result = append(result, newBuffer.String())
				}
			}(buffer.String(), index)
		}
		wg.Wait()
		chanResult <- result
	}
	//已经扫描完成所有ip,关闭写入文件通道
	close(chanResult)
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
	//解析完成所有ip,关闭ip通道
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

//写入日志
func WriteLog(file string) {
	//读写创建末尾追加打开文件
	fd, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("log error", err)
	}
	defer fd.Close()
	//绑定输出到文件/输出到控制台为同一个输出端口
	mw := io.MultiWriter(os.Stdout, fd)
	logs := log.New(mw, "[SCAN SUCCESS PORT]", log.LstdFlags)
	for body := range chanResult {
		for _, data := range body {
			logs.Print(data)
		}
	}
	//写入完成,关闭等待通道
	close(chanWait)
}

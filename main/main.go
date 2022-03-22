package main

import (
	"encoding/json"
	"fmt"
	"log"
	"mrpc"
	"mrpc/codec"
	"net"
	"time"
)

func startServer(addrChan chan string) {
	l, err := net.Listen("tcp", ":8088")
	if err != nil {
		log.Panic("network error: ", err)
	}
	addrChan <- l.Addr().String()
	mrpc.Accept(l)
}

func main() {
	var addrChan chan string
	go startServer(addrChan)
	conn, err := net.Dial("tcp", <-addrChan)
	defer conn.Close()
	if err != nil {
		log.Panic("dail server error: ", err)
	}
	time.Sleep(time.Second)

	json.NewEncoder(conn).Encode(mrpc.DefaultOption)

	client := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Service.Add",
			Seq:           1,
		}
		client.Write(h, fmt.Sprintf("req body,seq:%v", h.Seq))
		client.ReadHeader(h)
		var reply string
		client.ReadBody(&reply)
		fmt.Printf("header:%v,body:%v", h, reply)
	}

}

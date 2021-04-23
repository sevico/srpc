package main

import (
	"fmt"
	"log"
	"net"
	"srpc"
	"sync"
	"time"
)

func startServer(addr chan string){
	l,err:=net.Listen("tcp",":0")
	if err!=nil{
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	srpc.Accept(l)
}
//func main() {
//	addr:=make(chan string)
//	go startServer(addr)
//
//	conn,_:=net.Dial("tcp",<-addr)
//	defer func() { _ = conn.Close() }()
//	time.Sleep(time.Second)
//
//	_=json.NewEncoder(conn).Encode(srpc.DefaultOption)
//	cc := codec.NewGobCodec(conn)
//	for i := 0; i < 5; i++{
//		h:=&codec.Header{
//			ServiceMethod: "Foo.Sum",
//			Seq: uint64(i),
//		}
//		_= cc.Write(h,fmt.Sprintf("srpc req %d",h.Seq))
//		_=cc.ReadHeader(h)
//		reply:=""
//		_=cc.ReadBody(&reply)
//		log.Println("reply: ",reply)
//
//
//	}
//
//}

func main() {
	log.SetFlags(0)
	addr:=make(chan string)
	go startServer(addr)
	client,_:=srpc.Dial("tcp",<-addr)

	time.Sleep(time.Second)
	wg:=sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args:=fmt.Sprintf("srpc req %d",i)
			reply:=""
			if err:=client.Call("Foo.Sum",args,&reply);err!=nil{
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}

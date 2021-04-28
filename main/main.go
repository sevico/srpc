package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"srpc"
	"srpc/registry"
	"srpc/xclient"
	"sync"
	"time"
)

//func startServer(addr chan string){
//	l,err:=net.Listen("tcp",":0")
//	if err!=nil{
//		log.Fatal("network error:", err)
//	}
//	log.Println("start rpc server on", l.Addr())
//	addr <- l.Addr().String()
//	srpc.Accept(l)
//}
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

type Foo int
type Args struct {
	Num1,Num2 int
}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1+args.Num2
	return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

//
//func startServer(addr chan string){
//	var foo Foo
//	if err:=srpc.Register(&foo);err!=nil{
//		log.Fatal("register error:", err)
//	}
//
//	l,err:=net.Listen("tcp",":9999")
//	if err!=nil{
//		log.Fatal("network error:", err)
//	}
//
//	log.Println("start rpc server on", l.Addr())
//	srpc.HandleHTTP()
//	addr <- l.Addr().String()
//	_ = http.Serve(l,nil)
//
//	//addr <- l.Addr().String()
//	//srpc.Accept(l)
//}


func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	server := srpc.NewServer()
	_ = server.Register(&foo)
	registry.Heartbeat(registryAddr,"tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l)
}

func foo(xc *xclient.XClient,ctx context.Context,typ , serviceMethod string,args *Args){
	reply:=0
	var err error
	switch typ {
	case "call":
		err = xc.Call(ctx,serviceMethod,args,&reply)
	case "broadcast":
		err = xc.Broadcast(ctx,serviceMethod,args,&reply)
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}
func call(registry string){
	d:=xclient.NewSRegistryDiscovery(registry,0)
	xc:=xclient.NewXClient(d,xclient.RandomSelect,nil)

	defer func() { _ = xc.Close() }()
	wg:=sync.WaitGroup{}
	for i:=0;i<5;i++{
		wg.Add(1)
		func(i int) {
			defer wg.Done()
			fmt.Println(i)
			foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast(registry string) {
	d:=xclient.NewSRegistryDiscovery(registry,0)
	xc:=xclient.NewXClient(d,xclient.RandomSelect,nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}


//
//func call(addrCh chan string){
//	client, _ := srpc.DialHTTP("tcp", <-addrCh)
//	defer func() { _ = client.Close() }()
//
//	time.Sleep(time.Second)
//	// send request & receive response
//	var wg sync.WaitGroup
//	for i := 0; i < 5; i++ {
//		wg.Add(1)
//		go func(i int) {
//			defer wg.Done()
//			args := &Args{Num1: i, Num2: i * i}
//			var reply int
//			if err := client.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
//				log.Fatal("call Foo.Sum error:", err)
//			}
//			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
//		}(i)
//	}
//	wg.Wait()
//
//}

func startRegistry(wg *sync.WaitGroup){
	l,_:=net.Listen("tcp",":9998")
	registry.HandleHTTP()
	wg.Done()
	_=http.Serve(l,nil)
}



func main() {
	log.SetFlags(0)
	registryAddr := "http://localhost:9998/_geerpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startServer(registryAddr, &wg)
	go startServer(registryAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	call(registryAddr)
	broadcast(registryAddr)
}

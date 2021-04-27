package xclient

import (
	"context"
	"reflect"
	"srpc"
	"sync"
)

type XClient struct{
	d Discovery
	mode SelectMode
	opt *srpc.Option
	mu sync.Mutex  // protect following
	clients map[string]*srpc.Client
}

func NewXClient(d Discovery,mode SelectMode,opt *srpc.Option) *XClient{
	return &XClient{
		d:d,
		mode:mode,
		opt:opt,
		clients: make(map[string]*srpc.Client),
	}
}

func (c *XClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key,client:=range c.clients{
		_=client.Close()
		delete(c.clients,key)
	}
	return nil
}

func (c *XClient) dial(rpcAddr string) (*srpc.Client,error){
	c.mu.Lock()
	defer c.mu.Unlock()
	client,ok:=c.clients[rpcAddr]
	if ok && !client.IsAvailable(){
		_ = client.Close()
		delete(c.clients,rpcAddr)
		client=nil
	}
	if client==nil{
		var err error
		client,err = srpc.XDial(rpcAddr,c.opt)
		if err!=nil{
			return nil,err
		}
		c.clients[rpcAddr] = client
	}
	return client,nil
}
func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	client, err := xc.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// xc will choose a proper server.
func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	rpcAddr, err := xc.d.Get(xc.mode)
	if err != nil {
		return err
	}
	return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}

// Broadcast invokes the named function for every server registered in discovery
func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	servers,err:=xc.d.GetAll()
	if err!=nil{
		return err
	}

	wg:=sync.WaitGroup{}
	mu:=sync.Mutex{}
	var e error

	replySet:=reply==nil  // if reply is nil, don't need to set value
	ctx,cancel := context.WithCancel(ctx)
	for _,rpcAddr := range servers{
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var cloneReply interface{}
			if reply!=nil{
				cloneReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := xc.call(rpcAddr,ctx,serviceMethod,args,cloneReply)
			mu.Lock()
			if err!=nil && e==nil{
				e=err
				cancel()
			}
			if err==nil && !replySet{
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(cloneReply).Elem())
				replySet =true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}
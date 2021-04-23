package srpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"srpc/codec"
	"sync"
)

const MagicNumber = 0x3beef
type Option struct{
	MagicNumber int
	CodecType codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType: codec.GobType,
}


type Server struct{}

func NewServer() *Server{
	return &Server{}
}

var DefaultServer = NewServer()

func (s *Server) Accept(lis net.Listener) {
	for true {
		conn,err:=lis.Accept()
		if err!=nil{
			log.Println("rpc server: accept error:", err)
			return
		}
		go s.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()
	opt:=Option{}

	if err:=json.NewDecoder(conn).Decode(&opt);err!=nil{
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber{
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f:=codec.NewCodecFuncMap[opt.CodecType]
	if f==nil{
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending:=&sync.Mutex{}
	wg:=&sync.WaitGroup{}
	for{
		req,err:=server.readRequest(cc)
		if err!=nil{
			if req==nil{
				break
			}
			req.h.Error=err.Error()
			server.sendResponse(cc,req.h,invalidRequest,sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_=cc.Close()
}

type request struct{
	h *codec.Header
	argV,replyV reflect.Value
}

func (server *Server) readReuestHeader(cc codec.Codec) (*codec.Header,error) {
	h := codec.Header{}
	if err:=cc.ReadHeader(&h);err!=nil{
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h,nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h,err:=server.readReuestHeader(cc)
	if err!=nil{
		return nil, err
	}
	req:=&request{h: h}
	req.argV = reflect.New(reflect.TypeOf(""))
	if err=cc.ReadBody(req.argV.Interface());err!=nil{
		log.Println("rpc server: read argv err:", err)
	}
	return req,nil
}

func (server *Server) sendResponse(cc codec.Codec,h *codec.Header,body interface{},sending *sync.Mutex)  {
	sending.Lock()
	defer sending.Unlock()
	if err:=cc.Write(h,body);err!=nil{
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec,req *request,sending *sync.Mutex,wg *sync.WaitGroup)  {
	defer wg.Done()
	log.Println(req.h, req.argV.Elem())
	req.replyV = reflect.ValueOf(fmt.Sprintf("srpc resp %d", req.h.Seq))
	server.sendResponse(cc,req.h,req.replyV.Interface(),sending)
}
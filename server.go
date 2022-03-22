package mrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mrpc/codec"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x1b2b3c

type Option struct {
	MagicNumber int
	CodecType   codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (server *Server) Accept(listener net.Listener) {

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Panicln("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {

	var opt Option
	err := json.NewDecoder(conn).Decode(&opt)
	if err != nil {
		log.Println("rpc server: options decode error:", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number:%v", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Println("rpc server: invalid codec type:", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handRequest(cc, req, sending, wg)
	}
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var head codec.Header
	err := cc.ReadHeader(&head)
	if err != nil {
		log.Println("rpc server: ", err)
		return nil, err
	}
	return &head, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		log.Println("rpc server: ", err)
	}
	req := &request{
		h:    h,
		argv: reflect.New(reflect.TypeOf("")),
	}
	err = cc.ReadBody(req.argv.Interface())
	if err != nil {
		log.Println("rpc server: ", err)
	}
	return req, nil
}
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("prc server: ", err)
	}
}

func (server *Server) handRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h)
	log.Println(req.argv.Elem())

	req.replyv = reflect.ValueOf(fmt.Sprintf("grpc resp %v", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)

}

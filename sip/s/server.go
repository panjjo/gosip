package sip

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/panjjo/gosip/utils"
	"github.com/sirupsen/logrus"
)

var (
	bufferSize uint16 = 65535 - 20 - 8 // IPv4 max size - IPv4 Header size - UDP Header size
)

// RequestHandler RequestHandler
type RequestHandler func(req *Request, tx *Transaction)

// Server Server
type Server struct {
	udpaddr net.Addr
	tcpaddr net.Addr
	conn    Connection
	txs     *transacionts

	hmu             *sync.RWMutex
	requestHandlers map[RequestMethod]RequestHandler

	port *Port
	host net.IP

	tcpPort     *Port
	tcpHost     net.IP
	tcpListener *net.TCPListener

	parser *parser
}

// NewServer NewServer
func NewServer() *Server {
	activeTX = &transacionts{txs: map[string]*Transaction{}, rwm: &sync.RWMutex{}}
	srv := &Server{hmu: &sync.RWMutex{},
		txs:             activeTX,
		requestHandlers: map[RequestMethod]RequestHandler{}}
	return srv
}

func (s *Server) getTX(key string) *Transaction {
	return s.txs.getTX(key)
}

func (s *Server) mustTX(msg *Request) *Transaction {
	key := getTXKey(msg)
	tx := s.txs.getTX(key)
	if tx == nil {
		if msg.conn.Network() == "UDP" {
			tx = s.txs.newTX(key, s.conn)
		} else {
			tx = s.txs.newTX(key, msg.conn)
		}
	}
	return tx
}

func (s *Server) StartParser() {
	parser := newParser()
	s.parser = parser

	go parser.start()
	defer parser.stop()
	s.handlerListen(parser.out)
}

func (s *Server) ListenUDPServer(addr string) {
	udpaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logrus.Fatal("net.ResolveUDPAddr err", err, addr)
	}
	s.udpaddr = udpaddr
	s.port = NewPort(udpaddr.Port)
	s.host, err = utils.ResolveSelfIP()
	if err != nil {
		logrus.Fatal("net.ListenUDP resolveip err", err, addr)
	}
	udp, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		logrus.Fatal("net.ListenUDP err", err, addr)
	}
	s.conn = newUDPConnection(udp)
	var (
		raddr net.Addr
		num   int
	)
	buf := make([]byte, bufferSize)

	for {
		num, raddr, err = s.conn.ReadFrom(buf)
		if err != nil {
			logrus.Errorln("udp.ReadFromUDP err", err)
			continue
		}
		logrus.Println("udp raddr:", raddr)
		s.parser.in <- newPacket(append([]byte{}, buf[:num]...), raddr, s.conn)
	}
}

func (s *Server) ListenTCPServer(addr string) {
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logrus.Fatal("net.ResolveTCPAddr err", err, addr)
	}
	s.tcpaddr = tcpaddr
	s.tcpPort = NewPort(tcpaddr.Port)
	s.tcpHost, err = utils.ResolveSelfIP()
	if err != nil {
		logrus.Fatal("net.ListenTCP resolveip err", err, addr)
	}

	tcp, err := net.ListenTCP("tcp", tcpaddr)
	if err != nil {
		logrus.Fatal("net.ListenTCP err", err, addr)
	}
	defer tcp.Close()

	s.tcpListener = tcp

	for {
		conn, err := tcp.Accept()
		if err != nil {
			logrus.Warn("net.ListenTCP err", err, addr)
			continue
		}

		go s.processTcpConn(conn)
	}
}

func (s *Server) processTcpConn(conn net.Conn) {
	defer conn.Close() // 关闭连接
	reader := bufio.NewReader(conn)
	// lenBuf := make([]byte, 2)
	c := newTCPConnection(conn)
	{
		// rtp over tcp reader process.
		// 2byte len version.
		// for {
		// 	_, err := io.ReadFull(reader, lenBuf)
		// 	if err != nil {
		// 		break
		// 	}
		// 	len := 0
		// 	len = (len & (int)(lenBuf[0]))
		// 	len = ((len << 8) | (int)(lenBuf[1]))
		// 	buf := make([]byte, len)

		// 	n, err := io.ReadFull(reader, buf)
		// 	if err != nil || n != len {
		// 		break
		// 	}

		// 	s.parser.in <- newPacket(buf, conn.RemoteAddr(), c)
		// }
	}

	for {
		var buffer bytes.Buffer
		bodyLen := 0
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				logrus.Debugln("tcp conn read error:", err)
				return
			}

			buffer.Write(line)

			if len(line) <= 2 {
				if bodyLen <= 0 {
					break
				}

				// read body
				bodyBuf := make([]byte, bodyLen)
				n, err := io.ReadFull(reader, bodyBuf)
				if err != nil || n != bodyLen {
					logrus.Errorln("error while read full, err:", err)
					// err process
				}
				buffer.Write(bodyBuf)
				break
			}

			if strings.Contains(string(line), "Content-Length") {
				s := strings.Split(string(line), ":")
				value := strings.Trim(s[len(s)-1], " \r\n")
				bodyLen, err = strconv.Atoi(value)
				if err != nil {
					logrus.Errorln("parse Content-Length failed")
					break
				}
			}
		}

		s.parser.in <- newPacket(buffer.Bytes(), conn.RemoteAddr(), c)
	}
}

// RegistHandler RegistHandler
func (s *Server) RegistHandler(method RequestMethod, handler RequestHandler) {
	s.hmu.Lock()
	s.requestHandlers[method] = handler
	s.hmu.Unlock()
}

func (s *Server) handlerListen(msgs chan Message) {
	var msg Message
	for {
		msg = <-msgs
		switch tmsg := msg.(type) {
		case *Request:
			req := tmsg
			dst := s.udpaddr
			logrus.Println("handlerListen net:", req.conn.Network())

			// if req.source.Network() == "tcp" {
			if req.conn.Network() == "TCP" {
				dst = s.tcpaddr
			}
			req.SetDestination(dst)
			s.handlerRequest(req)
		case *Response:
			resp := tmsg

			logrus.Println("handlerListen net:", resp.conn.Network())

			dst := s.udpaddr
			// if resp.source.Network() == "tcp" {
			if resp.conn.Network() == "TCP" {
				dst = s.tcpaddr
			}

			resp.SetDestination(dst)
			s.handlerResponse(resp)
		default:
			logrus.Errorln("undefind msg type,", tmsg, msg.String())
		}
	}
}

func (s *Server) handlerRequest(msg *Request) {
	tx := s.mustTX(msg)
	logrus.Traceln("receive request from:", msg.Source(), ",method:", msg.Method(), "txKey:", tx.key, "message: \n", msg.String())
	s.hmu.RLock()
	handler, ok := s.requestHandlers[msg.Method()]
	s.hmu.RUnlock()

	logrus.Println(msg.Method())
	if !ok {
		logrus.Errorln(len(msg.Method()))
		logrus.Errorln((msg.Method()))
		logrus.Errorln("not found handler func,requestMethod:", msg.Method(), msg.String())
		go handlerMethodNotAllowed(msg, tx)
		return
	}

	go handler(msg, tx)
}

func (s *Server) handlerResponse(msg *Response) {
	tx := s.getTX(getTXKey(msg))
	if tx == nil {
		logrus.Infoln("not found tx. receive response from:", msg.Source(), "message: \n", msg.String())
	} else {
		logrus.Traceln("receive response from:", msg.Source(), "txKey:", tx.key, "message: \n", msg.String())
		tx.receiveResponse(msg)
	}
}

// Request Request
func (s *Server) Request(req *Request) (*Transaction, error) {
	viaHop, ok := req.ViaHop()
	if !ok {
		return nil, fmt.Errorf("missing required 'Via' header")
	}
	viaHop.Host = s.host.String()
	viaHop.Port = s.port
	if viaHop.Params == nil {
		viaHop.Params = NewParams().Add("branch", String{Str: GenerateBranch()})
	}
	if !viaHop.Params.Has("rport") {
		viaHop.Params.Add("rport", nil)
	}

	tx := s.mustTX(req)
	return tx, tx.Request(req)
}

func handlerMethodNotAllowed(req *Request, tx *Transaction) {
	resp := NewResponseFromRequest("", req, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed), []byte{})
	tx.Respond(resp)
}

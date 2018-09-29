package protocol

import (
	"net"
	"net/textproto"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/*
Stub for beanstalkd protocol - simply echo the client requests to stdout
*/

// The protocol can only receive and process this type of data
type dataType int

// An error response that might be sent by the server
type errResponse []byte

const (
	text dataType = iota
	body
)

const yamlFMT = "---\n%s"

// ErrOutOfMem - The server cannot allocate enough memory for the job.
// 	The client should try again later.
var ErrOutOfMem errResponse = []byte(`OUT_OF_MEMORY\r\n`)

// ErrInternal - This indicates a bug in the server. It should never happen.
var ErrInternal errResponse = []byte(`INTERNAL_ERROR\r\n`)

// ErrBadFormat - The client sent a command line that was not well-formed.
//    This can happen if the line does not end with \r\n, if non-numeric
//    characters occur where an integer is expected, if the wrong number of
//    arguments are present, or if the command line is mal-formed in any other
//    way.
var ErrBadFormat errResponse = []byte(`BAD_FORMAT\r\n`)

// ErrUnknownCmd - The client sent a command that the server does not know.
var ErrUnknownCmd errResponse = []byte(`UNKNOWN_COMMAND\r\n`)

// Server is a yaad server
type Server struct {
	l net.Listener
}

// Connection implements a yaad + beanstalkd protocol server
type Connection struct {
	*textproto.Conn
	srv         BeanstalkdSrv
	defaultTube Tube
}

// NewServer returns a pointer to a new yaad server
func NewServer() *Server {
	return &Server{}
}

// Listen to connections
func (s *Server) Listen(protocol, address string) error {
	if protocol != "tcp" {
		return errors.Errorf("Cannot listen to non-tcp connections. Given protocol: %s", protocol)
	}
	l, err := net.Listen(protocol, address)
	logrus.Info("Server bound to socket")

	if err != nil {
		return errors.Wrap(err, "Cannot start protocol server")
	}

	s.l = l
	return nil
}

// Close the listener
func (s *Server) Close() error {
	return s.l.Close()
}

// ListenAndServe starts listening for new connections (blocking)
func (s *Server) ListenAndServe(protocol, address string) error {
	if err := s.Listen(protocol, address); err != nil {
		return err
	}
	for {
		// Wait for a connection.
		conn, err := s.l.Accept()
		if err != nil {
			logrus.Fatal(err)
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		srv := NewSrvStub()
		dt, _ := srv.getTube("default")
		go serve(&Connection{
			Conn:        textproto.NewConn(conn),
			srv:         srv,
			defaultTube: dt})
	}
}

func serve(conn *Connection) {
	conn.srv = NewSrvStub()
	for {
		line, err := conn.ReadLine()
		if err == nil {
			if line == "quit" {
				err := conn.Close()
				logrus.Error(err)
				return
			}
			parts := strings.Split(line, " ")
			cmd := parts[0]

			logrus.Debugf("Serving cmd: %s", cmd)
			switch cmd {
			case listTubes:
				conn.listTubes()
			case listTubeUsed:
				conn.listTubeUsed()
			case pauseTube:
				conn.pauseTube(parts[1:])
			case put:
				body, _ := conn.ReadLineBytes()
				conn.put(parts[1:], body)
			case reserveWithTimeout, reserve:
				conn.reserve()
			default:
				// Echo cmd by default
				conn.Writer.PrintfLine("%s", line)
			}
		}
	}
}

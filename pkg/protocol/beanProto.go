package protocol

import (
	"net"
	"net/textproto"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urjitbhatia/goyaad/pkg/goyaad"
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

// Metrics
var putJobCtr = "beanproto.putjob"
var deleteJobCtr = "beanproto.deletejob"
var reserveJobCtr = "beanproto.reservejob"
var connectionsCtr = "beanproto.connections"

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
	l   net.Listener
	srv BeanstalkdSrv
}

// Connection implements a yaad + beanstalkd protocol server
type Connection struct {
	*textproto.Conn
	srv         BeanstalkdSrv
	defaultTube Tube
	id          int
}

// NewYaadServer returns a pointer to a new yaad server
func NewYaadServer(stub bool) *Server {
	if stub {
		return &Server{srv: NewSrvStub()}
	}
	return &Server{
		srv: NewSrvYaad(),
	}
}

// Listen to connections
func (s *Server) Listen(protocol, address string) error {
	if protocol != "tcp" {
		return errors.Errorf("Cannot listen to non-tcp connections. Given protocol: %s", protocol)
	}
	l, err := net.Listen(protocol, address)
	if err != nil {
		return errors.Wrap(err, "Cannot start protocol server")
	}
	logrus.Info("Server bound to socket at: ", l.Addr().String())

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

	tube := &TubeYaad{
		name:   "default",
		paused: false,
		hub:    goyaad.NewHub(time.Second * 5),
	}
	connectionID := 0
	for {
		// Wait for a connection.
		conn, err := s.l.Accept()
		if err != nil {
			logrus.Fatal(err)
		}
		go goyaad.IncrMetric(connectionsCtr)
		connectionID++
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go serve(&Connection{
			Conn:        textproto.NewConn(conn),
			srv:         s.srv,
			defaultTube: tube,
			id:          connectionID})
	}
}

func serve(conn *Connection) {
	defer goyaad.DecrMetric(connectionsCtr)

	for {
		line, err := conn.ReadLine()
		if err != nil || line == "quit" {
			err := conn.Close()
			if err != nil {
				logrus.WithError(err).Panic("Error closing proto connection")
			}
			conn.srv = nil
			conn = nil
			return
		}

		parts := strings.Split(line, " ")
		cmd := parts[0]

		logrus.Debugf("Serving cmd: %s", cmd)
		switch cmd {
		case listTubes:
			listTubesCmd(conn)
		case listTubeUsed:
			listTubeUsedCmd(conn)
		case pauseTube:
			pauseTubeCmd(conn, parts[1:])
		case put:
			go goyaad.IncrMetric(putJobCtr)
			body, err := conn.ReadLineBytes()
			if err != nil {
				logrus.WithError(err).Fatal("error reading data")
			}
			data := make([]byte, len(body))
			copy(data, body)
			body = nil
			putCmd(conn, parts[1:], data[:])
		case reserve:
			go goyaad.MetricsClient.Incr(reserveJobCtr, nil, 1)
			reserveCmd(conn, "0")
		case reserveWithTimeout:
			go goyaad.IncrMetric(reserveJobCtr)
			reserveCmd(conn, parts[1])
		case deleteJob:
			go goyaad.IncrMetric(deleteJobCtr)
			deleteJobCmd(conn, parts[1:])
		default:
			// Echo cmd by default
			conn.Writer.PrintfLine("%s", line)
		}
	}
}

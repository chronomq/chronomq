package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	// tube methods
	listTubes       string = "list-tubes"
	listTubeUsed    string = "list-tube-used"
	listTubeWatched string = "list-tube-watched"
	pauseTube       string = "pause-tube"
	// producer commands
	put string = "put"

	// worker commands
	reserve            string = "reserve"
	reserveWithTimeout string = "reserve-with-timeout"
	deleteJob          string = "delete"
)

func listTubesCmd(conn *Connection) {
	listsYML, _ := yaml.Marshal(conn.srv.listTubes())

	preamble := fmt.Sprintf("OK %d", len(listsYML))
	conn.Writer.PrintfLine("%s", preamble)
	conn.Writer.PrintfLine("%s", listsYML)
}

func listTubeUsedCmd(conn *Connection) {
	conn.Writer.PrintfLine("USING foo")
}

func pauseTubeCmd(conn *Connection, args []string) error {
	if len(args) != 2 {
		return errors.New("Pause tube missing args")
	}
	name := args[0]
	t, err := conn.srv.getTube(name)
	if err != nil {
		conn.Writer.PrintfLine("NOT_FOUND")
		return nil
	}

	delayStr := args[1]
	d, err := strconv.Atoi(delayStr)
	if err != nil {
		return err
	}
	t.pauseTube(time.Duration(d))
	conn.Writer.PrintfLine("PAUSED")
	return nil
}

func putCmd(conn *Connection, args []string, body []byte) error {
	logrus.Debugf("protocol putting job with args: %s", args)
	pri, _ := strconv.ParseInt(args[0], 10, 32)
	delay, _ := strconv.Atoi(args[1])
	ttr, _ := strconv.Atoi(args[2])
	id, _ := conn.defaultTube.put(delay, int32(pri), body, ttr)

	conn.PrintfLine("INSERTED %s", id)

	return nil
}

func reserveCmd(conn *Connection, timeoutSec string) {
	j := conn.defaultTube.reserve(timeoutSec)
	if j != nil {
		conn.PrintfLine("RESERVED %s %d", j.id, j.size)
		conn.W.Write(j.body)
		conn.PrintfLine("")
		return
	}
	// Timeout reserve - try later
	conn.PrintfLine("TIMED_OUT")
}

func deleteJobCmd(conn *Connection, args []string) {
	id, _ := strconv.Atoi(args[0])
	err := conn.defaultTube.deleteJob(id)
	if err != nil {
		conn.PrintfLine("NOT_FOUND")
		return
	}
	conn.PrintfLine("DELETED")
}

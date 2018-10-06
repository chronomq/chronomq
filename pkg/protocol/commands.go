package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"time"

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

func (conn *Connection) listTubes() {
	listsYML, _ := yaml.Marshal(conn.srv.listTubes())

	preamble := fmt.Sprintf("OK %d", len(listsYML))
	conn.Writer.PrintfLine("%s", preamble)
	conn.Writer.PrintfLine("%s", listsYML)
}

func (conn *Connection) listTubeUsed() {
	conn.Writer.PrintfLine("USING foo")
}

func (conn *Connection) pauseTube(args []string) error {
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

func (conn *Connection) put(args []string, body []byte) error {
	pri, _ := strconv.ParseInt(args[0], 10, 32)
	delay, _ := strconv.Atoi(args[1])
	ttr, _ := strconv.Atoi(args[2])
	id, _ := conn.defaultTube.put(delay, int32(pri), body, ttr)

	conn.PrintfLine("INSERTED %s", id)

	return nil
}

func (conn *Connection) reserve() {
	for {
		j := conn.defaultTube.reserve()
		if j != nil {
			conn.PrintfLine("RESERVED %s %d", j.id, j.size)
			conn.W.Write(j.body)
			conn.PrintfLine("")
			break
		}
	}
}

func (conn *Connection) deleteJob(args []string) {
	id, _ := strconv.Atoi(args[0])
	err := conn.defaultTube.deleteJob(id)
	if err != nil {
		conn.PrintfLine("NOT_FOUND")
		return
	}
	conn.PrintfLine("DELETED")
}

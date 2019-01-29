package protocol

import (
	"errors"
	"net/rpc"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrClientDisconnected means a client was used while it was disconnected from the remote server
var ErrClientDisconnected = errors.New("Client is not connected to the server")

// RPCClient communicates with the Yaad RPC server
type RPCClient struct {
	client *rpc.Client
}

// Connect to a Yaad RCP Server and return a connected client
// Once connected, a client may be used by multiple goroutines simultaneously.
func (c *RPCClient) Connect(addr string) error {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.client = client

	return nil
}

// PutWithID saves a job with Yaad against a given id.
func (c *RPCClient) PutWithID(id string, body []byte, delay time.Duration) error {
	if c.client == nil {
		return ErrClientDisconnected
	}
	job := &RPCJob{ID: id, Body: body, Delay: delay}
	return c.client.Call("RPCServer.PutWithID", job, &id)
}

// Put saves a job with Yaad and returns the auto-generated job id
func (c *RPCClient) Put(body []byte, delay time.Duration) (string, error) {
	if c.client == nil {
		return "", ErrClientDisconnected
	}
	job := &RPCJob{ID: "", Body: body, Delay: delay}
	var id string
	err := c.client.Call("RPCServer.PutWithID", job, &id)
	return id, err
}

// Cancel deletes a job identified by the given id. Calls to cancel are idempotent
func (c *RPCClient) Cancel(id string) error {
	if c.client == nil {
		return ErrClientDisconnected
	}
	var ignoredReply int8
	return c.client.Call("RPCServer.Cancel", id, &ignoredReply)
}

// Next wait at-most timeout duration to return a ready job body from Yaad
// If no job is available within the timeout, ErrTimeout is returned and clients should try again later
func (c *RPCClient) Next(timeout time.Duration) (string, []byte, error) {
	if c.client == nil {
		return "", nil, ErrClientDisconnected
	}
	var job RPCJob
	err := c.client.Call("RPCServer.Next", timeout, &job)
	if err != nil {
		return "", nil, err
	}
	return job.ID, job.Body, nil
}

// Close the client connection
func (c *RPCClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping the server and check connectivity
func (c *RPCClient) Ping() error {
	if c.client == nil {
		return ErrClientDisconnected
	}
	var pong string
	err := c.client.Call("RPCServer.Ping", 0, &pong)
	if err != nil {
		return err
	}
	if pong != "pong" {
		return errors.New("Unexpected ping response: " + pong)
	}
	logrus.Debug("Received pong from server")
	return nil
}

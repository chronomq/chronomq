package chronomq

import (
	"errors"
	"net/rpc"
	"time"

	"github.com/rs/zerolog/log"
)

// ErrClientDisconnected means a client was used while it was disconnected from the remote server
var ErrClientDisconnected = errors.New("Client is not connected to the server")

// Client communicates with the Chronomq RPC server
type Client struct {
	client *rpc.Client
}

// Job is a light wrapper struct representing job data on the wire without extra metadata that is stored internally
type Job struct {
	Body  []byte
	ID    string
	Delay time.Duration
}

// NewClient creates an rpc client and tries to connect to a Chronomq RCP Server.
// Returns a connected client
// Once connected, a client may be used by multiple goroutines simultaneously.
func NewClient(addr string) (*Client, error) {
	c := &Client{}
	err := c.connect(addr)
	return c, err
}

func (c *Client) connect(addr string) error {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.client = client

	return nil
}

// PutWithID saves a job with Chronomq against a given id.
func (c *Client) PutWithID(id string, body []byte, delay time.Duration) error {
	if c.client == nil {
		return ErrClientDisconnected
	}
	job := &Job{ID: id, Body: body, Delay: delay}
	return c.client.Call("RPCServer.PutWithID", job, &id)
}

// Put saves a job with Chronomq and returns the auto-generated job id
func (c *Client) Put(body []byte, delay time.Duration) (string, error) {
	if c.client == nil {
		return "", ErrClientDisconnected
	}
	job := &Job{ID: "", Body: body, Delay: delay}
	var id string
	err := c.client.Call("RPCServer.PutWithID", job, &id)
	return id, err
}

// Cancel deletes a job identified by the given id. Calls to cancel are idempotent
func (c *Client) Cancel(id string) error {
	if c.client == nil {
		return ErrClientDisconnected
	}
	var ignoredReply int8
	return c.client.Call("RPCServer.Cancel", id, &ignoredReply)
}

// Next wait at-most timeout duration to return a ready job body from Chronomq
// If no job is available within the timeout, ErrTimeout is returned and clients should try again later
func (c *Client) Next(timeout time.Duration) (string, []byte, error) {
	if c.client == nil {
		return "", nil, ErrClientDisconnected
	}
	var job Job
	err := c.client.Call("RPCServer.Next", timeout, &job)
	if err != nil {
		return "", nil, err
	}
	return job.ID, job.Body, nil
}

// Close the client connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping the server and check connectivity
func (c *Client) Ping() error {
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
	log.Debug().Msg("Received pong from server")
	return nil
}

// InspectN fetches upto n number of jobs from the server without consuming them
func (c *Client) InspectN(n int, jobs *[]*Job) error {
	if c.client != nil {
		return c.client.Call("RPCServer.InspectN", n, jobs)
	}
	return nil
}

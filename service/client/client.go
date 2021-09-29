package client

import (
	"context"
	"fmt"
	"go_im/im/client"
	"go_im/im/conn"
	"go_im/im/message"
	"go_im/service/pb"
	"go_im/service/route"
	"go_im/service/rpc"
)

type Client struct {
	rpc.Cli
}

func NewClient(options *rpc.ClientOptions) (*Client, error) {
	ret := &Client{}
	var err error
	ret.Cli, err = rpc.NewBaseClient(options)
	if err != nil {
		return nil, err
	}
	client.Manager = ret
	return ret, nil
}

func NewClientByRouter(rtOpts *rpc.ClientOptions) (*Client, error) {
	ret := &Client{}
	var err error
	ret.Cli, err = route.NewRouter(ServiceName, rtOpts)
	if err != nil {
		return nil, err
	}
	client.Manager = ret
	return ret, nil
}

// ClientConnected idle function
func (c *Client) ClientConnected(conn conn.Connection) int64 {
	return 0
}

func (c *Client) AddClient(uid int64, cs client.IClient) {}

func (c *Client) ClientSignIn(id int64, uid int64, device int64) {
	req := &pb.SignInRequest{
		Old:    id,
		Uid:    uid,
		Device: device,
	}
	resp := &pb.Response{}
	err := c.Call(uidTagContext(id, device), "ClientSignIn", req, resp)
	if err != nil {

	}
}

func (c *Client) ClientLogout(uid int64, device int64) {
	resp := &pb.Response{}
	err := c.Call(uidTagContext(uid, device), "ClientLogout", &pb.LogoutRequest{Uid: uid, Device: device}, resp)
	if err != nil {

	}
}

func (c *Client) EnqueueMessage(uid int64, device int64, message *message.Message) {

	req := &pb.EnqueueMessageRequest{
		Uid:     uid,
		Device:  device,
		Message: wrapMessage(message),
	}
	resp := &pb.Response{}
	err := c.Call(uidTagContext(uid, device), "EnqueueMessage", req, resp)
	if err != nil {

	}
}

func (c *Client) AllClient() []int64 {
	// TODO
	return nil
}

func uidTagContext(uid int64, device int64) context.Context {
	ret := rpc.NewCtxFrom(context.Background())
	t := fmt.Sprintf("uid_%d_%d", uid, device)
	ret.PutReqExtra(route.ExtraTag, t)
	return ret
}

func wrapMessage(msg *message.Message) *pb.Message {
	return &pb.Message{
		Seq:    msg.Seq,
		Action: string(msg.Action),
		Data:   msg.Data,
	}
}

package group

import (
	"go_im/im/client"
	"go_im/im/comm"
	"go_im/im/dao"
	"go_im/im/message"
	"go_im/pkg/logger"
)

type Group struct {
	Gid int64
	Cid int64

	nextMid int64
	group   *dao.Group

	members *groupMemberMap
}

func NewGroup(group *dao.Group) *Group {
	ret := new(Group)
	ret.members = newGroupMemberMap()
	ret.Gid = group.Gid
	ret.Cid = group.ChatId
	ret.group = group
	chat, err := dao.ChatDao.GetChat(group.ChatId)
	if err != nil {
		logger.E("new group error, chat not exist", err)
		return nil
	}
	ret.nextMid = chat.CurrentMid + 1
	return ret
}

func (g *Group) PutMember(member int64, s int32) {
	g.members.Put(member, s)
}

func (g *Group) RemoveMember(uid int64) {
	g.members.Delete(uid)
}

func (g *Group) HasMember(uid int64) bool {
	return g.members.Contain(uid)
}

func (g *Group) IsMemberOnline(uid int64) bool {
	return false
}

func (g *Group) EnqueueMessage(senderUid int64, msg *client.GroupMessage) {

	chatMessage := dao.ChatMessage{
		Mid:         g.nextMid,
		Cid:         g.Cid,
		Sender:      senderUid,
		SendAt:      dao.Timestamp{},
		Message:     msg.Message,
		MessageType: msg.MessageType,
		At:          "",
	}
	err := dao.ChatDao.NewGroupMessage(chatMessage)

	if err != nil {
		logger.E("dispatch group message", err)
		return
	}

	rMsg := client.ReceiverChatMessage{
		Mid:         g.nextMid,
		Cid:         g.Cid,
		Sender:      senderUid,
		MessageType: msg.MessageType,
		Message:     msg.Message,
		SendAt:      msg.SendAt,
	}

	resp := message.NewMessage(-1, message.ActionChatMessage, rMsg)

	g.SendMessage(resp)

	g.nextMid = dao.GetNextMessageId(g.Cid)
}

func (g *Group) SendMessage(message *message.Message) {
	logger.D("Group.SendMessage: %s", message)

	for id := range g.members.members {
		client.EnqueueMessage(id, message)
	}
}

////////////////////////////////////////////////////////////////////////////////

type groupMemberMap struct {
	*comm.Mutex
	members map[int64]int32
}

func newGroupMemberMap() *groupMemberMap {
	ret := new(groupMemberMap)
	ret.Mutex = new(comm.Mutex)
	ret.members = make(map[int64]int32)
	return ret
}

func (g *groupMemberMap) Size() int {
	return len(g.members)
}

func (g *groupMemberMap) Get(id int64) int32 {
	defer g.LockUtilReturn()()
	member, ok := g.members[id]
	if ok {
		return member
	}
	return 0
}

func (g *groupMemberMap) Contain(id int64) bool {
	_, ok := g.members[id]
	return ok
}

func (g *groupMemberMap) Put(id int64, member int32) {
	defer g.LockUtilReturn()()
	g.members[id] = member
}

func (g *groupMemberMap) Delete(id int64) {
	defer g.LockUtilReturn()()
	delete(g.members, id)
}

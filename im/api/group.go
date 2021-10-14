package api

import (
	"errors"
	"go_im/im/dao"
	"go_im/im/group"
	"go_im/im/message"
)

type GroupApi struct{}

func (m *GroupApi) CreateGroup(msg *RequestInfo, request *CreateGroupRequest) error {

	g, cid, err := m.createGroup(request.Name, msg.Uid)
	if err != nil {
		return err
	}
	members, _ := dao.GroupDao.GetMembers(g.Gid)
	c := ContactResponse{
		Friends: []*UserInfoResponse{},
		Groups: []*GroupResponse{{
			Group:   *g,
			Members: members,
		}},
	}
	respond(msg.Uid, -1, ActionUserAddFriend, c)

	// create user chat by default
	uc, err := dao.ChatDao.NewUserChat(cid, msg.Uid, g.Gid, dao.ChatTypeGroup)
	if err != nil {
		return err
	}
	respond(msg.Uid, -1, ActionUserNewChat, uc)

	// add invited member to group
	if len(request.Member) > 0 {
		nMsg := &RequestInfo{
			Uid: msg.Uid,
			Seq: -1,
		}
		nReq := &AddMemberRequest{
			Gid: g.Gid,
			Uid: request.Member,
		}
		err = m.AddGroupMember(nMsg, nReq)
		if err != nil {
			respond(msg.Uid, -1, ActionFailed, "add member failed, "+err.Error())
		}
	}
	respond(msg.Uid, msg.Seq, ActionSuccess, "create group success")
	return nil
}

func (m *GroupApi) GetGroupMember(msg *RequestInfo, request *GetGroupMemberRequest) error {

	members, err := dao.GroupDao.GetMembers(request.Gid)
	if err != nil {
		return err
	}

	ms := make([]*GroupMemberResponse, 0, len(members))
	for _, member := range members {
		ms = append(ms, &GroupMemberResponse{
			Uid:        member.Uid,
			Nickname:   "",
			RemarkName: member.Remark,
			Type:       member.Type,
			Online:     true,
		})
	}

	respond(msg.Uid, msg.Seq, ActionSuccess, ms)
	return nil
}

func (m *GroupApi) GetGroupInfo(msg *RequestInfo, request *GroupInfoRequest) error {

	var groups []*GroupResponse

	for _, gid := range request.Gid {
		group1, e := dao.GroupDao.GetGroup(gid)
		if e != nil {
			return e
		}
		ms, _ := dao.GroupDao.GetMembers(gid)
		gr := GroupResponse{
			Group:   *group1,
			Members: ms,
		}
		groups = append(groups, &gr)
	}
	respond(msg.Uid, msg.Seq, ActionSuccess, groups)
	return nil
}

func (m *GroupApi) RemoveMember(msg *RequestInfo, request *RemoveMemberRequest) error {

	for _, uid := range request.Uid {
		err := dao.GroupDao.RemoveMember(request.Gid, uid)
		if err != nil {
			return err
		}
		_ = group.Manager.RemoveMember(request.Gid, uid)
		notifyResp := message.NewMessage(-1, ActionGroupRemoveMember, "you have been removed from the group xxx by xxx")
		respondMessage(uid, notifyResp)
	}

	resp := message.NewMessage(msg.Seq, ActionSuccess, "remove member success")

	respondMessage(msg.Uid, resp)
	return nil
}

func (m *GroupApi) AddGroupMember(msg *RequestInfo, request *AddMemberRequest) error {

	g, err := dao.GroupDao.GetGroup(request.Gid)
	if err != nil {
		return err
	}

	members, err := m.addGroupMember(g.Gid, request.Uid...)
	if err != nil {
		return err
	}

	// notify group member update group
	groupNotify := GroupAddMemberResponse{
		Gid:     g.Gid,
		Members: members,
	}
	group.Manager.DispatchNotifyMessage(g.Gid, message.NewMessage(-1, ActionGroupAddMember, groupNotify))

	for _, member := range members {

		// add group to member's contacts list
		_, e := dao.UserDao.AddContacts(member.Uid, g.Gid, dao.ContactsTypeGroup, "")
		if e != nil {
			_ = group.Manager.RemoveMember(request.Gid, member.Uid)
			continue
		}
		ms, err := dao.GroupDao.GetMembers(request.Gid)
		if err != nil {
			return err
		}
		//notify update contacts list
		c := ContactResponse{
			Friends: []*UserInfoResponse{},
			Groups: []*GroupResponse{{
				Group:   *g,
				Members: ms,
			}},
		}
		respond(member.Uid, -1, ActionUserAddFriend, c)

		// default add user chat
		chat, er := dao.ChatDao.NewUserChat(g.ChatId, member.Uid, g.Gid, dao.ChatTypeGroup)
		if er != nil {
			continue
		}
		// member subscribe group message
		group.Manager.PutMember(g.Gid, map[int64]int32{member.Uid: 1})

		// notify update chat list
		respond(member.Uid, -1, ActionUserNewChat, chat)
	}
	return nil
}

func (m *GroupApi) ExitGroup(msg *RequestInfo, request *ExitGroupRequest) error {

	err := group.Manager.RemoveMember(request.Gid, msg.Uid)
	if err != nil {
		return err
	}

	err = dao.GroupDao.RemoveMember(request.Gid, msg.Uid)
	if err != nil {
		return err
	}
	resp := message.NewMessage(msg.Seq, ActionSuccess, "exit group success")
	respondMessage(msg.Uid, resp)
	return err
}

func (m *GroupApi) JoinGroup(msg *RequestInfo, request *JoinGroupRequest) error {

	g, err := dao.GroupDao.GetGroup(request.Gid)
	if err != nil {
		return err
	}

	if g == nil {
		return errors.New("group does not exist")
	}

	_, err = m.addGroupMember(request.Gid, msg.Uid)
	if err != nil {
		return err
	}

	_, err = dao.UserDao.AddContacts(msg.Uid, g.Gid, dao.ContactsTypeGroup, "")

	members, err := dao.GroupDao.GetMembers(request.Gid)
	if err != nil {
		return err
	}

	c := ContactResponse{
		Friends: []*UserInfoResponse{},
		Groups: []*GroupResponse{{
			Group:   *g,
			Members: members,
		}},
	}
	respond(msg.Uid, -1, ActionUserAddFriend, c)

	chat, err := dao.ChatDao.NewUserChat(g.ChatId, msg.Uid, g.Gid, dao.ChatTypeGroup)
	if err != nil {
		_ = dao.GroupDao.RemoveMember(request.Gid, msg.Uid)
		return err
	}
	group.Manager.PutMember(g.Gid, map[int64]int32{msg.Uid: 1})
	respond(msg.Uid, -1, ActionUserNewChat, chat)
	respond(msg.Uid, msg.Seq, ActionSuccess, "join group success")
	return nil
}

func (m *GroupApi) createGroup(name string, uid int64) (*dao.Group, int64, error) {

	gp, err := dao.GroupDao.CreateGroup(name, uid)
	if err != nil {
		return nil, 0, err
	}
	// create group chat
	chat, err := dao.ChatDao.CreateChat(dao.ChatTypeGroup, gp.Gid)
	if err != nil {
		// TODO undo
		return nil, 0, err
	}

	gp.ChatId = chat.Cid
	err = dao.GroupDao.UpdateGroupChatId(gp.Gid, chat.Cid)
	if err != nil {
		return nil, 0, err
	}

	_, err = dao.GroupDao.AddMember(gp.Gid, dao.GroupMemberAdmin, uid)
	if err != nil {
		// TODO undo create group
		return nil, 0, err
	}
	_, err = dao.UserDao.AddContacts(uid, gp.Gid, dao.ContactsTypeGroup, "")
	if err != nil {
		// TODO undo
		return nil, 0, err
	}
	group.Manager.AddGroup(gp.Gid)
	return gp, chat.Cid, nil
}

func (m *GroupApi) addGroupMember(gid int64, uid ...int64) ([]*dao.GroupMember, error) {

	memberUid := make([]int64, 0, len(uid))
	members, _ := dao.GroupDao.GetMember(gid, uid...)
	existsMember := map[int64]interface{}{}
	for _, i := range members {
		existsMember[i.Uid] = nil
	}

	for _, u := range uid {
		// member exist
		if _, ok := existsMember[u]; !ok {
			memberUid = append(memberUid, u)
		}
	}
	if len(memberUid) == 0 {
		return nil, errors.New("already added")
	}

	// TODO query user info and notify group members, optimize query time
	exist, err2 := dao.UserDao.HasUser(memberUid...)
	if err2 != nil {
		return nil, err2
	}
	if !exist {
		return nil, errors.New("user does not exist")
	}

	members, err := dao.GroupDao.AddMember(gid, dao.GroupMemberUser, memberUid...)
	if err != nil {
		return nil, err
	}
	return members, nil
}

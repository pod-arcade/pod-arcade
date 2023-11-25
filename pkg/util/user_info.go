package util

import (
	"os/user"
	"strconv"
)

type UserInfo struct {
	UID  uint32
	GID  uint32
	GIDS []uint32
}

func GetUserInfo(username string) (*UserInfo, error) {
	info := &UserInfo{}
	user, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}
	uid, err := strconv.ParseInt(user.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	info.UID = uint32(uid)
	gid, err := strconv.ParseInt(user.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	info.GID = uint32(gid)

	gids, err := user.GroupIds()
	if err != nil {
		return nil, err
	}

	for _, gid := range gids {
		gid, err := strconv.ParseInt(gid, 10, 32)
		if err != nil {
			return nil, err
		}
		info.GIDS = append(info.GIDS, uint32(gid))
	}

	return info, nil
}

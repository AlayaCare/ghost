package usersync

import "github.com/Alayacare/goliac/internal/entity"

type UserSyncPluginNoop struct {
}

func NewUserSyncPluginNoop() UserSyncPlugin {
	return &UserSyncPluginNoop{}
}

func (p *UserSyncPluginNoop) UpdateUsers(users map[string]*entity.User) map[string]*entity.User {
	return users
}

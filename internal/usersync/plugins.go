package usersync

import "github.com/Alayacare/goliac/internal/entity"

func init() {
	registerPlugins()
}

type UserSyncPlugin interface {
	// Get the current user list, returns the new user list
	UpdateUsers(map[string]*entity.User) map[string]*entity.User
}

var plugins map[string]UserSyncPlugin

func registerPlugins() {
	plugins = make(map[string]UserSyncPlugin)
	plugins["noop"] = NewUserSyncPluginNoop()
}

func GetUserSyncPlugin(pluginname string) (UserSyncPlugin, bool) {
	plugin, found := plugins[pluginname]
	return plugin, found
}

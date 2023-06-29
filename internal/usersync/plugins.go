package usersync

import "github.com/Alayacare/goliac/internal/entity"

func init() {
	registerPlugins()
}

type UserSyncPlugin interface {
	// Get the current user list directory path, returns the new user list
	UpdateUsers(orguserdirrectorypath string) (map[string]*entity.User, error)
}

var plugins map[string]UserSyncPlugin

func registerPlugins() {
	plugins = make(map[string]UserSyncPlugin)
	plugins["noop"] = NewUserSyncPluginNoop()
	plugins["shellscript"] = NewUserSyncPluginShellScript()
}

func GetUserSyncPlugin(pluginname string) (UserSyncPlugin, bool) {
	plugin, found := plugins[pluginname]
	return plugin, found
}

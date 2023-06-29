package usersync

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/entity"
	"github.com/spf13/afero"
)

type UserSyncPluginNoop struct {
	Fs afero.Fs
}

func NewUserSyncPluginNoop() UserSyncPlugin {
	return &UserSyncPluginNoop{
		Fs: afero.NewOsFs(),
	}
}

func (p *UserSyncPluginNoop) UpdateUsers(orguserdirrectorypath string) (map[string]*entity.User, error) {
	users, errs, _ := entity.ReadUserDirectory(p.Fs, orguserdirrectorypath)
	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot load org users (for example: %v)", errs[0])
	}

	return users, nil
}

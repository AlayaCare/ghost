package usersync

import (
	"fmt"
	"os/exec"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/entity"
	"github.com/spf13/afero"
)

type UserSyncPluginShellScript struct{}

func NewUserSyncPluginShellScript() UserSyncPlugin {
	return &UserSyncPluginShellScript{}
}

func (p *UserSyncPluginShellScript) UpdateUsers(repoconfig *config.RepositoryConfig, orguserdirrectorypath string) (map[string]*entity.User, error) {
	cmd := exec.Command(repoconfig.UserSync.Path, orguserdirrectorypath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	fs := afero.NewOsFs()

	users, errs, _ := entity.ReadUserDirectory(fs, orguserdirrectorypath)
	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot load org users (for example: %v)", errs[0])
	}

	return users, nil
}

package internal

import (
	"fmt"
	"sort"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/slugify"
	"github.com/sirupsen/logrus"
)

/*
 * CodeOwnersGenerator will collects all teams definition to update the .github/CODEOWNERS file
 * cf https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners
 */
type CodeOwnersGenerator struct {
}

func NewCodeOwnersGenerator() *CodeOwnersGenerator {
	cog := CodeOwnersGenerator{}
	return &cog
}

func (cog *CodeOwnersGenerator) UpdateCodeOwners(local GoliacLocal, dryrun bool) error {

	content, err := local.GetCodeOwnersFileContent()
	if err != nil {
		return err
	}
	regen := codeowners_regenerate(local)
	if string(content) != regen {
		logrus.Info(".github/CODEOWNERS needs to be regenerated")
		if !dryrun {
			return local.SaveCodeOwnersFileContent([]byte(regen))
		}
	}
	return nil
}

func codeowners_regenerate(local GoliacLocal) string {
	codeowners := "# DO NOT MODIFY THIS FILE MANUALLY\n"

	teamsnames := make([]string, 0)
	for _, t := range local.Teams() {
		teamsnames = append(teamsnames, t.Metadata.Name)
	}
	sort.Sort(sort.StringSlice(teamsnames))

	for _, t := range teamsnames {
		codeowners += fmt.Sprintf("/org/%s @%s/%s\n", t, config.Config.GithubAppOrganization, slugify.Make(t))
	}

	return codeowners
}

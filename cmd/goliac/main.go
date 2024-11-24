package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Alayacare/goliac/internal"
	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/notification"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var dryrunParameter bool
var forceParameter bool
var repositoryParameter string
var branchParameter string

func main() {
	verifyCmd := &cobra.Command{
		Use:   "verify <path>",
		Short: "Verify the validity of IAC directory structure",
		Long:  `Verify the validity of IAC directory structure`,
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]
			goliac, err := internal.NewGoliacLightImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			err = goliac.Validate(path)
			if err != nil {
				logrus.Fatalf("failed to verify: %s", err)
			}
		},
	}

	planCmd := &cobra.Command{
		Use:   "plan [--repository https_team_repository_url] [--branch branch]",
		Short: "Check the validity of IAC directory structure against a Github organization",
		Long: `Check the validity of IAC directory structure against a Github organization.
repository: a remote repository in the form https://github.com/...
repository can be passed by parameter or by defining GOLIAC_SERVER_GIT_REPOSITORY env variable
branch can be passed by parameter or by defining GOLIAC_SERVER_GIT_BRANCH env variable`,
		Run: func(cmd *cobra.Command, args []string) {
			repo := ""
			branch := ""

			if len(args) == 2 {
				repo = args[0]
				branch = args[1]
			} else {
				repo = repositoryParameter
				branch = branchParameter
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments. Try --help")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			ctx := context.Background()
			err, _, _, _ = goliac.Apply(ctx, true, repo, branch, true)
			if err != nil {
				logrus.Errorf("Failed to plan: %v", err)
			}
		},
	}

	planCmd.Flags().StringVarP(&repositoryParameter, "repository", "r", config.Config.ServerGitRepository, "repository (default env variable GOLIAC_SERVER_GIT_REPOSITORY)")
	planCmd.Flags().StringVarP(&branchParameter, "branch", "b", config.Config.ServerGitBranch, "branch (default env variable GOLIAC_SERVER_GIT_BRANCH)")

	applyCmd := &cobra.Command{
		Use:   "apply [--repository https_team_repository_url] [--branch branch]",
		Short: "Verify and apply a IAC directory structure to a Github organization",
		Long: `Apply a IAC directory structure to a Github organization.
repository: a remote repository in the form https://github.com/...
repository can be passed by parameter or by defining GOLIAC_SERVER_GIT_REPOSITORY env variable
branch can be passed by parameter or by defining GOLIAC_SERVER_GIT_BRANCH env variable`,
		Run: func(cmd *cobra.Command, args []string) {
			repo := ""
			branch := ""

			if len(args) == 2 {
				repo = repositoryParameter
				branch = branchParameter
			} else {
				repo = config.Config.ServerGitRepository
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments, try --help")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}

			ctx := context.Background()
			err, _, _, _ = goliac.Apply(ctx, false, repo, branch, true)
			if err != nil {
				logrus.Errorf("Failed to apply: %v", err)
			}
		},
	}
	applyCmd.Flags().StringVarP(&repositoryParameter, "repository", "r", config.Config.ServerGitRepository, "repository (default env variable GOLIAC_SERVER_GIT_REPOSITORY)")
	applyCmd.Flags().StringVarP(&branchParameter, "branch", "b", config.Config.ServerGitBranch, "branch (default env variable GOLIAC_SERVER_GIT_BRANCH)")

	postSyncUsersCmd := &cobra.Command{
		Use:   "syncusers <https_team_repository_url> <branch>",
		Short: "Update and commit users and teams definition",
		Long: `This command will use a user sync plugin to adjust users
 and team yaml definition, and commit them.
 repository: a remote repository in the form https://github.com/...
 branch: the branch to commit to.
 repository can be passed by parameter or by defining GOLIAC_SERVER_GIT_REPOSITORY env variable
 branch can be passed by parameter or by defining GOLIAC_SERVER_GIT_BRANCH env variable`,
		Args: cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			repo := ""
			branch := ""

			if len(args) == 2 {
				repo = args[0]
				branch = args[1]
			} else {
				repo = config.Config.ServerGitRepository
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments. Try --help")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			ctx := context.Background()
			err = goliac.UsersUpdate(ctx, repo, branch, dryrunParameter, forceParameter)
			if err != nil {
				logrus.Fatalf("failed to update and commit teams: %s", err)
			}
		},
	}
	postSyncUsersCmd.Flags().BoolVarP(&dryrunParameter, "dryrun", "d", false, "dryrun mode")
	postSyncUsersCmd.Flags().BoolVarP(&forceParameter, "force", "f", false, "force mode")

	scaffoldcmd := &cobra.Command{
		Use:   "scaffold <directory> <adminteam>",
		Short: "Will create a base directory based on your current Github organization",
		Long: `Base on your Github organization, this command will try to scaffold a
goliac directory to let you start with something.
The adminteam is your current team that contains Github administrator`,
		Args: cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			directory := args[0]
			adminteam := args[1]
			if directory == "" || adminteam == "" {
				logrus.Fatalf("missing arguments. Try --help")
			}
			scaffold, err := internal.NewScaffold()
			if err != nil {
				logrus.Fatalf("failed to create scaffold: %s", err)
			}
			err = scaffold.Generate(directory, adminteam)
			if err != nil {
				logrus.Fatalf("failed to create scaffold direcrory: %s", err)
			}
		},
	}

	servecmd := &cobra.Command{
		Use:   "serve",
		Short: "This will start the application in server mode",
		Long: `This will start the application in server mode, which will
apply periodically (env:GOLIAC_SERVER_APPLY_INTERVAL)
any changes from the teams Git repository to Github.`,
		Run: func(cmd *cobra.Command, args []string) {
			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			notificationService := notification.NewNullNotificationService()
			if config.Config.SlackToken != "" && config.Config.SlackChannel != "" {
				slackService := notification.NewSlackNotificationService(config.Config.SlackToken, config.Config.SlackChannel)
				notificationService = slackService
			}

			server := internal.NewGoliacServer(goliac, notificationService)
			server.Serve()
		},
	}

	rootCmd := &cobra.Command{
		Use:   "goliac",
		Short: "A CLI for the goliac organization",
		Long: `a CLI library for goliac (GithHub Organization Sync Tool.
This CLI can mainly be plan (verify) or apply a IAC style directory structure to Github
Either local directory, or remote git repository`,
	}

	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(postSyncUsersCmd)
	rootCmd.AddCommand(scaffoldcmd)
	rootCmd.AddCommand(servecmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

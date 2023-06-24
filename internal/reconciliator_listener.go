package internal

type ReconciliatorListener interface {
	CreateTeam(teamname string, description string, members []string)
	UpdateTeamAddMember(teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	//UpdateTeamUpdateMember(teamslug string, username string, role string) // role can be 'member' or 'maintainer'
	UpdateTeamRemoveMember(teamslug string, username string)
	DeleteTeam(teamslug string)

	CreateRepository(reponame string, descrition string, writers []string, readers []string, public bool)
	UpdateRepositoryUpdateArchived(reponame string, archived bool)
	UpdateRepositoryUpdatePrivate(reponame string, private bool)
	UpdateRepositoryAddTeamAccess(reponame string, teamslug string, permission string)    // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryUpdateTeamAccess(reponame string, teamslug string, permission string) // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	UpdateRepositoryRemoveTeamAccess(reponame string, teamslug string)
	//	UpdateRepositoryAddCollaboratorAccess(reponame string, username string, permission string)    // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	//	UpdateRepositoryUpdateCollaboratorAccess(reponame string, username string, permission string) // permission can be "pull", "push", or "admin" which correspond to read, write, and admin access.
	//	UpdateRepositoryRemoveCollaboratorAccess(reponame string, username string)
	DeleteRepository(reponame string)

	Begin()
	Rollback(error)
	Commit()
}

# For regular users

As a regular user, you want to be able to
- create new team
- edit team's definition
- manage your team's repositories

## Create a new team

If you want to create a new team (like `foobar`), you need to create a PR with a `/teams/foobar/team.yaml` file:

```yaml
apiVersion: v1
kind: Team
name: foobar
spec:
  owners:
    - user1
    - user2
  members:
    - user3
    - user4
```

The users defined there are in 2 different categories
- members: are part of the team (and will be writer on all repositories of the team)
- owners: are part of the team (and will be writer on all repositories of the team) AMD can approve PR in the `foobar` teams repository (when you want to change a team definition, or when you want to create/update a repository definition)

The users name used are the one defined in the `/users` sub directories (like `alice`)

## Create a repository

On a given team subdirectory you can create a repository definition via a yaml file (like `/teams/foobar/awesome-repository.yaml`):

```yaml
apiVersion: v1
kind: Repository
name: awesome-repository
```

This will create a `awesome-repository` repository under your organization, that will be
- private by default
- writable by all owners/members of this team (in our example `foobar`)

You can of course tweak that:

```yaml
apiVersion: v1
kind: Repository
name: awesome-repository
spec:
  public: true
  allow_auto_merge: true
  delete_branch_on_merge: true
  allow_update_branch: true
  writers:
  - anotherteamA
  - anotherteamB
  readers:
  - anotherteamC
  - anotherteamD
```

In this last example:
- the repository is now public
- the repository allows auto merge
- the repository will delete the branch on merge
- the repository allows to update the branch
- other teams have write (`anotherteamA`, `anotherteamB`) or read (`anotherteamC`, `anotherteamD`) access

## Rename a repository

You need to add a `renameTo` to the repository, and Goliac will rename it (and update the `goliac-teams` repository):

```yaml
apiVersion: v1
kind: Repository
name: awesome-repository
spec:
  public: true
  ...
renameTo: anotherName
```


## Archive a repository

You can archive a repository, by a PR that move the yaml repository file into the `/archived` directory

## Adding repository ruleset

You can add different rules on a specific repository (like branch protection) using the new Github rulesets.
Few rules are currently supported (but the software can be easily extended): `pull_request`, `required_signatures`, `required_status_checks`, `creation`, `update`, `deletion`, `non_fast_forward`

Note: team or app bypass is not supported yet

```yaml
apiVersion: v1
kind: Repository
name: awesome-repository
spec:
  public: true
  ...
  rulesets:
    - name: myruleset
      enforcement: active # disabled, active, evaluate
      conditions:
        include: 
          - "~DEFAULT_BRANCH" # ~DEFAULT_BRANCH, ~ALL, branch_name, ...
      rules:
        - ruletype: required_signatures
```

`pull_request` example

```yaml
apiVersion: v1
kind: Repository
name: awesome-repository
spec:
  public: true
  ...
  rulesets:
    - name: myruleset
      enforcement: active # disabled, active, evaluate
      conditions:
        include: 
          - develop
      rules:
        - ruletype: pull_request
          parameters: # dismissStaleReviewsOnPush, requireCodeOwnerReview, requiredApprovingReviewCount, requiredReviewThreadResolution, requireLastPushApproval
            requiredApprovingReviewCount: 1
            requireLastPushApproval: true
            ...
```

`required_status_checks` example

```yaml
apiVersion: v1
kind: Repository
name: awesome-repository
spec:
  public: true
  ...
  rulesets:
    - name: myruleset
      enforcement: active # disabled, active, evaluate
      conditions:
        include: 
          - "~ALL"
      rules:
        - ruletype: required_status_checks
          parameters: # requiredStatusChecks, strictRequiredStatusChecksPolicy
            requiredStatusChecks:
              - my_check
```

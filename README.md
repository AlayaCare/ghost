# ![Goliac](docs/logo_small.png) Goliac

Goliac (Github Organization Leveraged by Infrastructure As Code), is a tool to manage your Github Organization (users/teams/repositories) via
- yaml manifests files structured in a Github repository
- this IAC Github repositories can be updated by teams from your organization, but only the repositories they owns
- all repositories rules are enforced via a central configuration that only the IT team can update
- a Github App watching this repository and applying any changes

## Why not using terraform/another tool

You can use Terraform to achieve almost the same result, except that with terraform, you still need to centrally managed all operations via your IT team.

Goliac allows you to provide a self-served tool to all your employees

## How to install and use

You need to
- install the Goliac Github App (and points to the server)
- create the IAC github repository (with the Github actions associated) or clone the example repository
- deploy the server somewhere (and configured it)

## Creating the Goliac Githun App

- Register new GitHub App
- Under Organization permissions
  - Give Read/Write access to `Administration`
  - Give Read/Write access to `Members`
- Under Repository permissions
  - Give Read/Write access to `Administration` 
  - Give Read/Write access to `Repository Content` 
- Where can this GitHub App be installed: `Only on this account`
- And Create
- then you must
  - collect the AppID
  - Generate (and collect) a client secret
- Go to the left tab "Install App"
  - Click on "Install"

## Creating the IAC github repository

You can check https://github.com/goliac-project/teams

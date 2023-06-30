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
- create the IAC github repository (with the Github actions associated) or clone the example repository
- deploy the server somewhere (and configured it)
- install the Goliac Github App (and points to the server)


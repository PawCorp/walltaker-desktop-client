# Contributing to the Walltaker Desktop Client

:+1::tada: First off, thanks for taking the time to contribute! :tada::+1:

The following is a set of guidelines for contributing to the Walltaker desktop client on GitHub. 

# Contributing

When contributing to this repository, please first discuss the change you wish to make via issue, email, or any other method with the owners of this repository before making a change. 

## Important Resources

- Bug? [Check for open issues about it first](https://github.com/PawCorp/walltaker-desktop-client/issues?q=is%3Aissue+label%3Abug+is%3Aopen), and if none exist, [create a new bug report](https://github.com/PawCorp/walltaker-desktop-client/issues/new?assignees=&labels=bug&template=bug_report.md&title=%5BBug%5D+).

- Enhancement/new feature? [Check for open issues about it first](https://github.com/PawCorp/walltaker-desktop-client/issues?q=is%3Aissue+is%3Aopen+label%3Aenhancement+), and if none exist, [create a new feature request](https://github.com/PawCorp/walltaker-desktop-client/issues/new?assignees=&labels=enhancement&template=feature_request.md&title=%5BEnhancement%5D+).

## Git Workflow: Branching Model

**main** - the main branch: only fully stable, working releases. Represents the current/latest version of stable released code.

**release** - branch serving as a point-in-time snapshot of a given release. Used when tagging new releases.

**develop** - this is the place where all development happens.

**feature/bugfix/hotfix/docs-branches** - a single branch which can be created by any developer, which is responsible for new features, fixes or reported bugs. Every feature branch name should start with the proper prefix and be named after the corresponding issue with the appropriate label. Branch names should follow the following format: `{prefix}/{issue number}-{descriptive-name}`. The `{descriptive-name}` doesn't need to match the issue name exactly (keep it short), but it should convey the purpose of the branch.

Examples:

- release: release/v1.1.1
- feature: feature/9-discord-presence
- bugfix: bugfix/4-macos-wallpaper-in-use
- docs: docs/18-contributing-docs

## Creating new branches

New branches ***must*** be created from develop. Branches may be created either by forking, or in the main repo (if you have permissions). Forking is recommended, especially for minor or experimental features. Core feature branches may be created in the main repo to ease collaboration. NOTE: An exception is made for critical hotfix branches - they may be diverged from main, but must be merged into both main and develop.

### Develop

Branch on which the main development happens. New feature/fix branches must be created from develop. Feature branch pull requests must be made against develop. It is recommended to rebase the feature branch to clean/fix/squash unnecessary commits prior to any pull request. Merge conflicts must be resolved by the branch owner (or person issuing pull request).

### Release

Branches used for pinning code versions for release. Release branches are diverged from the develop branch. Release branches constitute a "feature-freeze". Any bugs may be fixed via pull request using fix branches created from the newest release branch. However, changes must be synced back to develop.

### Main

The main branch that contains only fully stable, already released iterations of project. Accepts merges from the latest release branch only. Every merge to main must be properly tagged.


## Pull Request Process

Once opening an issue, if you decide to work on that issue, fork the repository and make your changes. Follow the branch naming convention outlined in the section above. 

# Semantic Versioning

Releases follow rules outlined by https://semver.org/

# Conventions and Style

Use `go fmt` to format your code.
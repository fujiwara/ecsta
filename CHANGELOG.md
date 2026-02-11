# Changelog

## [v0.8.0](https://github.com/fujiwara/ecsta/compare/v0.7.4...v0.8.0) - 2026-02-11
- Update olekukonko/tablewriter to v1 by @fujiwara in https://github.com/fujiwara/ecsta/pull/99
- Update all dependencies to latest by @fujiwara in https://github.com/fujiwara/ecsta/pull/101
- go fix (modernize) by @fujiwara in https://github.com/fujiwara/ecsta/pull/102

## [v0.7.4](https://github.com/fujiwara/ecsta/compare/v0.7.3...v0.7.4) - 2025-11-21
- supports `aws login` - update to aws-sdk-go-v2 v1.40.0 by @fujiwara in https://github.com/fujiwara/ecsta/pull/97

## [v0.7.3](https://github.com/fujiwara/ecsta/compare/v0.7.2...v0.7.3) - 2025-10-26

## [v0.7.2](https://github.com/fujiwara/ecsta/compare/v0.7.1...v0.7.2) - 2025-10-26
- add assets files into repos. by @fujiwara in https://github.com/fujiwara/ecsta/pull/94

## [v0.7.1](https://github.com/fujiwara/ecsta/compare/v0.7.0...v0.7.1) - 2025-09-19
- Immutable release by @fujiwara in https://github.com/fujiwara/ecsta/pull/92

## [v0.7.0](https://github.com/fujiwara/ecsta/compare/v0.6.2...v0.7.0) - 2025-06-06
- feat: add customizable log formatting with sloghandler by @fujiwara in https://github.com/fujiwara/ecsta/pull/86
- docs: add CLAUDE.md with project-specific development guidelines by @fujiwara in https://github.com/fujiwara/ecsta/pull/87
- feat: update tncl to v0.0.5 by @fujiwara in https://github.com/fujiwara/ecsta/pull/88
- feat: update Go dependencies to latest minor versions by @fujiwara in https://github.com/fujiwara/ecsta/pull/90
- feat: add --public flag for port forwarding to bind on all interfaces by @fujiwara in https://github.com/fujiwara/ecsta/pull/91

## [v0.6.2](https://github.com/fujiwara/ecsta/compare/v0.6.1...v0.6.2) - 2024-12-03
- Bump the aws-sdk-go-v2 group across 1 directory with 5 updates by @dependabot in https://github.com/fujiwara/ecsta/pull/70
- fix resolve ECS endpoint. by @fujiwara in https://github.com/fujiwara/ecsta/pull/75

## [v0.6.1](https://github.com/fujiwara/ecsta/compare/v0.6.0...v0.6.1) - 2024-08-26
- fix filling hostname as task ID. by @fujiwara in https://github.com/fujiwara/ecsta/pull/62

## [v0.6.0](https://github.com/fujiwara/ecsta/compare/v0.5.1...v0.6.0) - 2024-08-26
- Add ecsta cp command by @fujiwara in https://github.com/fujiwara/ecsta/pull/61

## [v0.5.1](https://github.com/fujiwara/ecsta/compare/v0.5.0...v0.5.1) - 2024-08-20
- Add logs --json flag. by @fujiwara in https://github.com/fujiwara/ecsta/pull/60

## [v0.5.0](https://github.com/fujiwara/ecsta/compare/v0.4.5...v0.5.0) - 2024-08-20
- Bump goreleaser/goreleaser-action from 5 to 6 by @dependabot in https://github.com/fujiwara/ecsta/pull/51
- Add trace --json flag. by @fujiwara in https://github.com/fujiwara/ecsta/pull/55
- Bumps the aws-sdk-go-v2 group with 5 updates by @fujiwara in https://github.com/fujiwara/ecsta/pull/56
- Fix/update mods by @fujiwara in https://github.com/fujiwara/ecsta/pull/57
- Switch to log/slog. by @fujiwara in https://github.com/fujiwara/ecsta/pull/58

## [v0.4.5](https://github.com/fujiwara/ecsta/compare/v0.4.4...v0.4.5) - 2024-04-19
- a local port allows "" or 0. use ephemeral port. by @fujiwara in https://github.com/fujiwara/ecsta/pull/48

## [v0.4.4](https://github.com/fujiwara/ecsta/compare/v0.4.3...v0.4.4) - 2024-04-19
- Add portforward -L flag by @fujiwara in https://github.com/fujiwara/ecsta/pull/44
- Bump actions/checkout from 3 to 4 by @dependabot in https://github.com/fujiwara/ecsta/pull/28
- Bump goreleaser/goreleaser-action from 4 to 5 by @dependabot in https://github.com/fujiwara/ecsta/pull/26
- Bump actions/setup-go from 4 to 5 by @dependabot in https://github.com/fujiwara/ecsta/pull/27

## [v0.4.3](https://github.com/fujiwara/ecsta/compare/v0.4.2...v0.4.3) - 2024-01-13
- update aws-sdk-go-v2 all modules to latest by @fujiwara in https://github.com/fujiwara/ecsta/pull/25

## [v0.4.2](https://github.com/fujiwara/ecsta/compare/v0.4.1...v0.4.2) - 2023-12-20
- Don't show a selection UI when a single task is found. by @fujiwara in https://github.com/fujiwara/ecsta/pull/23

## [v0.4.1](https://github.com/fujiwara/ecsta/compare/v0.4.0...v0.4.1) - 2023-11-16
- apply pty to session-manager-plugin if running without tty. by @fujiwara in https://github.com/fujiwara/ecsta/pull/22

## [v0.4.0](https://github.com/fujiwara/ecsta/compare/v0.3.5...v0.4.0) - 2023-10-13
- Fix/logs follow exit by @fujiwara in https://github.com/fujiwara/ecsta/pull/20
- Add ecsta list --output-tags and --tags option. by @fujiwara in https://github.com/fujiwara/ecsta/pull/21

## [v0.3.5](https://github.com/fujiwara/ecsta/compare/v0.3.4...v0.3.5) - 2023-08-10

## [v0.3.4](https://github.com/fujiwara/ecsta/compare/v0.3.3...v0.3.4) - 2023-07-25

## [v0.3.3](https://github.com/fujiwara/ecsta/compare/v0.3.2...v0.3.3) - 2023-07-14
- setConfigDir called at once. by @fujiwara in https://github.com/fujiwara/ecsta/pull/19

## [v0.3.2](https://github.com/fujiwara/ecsta/compare/v0.3.1...v0.3.2) - 2023-07-14
- refactoring config by @fujiwara in https://github.com/fujiwara/ecsta/pull/16
- kill session-manager-plugin when the task is stopping. by @fujiwara in https://github.com/fujiwara/ecsta/pull/17
- stop command shows tasks exclude stopped tasks. by @fujiwara in https://github.com/fujiwara/ecsta/pull/18

## [v0.3.1](https://github.com/fujiwara/ecsta/compare/v0.3.0...v0.3.1) - 2023-06-16
- Handle os.Interrupt for graceful shutdown. by @fujiwara in https://github.com/fujiwara/ecsta/pull/15

## [v0.3.0](https://github.com/fujiwara/ecsta/compare/v0.2.3...v0.3.0) - 2023-03-07
- exec and portforward commands show tasks without the last status being "STOPPED". by @fujiwara in https://github.com/fujiwara/ecsta/pull/13

## [v0.2.3](https://github.com/fujiwara/ecsta/compare/v0.2.2...v0.2.3) - 2023-03-03
- Fix AWS SSO by @fujiwara in https://github.com/fujiwara/ecsta/pull/12

## [v0.2.2](https://github.com/fujiwara/ecsta/compare/v0.2.1...v0.2.2) - 2023-02-20
- fix logs --follow didn't work. by @fujiwara in https://github.com/fujiwara/ecsta/pull/11

## [v0.2.1](https://github.com/fujiwara/ecsta/compare/v0.2.0...v0.2.1) - 2023-02-10
- Add `--start-time` option to logs command by @fujiwara in https://github.com/fujiwara/ecsta/pull/10

## [v0.2.0](https://github.com/fujiwara/ecsta/compare/v0.1.3...v0.2.0) - 2023-02-03
- Add -q (--task-format-query) option. by @fujiwara in https://github.com/fujiwara/ecsta/pull/9

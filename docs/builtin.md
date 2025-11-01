# Ansible Builtins Compatibility Status

## Modules

| Module Name                                | Implemented        | Compatible         | Test Playbook |
|--------------------------------------------|--------------------|--------------------|---------------|
| [apt](builtins/apt.md)                     | :white_check_mark: | :x:                | [playbook-apt.yaml](../data/playbooks/playbook-apt.yaml) |
| [command](builtins/command.md)             | :white_check_mark: | :x:                | [playbook-command.yaml](../data/playbooks/playbook-command.yaml) |
| [copy](builtins/copy.md)                   | :white_check_mark: | :x:                | [playbook-copy.yaml](../data/playbooks/playbook-copy.yaml) |
| [file](builtins/file.md)                   | :white_check_mark: | :x:                | [playbook-file.yaml](../data/playbooks/playbook-file.yaml) |
| [get_url](builtins/get_url.md)             | :white_check_mark: | :x:                | [playbook-get-url.yaml](../data/playbooks/playbook-get-url) |
| [import_tasks](builtins/import_tasks.md)   | :white_check_mark: | :white_check_mark: | [playbook-import-tasks](../data/playbooks/playbook-import-tasks.yaml) |
| [include_tasks](builtins/include_tasks.md) | :white_check_mark: | :x:                | [playbook-include-tasks](../data/playbooks/playbook-include-tasks.yaml) |
| [shell](builtins/shell.md)                 | :white_check_mark: | :white_check_mark: | [playbook-shell.yaml](../data/playbooks/playbook-shell.yaml) |
| add_host               | :x: | :x: | |
| apt_key                | :x: | :x: | |
| apt_repository         | :x: | :x: | |
| assemble               | :x: | :x: | |
| assert                 | :x: | :x: | |
| async_status           | :x: | :x: | |
| blockinfile            | :x: | :x: | |
| cron                   | :x: | :x: | |
| deb822_repository      | :x: | :x: | |
| debconf                | :x: | :x: | |
| debug                  | :x: | :x: | |
| dnf                    | :x: | :x: | |
| dnf5                   | :x: | :x: | |
| dpkg_selections        | :x: | :x: | |
| expect                 | :x: | :x: | |
| fail                   | :x: | :x: | |
| fetch                  | :x: | :x: | |
| find                   | :x: | :x: | |
| gather_facts           | :x: | :x: | |
| getent                 | :x: | :x: | |
| git                    | :x: | :x: | |
| group                  | :x: | :x: | |
| group_by               | :x: | :x: | |
| hostname               | :x: | :x: | |
| import_playbook        | :x: | :x: | |
| import_role            | :x: | :x: | |
| include_role           | :x: | :x: | |
| include_vars           | :x: | :x: | |
| iptables               | :x: | :x: | |
| known_hosts            | :x: | :x: | |
| lineinfile             | :x: | :x: | |
| meta                   | :x: | :x: | |
| mount_facts            | :x: | :x: | |
| package                | :x: | :x: | |
| package_facts          | :x: | :x: | |
| pause                  | :x: | :x: | |
| ping                   | :x: | :x: | |
| pip                    | :x: | :x: | |
| raw                    | :x: | :x: | |
| reboot                 | :x: | :x: | |
| replace                | :x: | :x: | |
| rpm_key                | :x: | :x: | |
| script                 | :x: | :x: | |
| service                | :x: | :x: | |
| service_facts          | :x: | :x: | |
| set_fact               | :x: | :x: | |
| set_stats              | :x: | :x: | |
| setup                  | :x: | :x: | |
| slurp                  | :x: | :x: | |
| stat                   | :x: | :x: | |
| subversion             | :x: | :x: | |
| systemd_service        | :x: | :x: | |
| sysvinit               | :x: | :x: | |
| tempfile               | :x: | :x: | |
| template               | :x: | :x: | |
| unarchive              | :x: | :x: | |
| uri                    | :x: | :x: | |
| user                   | :x: | :x: | |
| validate_argument_spec | :x: | :x: | |
| wait_for               | :x: | :x: | |
| wait_for_connection    | :x: | :x: | |
| yum_repository         | :x: | :x: | |

## Others (strategy, inventory, connection, ..., plugins)

Not implemented.

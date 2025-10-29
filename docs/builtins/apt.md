# ansible.builtin.apt

## Implementation

| Source | Parameters | Deviations |
|--------|------------|------------|
| [apt.go](../../pkg/exec/apt.go) | :x: | :x: |

## Parameters

| Name | Implemented |
|------|-------------|
| allow_change_held_packages |  :x:  |
| allow_downgrade |  :x:  |
| allow_unauthenticated |  :x:  |
| auto_install_module_deps |  :x:  |
| autoclean |  :x:  |
| autoremove |  :x:  |
| cache_valid_time |  :white_check_mark:  |
| clean |  :x:  |
| deb |  :x:  |
| default_release |  :x:  |
| dpkg_options |  :x:  |
| fail_on_autoremove |  :x:  |
| force |  :x:  |
| force_apt_get |  :x:  |
| install_recommends |  :x:  |
| lock_timeout |  :x:  |
| name |  :white_check_mark:  |
| only_upgrade |  :x:  |
| policy_rc_d |  :x:  |
| purge |  :x:  |
| state |  :white_check_mark:  |
| update_cache |  :white_check_mark:  |
| update_cache_retries |  :x:  |
| update_cache_retry_max_delay |  :x:  |
| upgrade |  :white_check_mark:  |

## Deviations

* `state` only supports `present`, `latest` and `absent`
* `upgrade` only supports `dist` and `yes`
* aliases for `name` are not supported
* version strings in package names are not supported
* `name` needs to be a list (one element is ok), a single string is not supported


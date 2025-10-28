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
| cache_valid_time |  :x:  |
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
| state |  :x:  |
| update_cache |  :white_check_mark:  |
| update_cache_retries |  :x:  |
| update_cache_retry_max_delay |  :x:  |
| upgrade |  :x:  |

## Deviations

* `state` only supports `present` and `absent`* aliases for `name` are not supported

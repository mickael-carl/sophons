# ansible.builtin.get_url

## Implementation

| Source | Parameters | Deviations |
|--------|------------|------------|
| [get_url.go](../../pkg/exec/get_url.go) | :x: | :x: |

## Parameters

| Name | Implemented |
|------|-------------|
| attributes |  :x:  |
| backup |  :x:  |
| checksum |  :x:  |
| ciphers |  :x:  |
| client_cert |  :x:  |
| client_key |  :x:  |
| decompress |  :x:  |
| dest |  :white_check_mark:  |
| force |  :x:  |
| force_basic_auth |  :x:  |
| group |  :white_check_mark:  |
| headers |  :x:  |
| mode |  :white_check_mark:  |
| owner |  :white_check_mark:  |
| selevel |  :x:  |
| serole |  :x:  |
| setype |  :x:  |
| seuser |  :x:  |
| timeout |  :x:  |
| tmp_dest |  :x:  |
| unredirected_headers |  :x:  |
| unsafe_writes |  :x:  |
| url |  :white_check_mark:  |
| url_password |  :x:  |
| url_username |  :x:  |
| use_gssapi |  :x:  |
| use_netrc |  :x:  |
| use_proxy |  :x:  |
| validate_certs |  :x:  |

## Deviations

* `url` doesn't support `ftp` or `file` schemes.


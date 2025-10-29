# ansible.builtin.copy

## Implementation

| Source | Parameters | Deviations |
|--------|------------|------------|
| [copy.go](../../pkg/exec/copy.go) | :x: | :x: |

## Parameters

| Name | Implemented |
|------|-------------|
| attributes |  :x:  |
| backup |  :x:  |
| checksum |  :x:  |
| content |  :white_check_mark:  |
| decrypt |  :x:  |
| dest |  :white_check_mark:  |
| directory_mode |  :x:  |
| follow |  :x:  |
| force |  :x:  |
| group |  :x:  |
| local_follow |  :x:  |
| mode |  :x:  |
| owner |  :x:  |
| remote_src |  :x:  |
| selevel |  :x:  |
| serole |  :x:  |
| setype |  :x:  |
| seuser |  :x:  |
| src |  :white_check_mark:  |
| unsafe_writes |  :x:  |
| validate |  :x:  |

## Deviations

* `src` doesn't support absolute paths when `remote_src` is false.


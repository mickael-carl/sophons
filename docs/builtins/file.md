# ansible.builtin.command

## Implementation

| Source                            | Parameters | Deviations |
|-----------------------------------|------------|------------|
| [file.go](../../pkg/exec/file.go) | :x:        | :x:        |

## Parameters

| Name                     | Implemented        |
|--------------------------|--------------------|
| access_time              | :x:                |
| access_time_format       | :x:                |
| attributes               | :x:                |
| follow                   | :white_check_mark: |
| force                    | :x:                |
| group                    | :white_check_mark: |
| mode                     | :white_check_mark: |
| modification_time        | :x:                |
| modification_time_format | :x:                |
| owner                    | :white_check_mark: |
| path                     | :white_check_mark: |
| recurse                  | :white_check_mark: |
| selevel                  | :x:                |
| serole                   | :x:                |
| setype                   | :x:                |
| seuser                   | :x:                |
| src                      | :white_check_mark: |
| state                    | :white_check_mark: |
| unsafe_writes            | :x:                |

## Deviations

* `mode` doesn't support string syntax, e.g. `u+rw,o=r".

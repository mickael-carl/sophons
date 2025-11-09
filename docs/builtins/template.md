# ansible.builtin.template

## Implementation

| Source | Parameters | Deviations |
|--------|------------|------------|
| [template.go](../../pkg/exec/template.go) | :x: | :x: |

## Parameters

| Name | Implemented |
|------|-------------|
| attributes |  :x:  |
| backup |  :x:  |
| block_end_string |  :x:  |
| block_start_string |  :x:  |
| comment_end_string |  :x:  |
| comment_start_string |  :x:  |
| dest |  :white_check_mark:  |
| follow |  :x:  |
| force |  :x:  |
| group |  :white_check_mark:  |
| lstrip_blocks |  :x:  |
| mode |  :white_check_mark:  |
| newline_sequence |  :x:  |
| output_encoding |  :x:  |
| owner |  :white_check_mark:  |
| selevel |  :x:  |
| serole |  :x:  |
| setype |  :x:  |
| seuser |  :x:  |
| src |  :white_check_mark:  |
| trim_blocks |  :x:  |
| unsafe_writes |  :x:  |
| validate |  :x:  |
| variable_end_string |  :x:  |
| variable_start_string |  :x:  |

## Deviations

* `src` doesn't support absolute paths.


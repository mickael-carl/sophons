package registry

import "reflect"

// TaskRegistration holds all information needed to handle a task type.
type TaskRegistration struct {
	// ProtoFactory creates a new proto message (e.g., &proto.Copy{})
	ProtoFactory func() any

	// ProtoWrapper wraps the proto message in the Task oneof (e.g.,
	// &Task_Copy{Copy: msg.(*Copy)})
	ProtoWrapper func(any) any

	// ExecAdapter converts proto oneof to exec.TaskContent (e.g., *Task_Copy
	// -> *exec.Copy)
	ExecAdapter func(any) any
}

// NameRegistry maps task names (e.g., "copy", "ansible.builtin.copy") to
// registrations. Used during YAML unmarshaling to determine which proto type
// to create.
var NameRegistry = map[string]TaskRegistration{}

// TypeRegistry maps proto oneof types (e.g., *proto.Task_Copy) to
// registrations. Used during proto->exec conversion to determine which exec
// type to create.
var TypeRegistry = map[reflect.Type]TaskRegistration{}

// Register registers a task type for both YAML unmarshaling and proto
// conversion.
func Register(name string, reg TaskRegistration, protoTypeExample any) {
	NameRegistry[name] = reg

	if protoTypeExample != nil {
		t := reflect.TypeOf(protoTypeExample)
		TypeRegistry[t] = reg
	}
}

package ssh

import (
	"fmt"
)

// ParseExecArgs splits cobra args into the VM reference and remote command argv.
func ParseExecArgs(args []string) (vm string, command []string, err error) {
	if len(args) < 1 {
		return "", nil, fmt.Errorf("vm name or slug is required")
	}
	vm = args[0]

	sep := -1
	for i := 1; i < len(args); i++ {
		if args[i] == "--" {
			sep = i
			break
		}
	}

	switch {
	case sep >= 0:
		command = args[sep+1:]
	case len(args) >= 2:
		command = args[1:]
	default:
		return "", nil, fmt.Errorf("remote command is required (e.g. rumpty exec %s -- uptime)", vm)
	}

	if len(command) == 0 {
		return "", nil, fmt.Errorf("remote command is required (e.g. rumpty exec %s -- uptime)", vm)
	}
	return vm, command, nil
}

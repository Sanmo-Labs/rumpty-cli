package ssh

import (
	"fmt"
	"strings"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
)

// CopyPaths holds parsed local/remote paths for a copy operation.
type CopyPaths struct {
	VMRef  string
	Local  string
	Remote string
	Upload bool
}

// ParseCopyPaths parses scp-style src and dest arguments. Exactly one side must use vm:path.
func ParseCopyPaths(src, dest string) (CopyPaths, error) {
	srcVM, srcPath, srcRemote := splitCopyRemote(src)
	destVM, destPath, destRemote := splitCopyRemote(dest)

	switch {
	case srcRemote && destRemote:
		return CopyPaths{}, fmt.Errorf("only one of src or dest may be a remote path (vm:path)")
	case !srcRemote && !destRemote:
		return CopyPaths{}, fmt.Errorf("one of src or dest must be a remote path (vm:path)")
	case destRemote:
		if strings.TrimSpace(destVM) == "" {
			return CopyPaths{}, fmt.Errorf("remote dest is missing a VM name")
		}
		return CopyPaths{
			VMRef:  destVM,
			Local:  src,
			Remote: destPath,
			Upload: true,
		}, nil
	default:
		if strings.TrimSpace(srcVM) == "" {
			return CopyPaths{}, fmt.Errorf("remote src is missing a VM name")
		}
		return CopyPaths{
			VMRef:  srcVM,
			Local:  dest,
			Remote: srcPath,
			Upload: false,
		}, nil
	}
}

func splitCopyRemote(spec string) (vmRef, remotePath string, ok bool) {
	idx := strings.Index(spec, ":")
	if idx <= 0 {
		return "", "", false
	}
	// Windows drive letter (C:\ or C:/).
	if idx == 1 && len(spec) > 2 && (spec[2] == '\\' || spec[2] == '/') {
		return "", "", false
	}
	vmRef = strings.TrimSpace(spec[:idx])
	remotePath = spec[idx+1:]
	if remotePath == "" {
		remotePath = "."
	}
	return vmRef, remotePath, true
}

func remoteCopyTarget(session *api.CertResponse, remotePath string) string {
	return session.Username + "@" + session.VMSlug + ":" + remotePath
}

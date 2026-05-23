package ssh

import "github.com/Sanmo-Labs/rumpty-cli/internal/api"

func BuildSSHArgsForTest(proxyCommand string, session *api.CertResponse, opts *Options) []string {
	return buildSSHArgs(proxyCommand, session, opts)
}

func BuildProxyCommandForTest(sshBin string, session *api.CertResponse, keyPath, certPath string) string {
	return buildProxyCommand(sshBin, session, keyPath, certPath, false)
}

func KnownHostsFileForTest() (string, error) {
	return knownHostsFile()
}

func NeedsListLookupForTest(ref string) bool {
	return needsListLookup(ref)
}

func ResetCertCacheForTest() {
	resetCertCacheForTest()
}

func ResetCertStoreForTest() error {
	return resetCertStoreForTest()
}

func PutCertCacheForTest(apiURL, workspace, vmSlug, guestUser string, key KeyPair, session *api.CertResponse) {
	sessionCache.put(apiURL, workspace, vmSlug, guestUser, key, session)
}

func GetCertCacheForTest(apiURL, workspace, vmSlug, guestUser string) (KeyPair, api.CertResponse, bool) {
	return sessionCache.get(apiURL, workspace, vmSlug, guestUser)
}

func BuildRsyncArgsForTest(rsyncSSHWrapper string, session *api.CertResponse, paths CopyPaths, opts *Options) []string {
	return buildRsyncArgs(rsyncSSHWrapper, session, paths, opts)
}

func WriteRsyncSSHWrapperForTest(dir, sshBin, proxyCommand string, opts *Options) (string, error) {
	return writeRsyncSSHWrapper(dir, sshBin, proxyCommand, opts)
}

func RsyncMissingOnRemoteForTest(output string) bool {
	return rsyncMissingOnRemote(output)
}

func CopyRecursiveForTest(paths CopyPaths, flag bool) bool {
	return copyRecursive(paths, flag)
}

func BuildSCPArgsForTest(proxyCommand string, session *api.CertResponse, paths CopyPaths, opts *Options, recursive bool) (src, dest string, args []string) {
	return buildSCPArgs(proxyCommand, session, paths, opts, recursive)
}

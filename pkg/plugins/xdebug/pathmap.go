package xdebug

import "strings"

func toRelative(uri, remoteRoot string) string {
	path := strings.TrimPrefix(uri, "file://")

	if remoteRoot == "" {
		return path
	}

	return strings.TrimPrefix(path, remoteRoot+"/")
}

func toURI(relativePath, root, remoteRoot string) string {
	if strings.HasPrefix(relativePath, "/") {
		return "file://" + relativePath
	}

	base := remoteRoot
	if base == "" {
		base = root
	}

	return "file://" + base + "/" + relativePath
}

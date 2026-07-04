package pathmap

import "strings"

func ToRelative(uri, remoteRoot string) string {
	path := strings.TrimPrefix(uri, "file://")

	if remoteRoot == "" {
		return path
	}

	return strings.TrimPrefix(path, remoteRoot+"/")
}

func ToURI(relativePath, root, remoteRoot string) string {
	if strings.HasPrefix(relativePath, "/") {
		return "file://" + relativePath
	}

	base := remoteRoot
	if base == "" {
		base = root
	}

	return "file://" + base + "/" + relativePath
}

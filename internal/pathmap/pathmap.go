package pathmap

import "strings"

func ToRelative(uri, remoteRoot string) string {
	path := strings.TrimPrefix(uri, "file://")

	if remoteRoot == "" {
		return path
	}

	return strings.TrimPrefix(path, remoteRoot+"/")
}

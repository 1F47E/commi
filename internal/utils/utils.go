package utils

import "os"

func IsDebug() bool {
	return os.Getenv("DEBUG") != ""
}

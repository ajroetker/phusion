package main

import (
    "os"
    "fmt"
)

func init() {
    dir, err := os.Getwd()
    logFatal(err)
    tarballs := fmt.Sprintf("%v/tarballs", dir)
    os.MkdirAll( tarballs, 0755)
    versions := fmt.Sprintf("%v/versions", dir)
    os.MkdirAll( versions, 0755)
}

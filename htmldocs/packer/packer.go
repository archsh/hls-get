// +build ignore
package main

import (
    "log"
    "github.com/shurcooL/vfsgen"
    "net/http"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatalln("Usage: packer <DIR>")
        os.Exit(1)
    }
    err := vfsgen.Generate(http.Dir(os.Args[1]), vfsgen.Options{
        PackageName:  "htmldocs",
        BuildTags:    "release",
        VariableName: "Assets",
    })
    if err != nil {
        log.Fatalln(err)
        os.Exit(1)
    }
}
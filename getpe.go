package main

import (
    "net/http"
    "io/ioutil"
    "io"
    "fmt"
    "log"
    "flag"
    "strings"
    "os"
    "archive/tar"
    "github.com/cheggaaa/pb"
)

type PE struct {
    Version, Platform, Build string
}

var version, platform *string

func fetch( latest, tarball string ) ( err error ) {
    url := fmt.Sprintf( "http://neptune.puppetlabs.lan/%v/ci-ready/puppet-enterprise-%v-%v.tar", *version, latest, *platform)
    resp, err := http.Get(url)
    defer resp.Body.Close()
    if err != nil {
        return err
    }
    pe, err := os.Create(tarball)
    defer pe.Close()
    if err != nil {
        return err
    }
    // create and start bar
    bar := pb.New64(resp.ContentLength).SetUnits(pb.U_BYTES)
    bar.Start()
    writer := io.MultiWriter(pe, bar)
    _, err = io.Copy(writer, resp.Body)
    return err
}

func unpack( tarball, versions, latest string ) ( err error ) {
    version := fmt.Sprintf( "%v/puppet-enterprise-%v-%v", versions, latest, *platform )
    os.MkdirAll( version, 0755)

    file, err := os.Open( tarball )
    defer file.Close()
    if err != nil {
        return
    }
    tr := tar.NewReader( file )
    absPath := func( name string ) string {
        return fmt.Sprintf("%v/%v", versions, name)
    }
    // Iterate through the files in the archive.
    for {
        hdr, err := tr.Next()
        if err == io.EOF {
            // end of tar archive
            break
        }
        if err != nil {
            return err
        }
        if splitPath := strings.Split(hdr.Name, "/"); len(splitPath) > 2 {
            // Remove the file name so we can make all
            // the directories up to it
            splitParentPath := splitPath[:len(splitPath)-1]
            parentPath := absPath( strings.Join(splitParentPath, "/") )
            os.Mkdir(parentPath, 0755)
        }
        path := absPath(hdr.Name)
        tmp, err := os.Create(path)
        defer tmp.Close()
        if err != nil {
            return err
        }
        if _, err := io.Copy(tmp, tr); err != nil {
            return err
        }
    }
    return nil
}

func init() {
    version = flag.String("version", "3.4", "Puppet Enterprise Version")
    platform = flag.String("platform", "debian-7-amd64", "Target Platform Tag (e.g el-6-x86_64)")
    url := fmt.Sprintf( "http://neptune.puppetlabs.lan/%v/ci-ready/LATEST", *version )
    resp, err := http.Get(url)
    logFatal( err )
    defer resp.Body.Close()
    bytes, err := ioutil.ReadAll(resp.Body)
    logFatal( err )
    latest := strings.TrimRight(string(bytes), "\n")
    log.Printf("[ INFO ] Latest PE is %v", latest)
    dir, err := os.Getwd()
    logFatal( err )
    tarball := fmt.Sprintf( "%v/tarballs/puppet-enterprise-%v-%v.tar", dir, latest, *platform )
    if _, err := os.Stat(tarball); os.IsNotExist(err) {
        if err = fetch(latest, tarball); err != nil {
            logError( err )
        } else {
            log.Printf("[ INFO ] Downloaded %v for %v", latest, *platform)
        }
    } else {
        log.Printf("[ INFO ] Already at newest version")
    }
    versions := fmt.Sprintf("%v/versions", dir)
    err = unpack( tarball, versions, latest )
    logFatal( err )
    version := fmt.Sprintf( "%v/puppet-enterprise-%v-%v", versions, latest, *platform )
    pesymlink := fmt.Sprintf( "%v/puppet-enterprise", dir )
    err = os.RemoveAll(pesymlink)
    logFatal( err )
    err = os.Link(version,pesymlink)
    logFatal( err )
}

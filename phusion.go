package main

import (
    "fmt"
    "flag"
    "bytes"
    "os/exec"
    "log"
    "os"
    git "github.com/libgit2/git2go"
    "code.google.com/p/gcfg"
)

type Component interface {
    retrieve()
    destroy()
    build()
    clean()
    install()
}

type EnterpriseDist struct {
  Owner, Path, Branch string
}

func (ed *EnterpriseDist) retrieve() {
    if _, err := os.Stat(ed.Path); os.IsNotExist(err) {
        hub_clone( ed.Owner, "enterprise-dist", ed.Branch )
    }
}

func (ed *EnterpriseDist) destroy() error {
    log.Printf( "[ INFO ] removing %v", ed.Path)
    return os.RemoveAll(ed.Path)
}

func (ed *EnterpriseDist) install() {
    ederb := fmt.Sprintf("%v/ext/erb", ed.Path)
    peerb := fmt.Sprintf("%v/puppet-enterprise/erb", modulepath)
    err := os.RemoveAll( peerb )
    if err != nil {
        logError( err )
        return
    }
    err = os.Symlink( ederb, peerb )
    if err != nil {
        logError( err )
        return
    }
    for _, component := range []string{
        "puppet-enterprise-installer",
        "puppet-enterprise-uninstaller",
        "utilities",
        "pe-classification.rb",
    } {
        edcp := fmt.Sprintf("%v/installer/%v", ed.Path, component)
        pecp := fmt.Sprintf("%v/puppet-enterprise/%v", modulepath, component)
        err = CopyFile(edcp, pecp)
        if err != nil {
            logError( fmt.Errorf( "CopyFile failed %q", err ) )
        } else {
            log.Printf("[ INFO ] CopyFile succeeded %v", component)
        }
    }
}

type Module struct {
  Name, Owner, Path, Branch string
}

func (m *Module) install() {

    pkgPath  := fmt.Sprintf( "%v/pkg", m.Path )
    tarRegex := fmt.Sprintf( "%v-.*tar.gz", m.Name )
    pkgName, err := getFile( pkgPath, tarRegex )
    if err != nil {
        logError( err )
        return
    }
    pkg := fmt.Sprintf( "%v/%v", pkgPath, pkgName )
    pemodulepath := fmt.Sprintf("%v/puppet-enterprise/modules", modulepath)
    old, err := getFile( pemodulepath, tarRegex )
    if err != nil {
        logError( err )
        return
    }
    err = os.RemoveAll( old )
    if err != nil {
        logError( err )
        return
    }
    dst :=  fmt.Sprintf("%v/%v", pemodulepath, pkgName)
    log.Printf("[ INFO ] Copying %s to %s", pkg, dst)
    err = CopyFile(pkg, dst)
    if err != nil {
        logError( fmt.Errorf( "CopyFile failed %q", err ) )
    } else {
        log.Printf("[ INFO ] CopyFile succeeded moving %v", pkgName)
    }

}

func (m *Module) retrieve() {
    if _, err := os.Stat(m.Path); os.IsNotExist(err) {
        hub_clone( m.Owner, m.Name, m.Branch )
    }
}

func (m *Module) build() {
    cmd := exec.Command("puppet", "module", "build", m.Path)
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    logError( err )
    if err == nil {
        log.Printf( "[ INFO ] %v", out.String() )
    }
}

func (m *Module) clean() error {
    pkg := fmt.Sprintf("%s/pkg", m.Path)
    log.Printf( "[ INFO ] cleaning %v", pkg)
    return os.RemoveAll(pkg)
}

func (m *Module) destroy() error {
    log.Printf( "[ INFO ] removing %v", m.Path)
    return os.RemoveAll(m.Path)
}

var username, password, modulepath string

func init() {
    const (
        defaultUser = "ajroetker@gmail.com"
        defaultPassword = "1234!@#$"
    )
    flag.StringVar(&username, "user", defaultUser, "GitHub User")
    flag.StringVar(&username, "u", defaultUser, "GitHub User (shorthand)")
    flag.StringVar(&password, "password", defaultPassword, "GitHub Password")
    flag.StringVar(&password, "p", defaultPassword, "GitHub Password (shorthand)")
    dir, err := os.Getwd()
    logFatal(err)
    flag.StringVar(&modulepath, "modulepath", dir, "Build Directory")
    flag.StringVar(&modulepath, "m", dir, "Build Directory (shorthand)")
}

func logError( err error ){
    if err != nil {
        log.Printf( "[ ERROR ] %v", err.Error() )
    }
}

func logFatal(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func transfer_progress_cb(stats git.TransferProgress) int {
    fmt.Printf("Recieved %v out of %v objects...\r", stats.ReceivedObjects , stats.TotalObjects)
    return 0
}

func cred_acquire_cb(url string,
                     username_from_url string,
                     allowed_types git.CredType) (int, *git.Cred) {
    status, cred := git.NewCredUserpassPlaintext(username, password)
    return status, &cred
}

func hub_clone(username, project, branch string) {
    dir, err := os.Getwd()
    logFatal(err)

    project_path := fmt.Sprintf("%v/%v", dir, project)
    repo_path := fmt.Sprintf("%v/%v", project_path, ".git")
    err = os.MkdirAll(repo_path, 0755)
    logFatal(err)

    github_repo_url := fmt.Sprintf("https://github.com/%v/%v", username, project)

    remote_cbs := &git.RemoteCallbacks{
        CredentialsCallback: cred_acquire_cb,
        TransferProgressCallback: transfer_progress_cb,
    }
    clone_options := &git.CloneOptions{
        Bare: false,
        RemoteCallbacks: remote_cbs,
        CheckoutBranch: branch,
    }
    _, err = git.Clone(github_repo_url, repo_path, clone_options)
    logFatal(err)
    if err == nil {
        log.Printf("[ INFO ] Cloned %v/%v at branch %v from GitHub\n", username, project, branch)
    }
}

func ( m *Module ) phuse() {
    m.retrieve()
    m.build()
    defer m.clean()
    m.install()
}
func ( ed *EnterpriseDist ) phuse() {
    ed.retrieve()
    ed.install()
}

func main() {
    cfg := struct {
        Enterprise_Dist EnterpriseDist
        Module map[string]*Module
    }{}
    gcfg.ReadFileInto( &cfg, "phusion.gcfg" )
    cfg.Enterprise_Dist.Path = fmt.Sprintf( "%v/enterprise-dist", modulepath )
    for module_name := range cfg.Module {
        cfg.Module[module_name].Name = module_name
        cfg.Module[module_name].Path = fmt.Sprintf( "%v/%v", modulepath, cfg.Module[module_name].Name )
    }
    enterprise_dist := cfg.Enterprise_Dist
    pe_module := cfg.Module["puppetlabs-puppet_enterprise"]
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "clean":
            enterprise_dist.destroy()
            pe_module.destroy()
        case "install":
            install()
        }
    } else {
        enterprise_dist.phuse()
        pe_module.phuse()
    }
}

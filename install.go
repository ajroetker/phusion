package main

import (
    ssh "code.google.com/p/go.crypto/ssh"
    "log"
    "io/ioutil"
    "os"
)

func install() {
    // Create the signer
    privKey, err := ioutil.ReadFile("/Users/aroetker/.ssh/id_rsa")
    if err != nil {
        log.Fatal(err)
    }
    encPrivKey, err := ssh.ParseRawPrivateKey(privKey)
    if err != nil {
        log.Fatal(err)
    }
    signer, err := ssh.NewSignerFromKey( encPrivKey )
    if err != nil {
        log.Fatal(err)
    }

    // Create client config
    config := &ssh.ClientConfig{
        User: "root",
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(signer),
            //Password("password"),
        },
    }
    // Connect to ssh server
    conn, err := ssh.Dial("tcp", "pe-all-in-one.aroetker.lan:22", config)
    if err != nil {
        log.Fatalf("unable to connect: %s", err)
    }
    defer conn.Close()
    // Create a session
    session, err := conn.NewSession()
    if err != nil {
        log.Fatalf("unable to create session: %s", err)
    }
    defer session.Close()
    // Once a Session is created, you can execute a single command on
    // the remote side using the Run method.
    session.Stdout = os.Stdout

    cmd := "puppet-enterprise-uninstaller -dpy -l /vagrant/uninstall.log && puppet-enterprise-installer -D -a /vagrant/answers/pe-all-in-one.answer -l /vagrant/pe-all-in-one-install.log"
    if err := session.Run(cmd); err != nil {
        panic("Failed to run: " + err.Error())
    }
}

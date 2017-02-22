package main

import (
	"io"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func SftpTransfer(t Transfer, file string) error {
	config := &ssh.ClientConfig{
		User:            t.User,
		HostKeyCallback: nil,
		Auth: []ssh.AuthMethod{
			ssh.Password(t.Password),
		},
	}
	config.SetDefaults()
	sshConn, err := ssh.Dial("tcp", t.Host, config)
	if err != nil {
		log.Error("connection failed: ", err)
		return err
	}
	defer sshConn.Close()

	client, err := sftp.NewClient(sshConn)
	if err != nil {
		log.Error("failed to create sftp client: ", err)
		return err
	}

	dstPath := client.Join(t.DestDir, filepath.Base(file))
	dstFile, err := client.Create(dstPath)
	if err != nil {
		log.Error("failed to create dest file: ", err)
		return err
	}
	defer dstFile.Close()

	srcFile, err := os.Open(file)
	if err != nil {
		log.Error("failed to open src file: ", err)
		return err
	}
	bytes, err := io.Copy(dstFile, io.LimitReader(srcFile, 300))
	//bytes, err := dstFile.ReadFrom(srcFile)
	if err != nil {
		log.Error("failed during copy: ", err)
		return err
	}
	log.Infof("successfully written %d bytes\n", bytes)

	srcFile.Close()
	err = os.Remove(file)
	if err != nil {
		log.Error("could not delete file: ", srcFile)
	}
	return nil
}

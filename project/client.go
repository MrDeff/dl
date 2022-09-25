package project

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/melbahja/goph"
	"github.com/pkg/sftp"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"github.com/varrcan/dl/helper"
	"github.com/varrcan/dl/utils/disk"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// SshClient represents ssh client
type SshClient struct {
	*goph.Client
	Server *Server
}

// Server config
type Server struct {
	Addr, Key, User, Catalog, FwType string
	UsePassword, UseKeyPassphrase    bool
	Port                             uint
}

// NewClient returns new client and error if any
func NewClient(server *Server) (c *SshClient, err error) {
	c = &SshClient{
		Server: server,
	}

	c.Client, err = goph.NewConn(&goph.Config{
		User:     c.Server.User,
		Addr:     c.Server.Addr,
		Port:     c.Server.Port,
		Auth:     getAuth(server),
		Callback: verifyHost,
	})

	return
}

func getAuth(server *Server) goph.Auth {
	if server.UsePassword {
		auth := goph.Password(askPass("Enter SSH Password: "))

		return auth
	} else {
		home, _ := helper.HomeDir()

		auth, err := goph.Key(filepath.Join(home, ".ssh", server.Key), getPassphrase(server.UseKeyPassphrase))
		if err != nil {
			pterm.FgRed.Println(err)
			return nil
		}

		return auth
	}
}

func getPassphrase(ask bool) string {
	if ask {
		return askPass("Enter Private Key Passphrase: ")
	}
	return ""
}

func askPass(msg string) string {
	fmt.Print(msg)
	pass, err := terminal.ReadPassword(0)
	if err != nil {
		panic(err)
	}

	fmt.Println("")

	return strings.TrimSpace(string(pass))
}

func verifyHost(host string, remote net.Addr, key ssh.PublicKey) error {
	hostFound, err := goph.CheckKnownHost(host, remote, key, "")

	// Host in known hosts but key mismatch
	if hostFound && err != nil {
		return err
	}

	// handshake because public key already exists
	if hostFound && err == nil {
		return nil
	}

	if askIsHostTrusted(host, key) == false {
		pterm.FgRed.Println("Connection aborted")
		return nil
	}

	return goph.AddKnownHost(host, remote, key, "")
}

func askIsHostTrusted(host string, key ssh.PublicKey) bool {
	reader := bufio.NewReader(os.Stdin)

	pterm.FgYellow.Printf("The authenticity of host %s can't be established \nFingerprint key: %s \n", host, ssh.FingerprintSHA256(key))
	pterm.FgYellow.Print("Are you sure you want to continue connecting (Y/n)? ")

	a, err := reader.ReadString('\n')
	if err != nil {
		pterm.FgRed.Println(err)
		return false
	}

	a = strings.TrimSpace(a)
	return strings.ToLower(a) == "y" || a == ""
}

// cleanRemote Deleting file on the server
func (c SshClient) cleanRemote(remotePath string) (err error) {
	ftp, err := c.NewSftp()
	if err != nil {
		return err
	}

	defer func(ftp *sftp.Client) {
		err := ftp.Close()
		if err != nil {
			pterm.FgRed.Println(err)
		}
	}(ftp)

	logrus.Infof("Delete file: %s", remotePath)
	err = ftp.Remove(remotePath)

	return err
}

// Download file from remote server
//goland:noinspection GoUnhandledErrorResult
func (c SshClient) download(ctx context.Context, remotePath, localPath string) (err error) {
	// w := progress.ContextWriter(ctx)
	local, err := os.Create(localPath)
	if err != nil {
		return
	}
	defer local.Close()

	ftp, err := c.NewSftp()
	if err != nil {
		return
	}
	defer ftp.Close()

	fileInfo, err := ftp.Stat(remotePath)
	if err != nil {
		return
	}
	defer ftp.Close()

	localDisk := disk.FreeSpaceHome()
	if fileInfo.Size() > int64(localDisk.Free) {
		remoteSize := disk.HumanSize(float64(fileInfo.Size()))
		localSize := disk.HumanSize(float64(localDisk.Free))
		return errors.New(fmt.Sprintf("No disk space. Filesize %s, free space %s", remoteSize, localSize))
	}

	remote, err := ftp.Open(remotePath)
	if err != nil {
		return
	}
	defer remote.Close()

	if _, err = io.Copy(local, remote); err != nil {
		return
	}

	return local.Sync()
}

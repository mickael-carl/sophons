package dialer

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"

	"github.com/mickael-carl/sophons/pkg/util"
)

func tempDirName() string {
	return "sophons-" + strconv.Itoa(rand.Intn(10000))
}

type dialer struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func NewDialer(host, port string, config *ssh.ClientConfig) (*dialer, error) {
	client, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", host, err)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return nil, fmt.Errorf("failed to create sftp client: %v", err)
	}

	return &dialer{
		sshClient:  client,
		sftpClient: sftpClient,
	}, nil
}

func (d *dialer) Close() {
	d.sftpClient.Close()
	d.sshClient.Close()
}

func (d *dialer) runCommand(command string) (string, error) {
	session, err := d.sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	return string(output), err
}

func (d *dialer) getRemoteOS() (string, error) {
	return d.runCommand("uname -s")
}

func (d *dialer) getRemoteArch() (string, error) {
	return d.runCommand("uname -m")
}

func (d *dialer) executerBinName() (string, error) {
	o, err := d.getRemoteOS()
	if err != nil {
		return "", nil
	}

	os := strings.ToLower(strings.TrimSpace(o))
	if os != "linux" && os != "darwin" {
		return "", fmt.Errorf("unsupported OS: %s", os)
	}

	a, err := d.getRemoteArch()
	if err != nil {
		return "", nil
	}

	arch := strings.TrimSpace(a)
	var binName string
	switch arch {
	case "amd64", "x86_64":
		binName = fmt.Sprintf("executer-%s-x86_64", os)
	case "arm64", "aarch64":
		binName = fmt.Sprintf("executer-%s-arm64", os)
	default:
		return "", errors.New("unsupported architecture")
	}

	return binName, nil
}

func (d *dialer) copyFile(localPath, remotePath string, executable bool) error {
	dstFile, err := d.sftpClient.Create(remotePath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	srcFile, err := os.Open(localPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	if executable {
		if err := d.sftpClient.Chmod(remotePath, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (d *dialer) copyExecuterBinary(localDir, remoteDir string) error {
	binName, err := d.executerBinName()
	if err != nil {
		return err
	}

	return d.copyFile(path.Join(localDir, binName), path.Join(remoteDir, "executer"), true)
}

func (d *dialer) Execute(host, binDir, inventory, playbook string) (string, error) {
	dirPath := path.Join("/tmp", tempDirName())
	if err := d.sftpClient.Mkdir(dirPath); err != nil {
		// TODO: don't crash mid-run, throw a warning.
		return "", fmt.Errorf("failed to create temporary directory on target host: %w", err)
	}
	defer d.sftpClient.RemoveAll(dirPath) //nolint:errcheck

	if err := d.copyExecuterBinary(binDir, dirPath); err != nil {
		return "", err
	}

	if err := d.copyFile(inventory, path.Join(dirPath, "inventory.yaml"), false); err != nil {
		return "", fmt.Errorf("failed to copy inventory to target host: %w", err)
	}

	// TODO: ansible looks in other places for roles.
	archivePath, err := util.Tar(filepath.Dir(playbook))
	if err != nil {
		return "", fmt.Errorf("failed to archive and copy %s to target host: %w", filepath.Dir(playbook), err)
	}

	if err := d.copyFile(archivePath, path.Join(dirPath, "data.tar.gz"), false); err != nil {
		return "", fmt.Errorf("failed to copy data from %s to target host: %w", archivePath, err)
	}

	session, err := d.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	playbookFileName := filepath.Base(playbook)
	playbookDirName := filepath.Base(filepath.Dir(playbook))
	// TODO: strings.Builder
	cmdLine := path.Join(dirPath, "executer")
	cmdLine += fmt.Sprintf(" -i %s", path.Join(dirPath, "inventory.yaml"))
	cmdLine += fmt.Sprintf(" -d %s", path.Join(dirPath, "data.tar.gz"))
	cmdLine += fmt.Sprintf(" -p %s", playbookDirName)
	cmdLine += fmt.Sprintf(" -n %s %s", host, path.Join(dirPath, playbookDirName, playbookFileName))

	return d.runCommand(cmdLine)
}

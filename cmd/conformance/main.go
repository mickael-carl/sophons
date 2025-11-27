package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	host     = "127.0.0.1"
	basePort = 2222
)

var (
	concurrency         int
	sophonsTestingImage string
	only                string
	list                bool
)

func main() {
	flag.IntVar(&concurrency, "concurrency", 4, "number of concurrent tests to run")
	flag.StringVar(&sophonsTestingImage, "testing-image", "sophons-testing:latest", "container image to use for tests")
	flag.StringVar(&only, "only", "", "comma-separated list of playbook names to run (e.g., playbook-apt.yaml,playbook-file.yaml)")
	flag.BoolVar(&list, "list", false, "list all available tests and exit")
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// TODO: a lot of this does a bunch of shelling out that's actually not
// necessary: docker has an API, generating SSH keys is bound to be possible in
// pure Go, etc.
func run() error {
	playbooks, err := filepath.Glob("data/playbooks/playbook*.yaml")
	if err != nil {
		return fmt.Errorf("failed to find playbooks: %w", err)
	}

	// List all available tests if -list flag is set
	if list {
		fmt.Printf("Available tests (%d):\n", len(playbooks))
		for _, playbook := range playbooks {
			fmt.Printf("  %s\n", filepath.Base(playbook))
		}
		return nil
	}

	log.Print("running conformance tests")

	// Filter playbooks if -only flag is set
	if only != "" {
		allowedNames := make(map[string]bool)
		for name := range strings.SplitSeq(only, ",") {
			allowedNames[strings.TrimSpace(name)] = true
		}

		var filteredPlaybooks []string
		for _, playbook := range playbooks {
			basename := filepath.Base(playbook)
			if allowedNames[basename] {
				filteredPlaybooks = append(filteredPlaybooks, playbook)
			}
		}
		playbooks = filteredPlaybooks

		if len(playbooks) == 0 {
			return fmt.Errorf("no playbooks matched the -only filter: %s", only)
		}
	}

	var wg sync.WaitGroup
	playbookCh := make(chan string)
	errCh := make(chan error, len(playbooks))

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			port := basePort + workerID
			for playbook := range playbookCh {
				log.Printf("running %s", playbook)
				if err := runConformanceTest(port, playbook); err != nil {
					errCh <- fmt.Errorf("test failed for playbook %s: %w", playbook, err)
				}
			}
		}(i)
	}

	for _, playbook := range playbooks {
		playbookCh <- playbook
	}
	close(playbookCh)

	wg.Wait()
	close(errCh)

	var allErrors []string
	for err := range errCh {
		allErrors = append(allErrors, err.Error())
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("the following tests failed:\n%s", strings.Join(allErrors, "\n"))
	}

	return nil
}

func runConformanceTest(port int, playbook string) error {
	dir, err := os.MkdirTemp("/tmp", "sophons-conformance")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	sshKeyPath := filepath.Join(dir, "testing")
	if err := runCommand(nil, "ssh-keygen", "-f", sshKeyPath, "-N", ""); err != nil {
		return err
	}

	portStr := strconv.Itoa(port)

	sophonsTarPath := filepath.Join(dir, "sophons.tar")
	if err := runPlaybook(dir, sshKeyPath, playbook, sophonsTarPath, "sophons", portStr); err != nil {
		return err
	}

	ansibleTarPath := filepath.Join(dir, "ansible.tar")
	if err := runPlaybook(dir, sshKeyPath, playbook, ansibleTarPath, "ansible", portStr); err != nil {
		return err
	}

	return runCommand(nil, "./bin/tardiff", ansibleTarPath, sophonsTarPath)
}

func containerCleanup(sha string) {
	if err := runCommand(nil, "docker", "stop", sha); err != nil {
		log.Printf("failed to stop container %s: %v", sha, err)
	}
	if err := runCommand(nil, "docker", "rm", sha); err != nil {
		log.Printf("failed to remove container %s: %v", sha, err)
	}
}

func runPlaybook(dir, sshKeyPath, playbook, tarPath, mode, port string) error {
	sha, err := startContainer(port)
	if err != nil {
		return err
	}
	defer containerCleanup(sha)

	knownHostsPath := filepath.Join(dir, "known_hosts")
	if err := setupKnownHosts(knownHostsPath, port); err != nil {
		return err
	}

	if err := runCommand(nil, "docker", "cp", sshKeyPath+".pub", sha+":/root/.ssh/authorized_keys"); err != nil {
		return err
	}
	if err := runCommand(nil, "docker", "exec", sha, "chown", "root:root", "/root/.ssh/authorized_keys"); err != nil {
		return err
	}

	switch mode {
	case "sophons":
		err = runCommand(nil, "./bin/dialer", "-p", port, "-b", "bin/", "-i", "data/inventory-testing.yaml", "-k", sshKeyPath, "-u", "root", "--known-hosts", knownHostsPath, playbook)
	case "ansible":
		controlPathDir := filepath.Join(dir, "ansible-cp")
		if err := os.Mkdir(controlPathDir, 0o755); err != nil {
			return err
		}
		env := map[string]string{
			"ANSIBLE_SSH_CONTROL_PATH_DIR": controlPathDir,
		}
		err = runCommand(env, "ansible-playbook", "--key-file", sshKeyPath, "-u", "root", "-i", "data/inventory-testing.yaml", "--ssh-common-args=-p "+port+" -o UserKnownHostsFile="+knownHostsPath, playbook)
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
	if err != nil {
		return err
	}

	return runCommand(nil, "docker", "export", "-o", tarPath, sha)
}

func startContainer(port string) (string, error) {
	cmd := exec.Command("docker", "run", "-d", "-p", "127.0.0.1:"+port+":22", sophonsTestingImage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w\n%s", err, output)
	}
	outputStr := strings.TrimSpace(string(output))
	outputLines := strings.Split(outputStr, "\n")
	sha := outputLines[len(outputLines)-1]

	// Wait for the SSH server to be ready.
	address := fmt.Sprintf("127.0.0.1:%s", port)
	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1 * time.Second,
	}

	for range 20 {
		client, err := ssh.Dial("tcp", address, config)
		if err != nil {
			// An error containing "unable to authenticate" means the server is up
			// and is rejecting our (non-existent) auth. That's good enough
			// for us.
			if strings.Contains(err.Error(), "unable to authenticate") {
				return sha, nil
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		client.Close()
		return sha, nil
	}

	// If we reach here, we couldn't connect.
	// Stop and remove the container to cleanup.
	containerCleanup(sha)
	return "", fmt.Errorf("could not connect to container's SSH server at %s", address)
}

func setupKnownHosts(knownHostsPath, port string) error {
	cmd := exec.Command("ssh-keyscan", "-p", port, host)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run ssh-keyscan: %w\n%s", err, output)
	}

	var result strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, host+" ") {
			line = strings.Replace(line, host+" ", fmt.Sprintf("[%s]:%s ", host, port), 1)
		}
		result.WriteString(line)
		result.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning ssh-keyscan output: %w", err)
	}

	return os.WriteFile(knownHostsPath, []byte(result.String()), 0o644)
}

func runCommand(env map[string]string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if env != nil {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run command '%s %s': %w\n%s", name, strings.Join(arg, " "), err, output)
	}
	return nil
}

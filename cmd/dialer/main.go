package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/goccy/go-yaml"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/mickael-carl/sophons/pkg/dialer"
	"github.com/mickael-carl/sophons/pkg/inventory"
)

var (
	username      = flag.String("u", "", "username to connect to hosts")
	keyPath       = flag.String("k", "", "path to the SSH key to use")
	inventoryPath = flag.String("i", "", "path to inventory file")
	sshPort       = flag.String("p", "22", "port to use for SSH")
	// TODO: we can use //go:embed here to embed all necessary binaries? That
	// might make the dialer extremely large though: each executer binary is
	// about 4.5MB right now, meaning a ~30MB binary total. It's not horrible
	// but it's worth keeping in mind as binary size increases.
	binDir         = flag.String("b", "", "dir containing executer binaries")
	knownHostsPath = flag.String("known-hosts", os.ExpandEnv("$HOME/.ssh/known_hosts"), "path to the known hosts file")
)

func sshConfig(u, k, knownHosts string) (*ssh.ClientConfig, error) {
	key, err := os.ReadFile(k)
	if err != nil {
		return &ssh.ClientConfig{}, fmt.Errorf("failed reading private key %q: %v", k, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return &ssh.ClientConfig{}, fmt.Errorf("failed parsing private key: %v", err)
	}

	hostKeyCallback, err := knownhosts.New(knownHosts)
	if err != nil {
		return &ssh.ClientConfig{}, fmt.Errorf("could not create hostkey callback from %s: %v", knownHosts, err)
	}

	return &ssh.ClientConfig{
		User: u,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}, nil
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatal("missing playbook path. Usage: dialer -b binary-directory -i inventory.yaml playbook.yaml")
	}

	if *binDir == "" {
		log.Fatal("`-b` flag is required")
	}

	if *inventoryPath == "" {
		log.Fatal("`-i` flag is required")
	}

	inventoryData, err := os.ReadFile(*inventoryPath)
	if err != nil {
		log.Fatalf("failed to read inventory from %s: %v", *inventoryPath, err)
	}

	var inventory inventory.Inventory
	if err := yaml.Unmarshal(inventoryData, &inventory); err != nil {
		log.Fatalf("failed to unmarshal inventory from %s: %v", *inventoryPath, err)
	}
	hosts := inventory.All()

	config, err := sshConfig(*username, *keyPath, *knownHostsPath)
	if err != nil {
		log.Fatalf("failed to create SSH config with username %s and key from %s: %v", *username, *keyPath, err)
	}

	for host := range hosts {
		dialer, err := dialer.NewDialer(host, *sshPort, config)
		if err != nil {
			log.Fatalf("failed to create dialer for %s:%s: %v", host, *sshPort, err)
		}

		out, err := dialer.Execute(host, *binDir, *inventoryPath, flag.Args()[0])
		// Output regardless of error: stderr is in `out` as well. Also close
		// everything before crashing if needed.
		fmt.Println(string(out))
		dialer.Close()
		if err != nil {
			log.Fatalf("running sophons against %s failed: %v", host, err)
		}
	}
}

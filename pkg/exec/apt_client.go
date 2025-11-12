package exec

import "github.com/arduino/go-apt-client"

//go:generate mockgen -source=$GOFILE -destination=mock_apt_client_test.go -package=exec

type aptClient interface {
	CheckForUpdates() (string, error)
	Clean() (string, error)
	DistUpgrade() (string, error)
	Install(pkgs ...*apt.Package) (string, error)
	ListInstalled() ([]*apt.Package, error)
	Remove(pkgs ...*apt.Package) (string, error)
	UpgradeAll() (string, error)
}

type realAptClient struct{}

func (c *realAptClient) CheckForUpdates() (string, error) {
	out, err := apt.CheckForUpdates()
	return string(out), err
}

func (c *realAptClient) Clean() (string, error) {
	out, err := apt.Clean()
	return string(out), err
}

func (c *realAptClient) DistUpgrade() (string, error) {
	out, err := apt.DistUpgrade()
	return string(out), err
}

func (c *realAptClient) Install(pkgs ...*apt.Package) (string, error) {
	out, err := apt.Install(pkgs...)
	return string(out), err
}

func (c *realAptClient) ListInstalled() ([]*apt.Package, error) {
	return apt.ListInstalled()
}

func (c *realAptClient) Remove(pkgs ...*apt.Package) (string, error) {
	out, err := apt.Remove(pkgs...)
	return string(out), err
}

func (c *realAptClient) UpgradeAll() (string, error) {
	out, err := apt.UpgradeAll()
	return string(out), err
}

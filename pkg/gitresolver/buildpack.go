package gitresolver

import (
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/pkg/errors"
)

// InitBuildPack initialises the build pack URL and git ref returning the packs dir or an error
func InitBuildPack(gitter gitclient.Interface, packURL string, packRef string) (string, error) {
	dir, err := gitclient.CloneToDir(gitter, packURL, "")
	if err != nil {
		return "", err
	}
	if packRef != "master" && packRef != "main" && packRef != "" {
		err = gitclient.Checkout(gitter, dir, packRef)
		if err != nil {
			return "", errors.Wrapf(err, "failed to checkout %s of %s in dir %s", packRef, packURL, dir)
		}
	}
	return filepath.Join(dir, "packs"), nil
}

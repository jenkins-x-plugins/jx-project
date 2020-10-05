package gitresolver

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/pkg/errors"
)

// InitBuildPack initialises the build pack URL and git ref returning the packs dir or an error
func InitBuildPack(gitter gitclient.Interface, packURL string, packRef string) (string, error) {
	dir, err := gitclient.CloneToDir(gitter, packURL, "")
	if err != nil {
		return "", err
	}
	if packRef != "master" && packRef != "" {
		err = gitclient.FetchTags(gitter, dir)
		if err != nil {
			return "", errors.Wrapf(err, "fetching tags from %s", packURL)
		}
		tags, err := gitclient.FilterTags(gitter, dir, packRef)
		if err != nil {
			return "", errors.Wrapf(err, "filtering tags for %s", packRef)
		}
		if len(tags) == 0 {
			tags, err = gitclient.FilterTags(gitter, dir, fmt.Sprintf("v%s", packRef))
			if err != nil {
				return "", errors.Wrapf(err, "filtering tags for v%s", packRef)
			}
		}
		if len(tags) == 1 {
			tag := tags[0]
			branchName := fmt.Sprintf("tag-%s", tag)
			err = gitclient.CreateBranchFrom(gitter, dir, branchName, tag)
			if err != nil {
				return "", errors.Wrapf(err, "creating branch %s from %s", branchName, tag)
			}
			err = gitclient.Checkout(gitter, dir, branchName)
			if err != nil {
				return "", errors.Wrapf(err, "checking out branch %s", branchName)
			}
		} else {
			if len(tags) > 1 {
				log.Logger().Debugf("more than one tag matched %s or v%s, ignoring tags", packRef, packRef)
			}
			err = gitclient.CheckoutRemoteBranch(gitter, dir, packRef)
			if err != nil {
				return "", errors.Wrapf(err, "checking out tracking branch %s", packRef)
			}
		}

	}
	return filepath.Join(dir, "packs"), nil
}

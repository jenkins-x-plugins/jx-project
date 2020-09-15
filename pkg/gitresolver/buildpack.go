package gitresolver

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-logging/pkg/log"

	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

// InitBuildPack initialises the build pack URL and git ref returning the packs dir or an error
func InitBuildPack(gitter gitclient.Interface, packURL string, packRef string) (string, error) {
	u, err := url.Parse(strings.TrimSuffix(packURL, ".git"))
	if err != nil {
		return "", fmt.Errorf("Failed to parse build pack URL: %s: %s", packURL, err)
	}

	draftDir, err := util.DraftDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(draftDir, "packs", u.Host, u.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("Could not create %s: %s", dir, err)
	}

	err = ensureBranchTracksOrigin(dir, packRef, gitter)
	if err != nil {
		return "", errors.Wrapf(err, "there was a problem ensuring the branch %s has tracking info", packRef)
	}

	err = gitclient.CloneOrPull(gitter, packURL, dir)
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

func ensureBranchTracksOrigin(dir string, packRef string, gitter gitclient.Interface) error {
	empty, err := util.IsEmpty(dir)
	if err != nil {
		return errors.Wrapf(err, "there was a problem checking if %s is empty", dir)
	}

	// The repository is cloned, before the pull, we have to make sure we fetch & checkout <packRef> and we are tracking origin/<packRef>
	// This is due to a bug happening on old clones done by the old cloning func
	if !empty {
		err := gitclient.FetchBranch(gitter, dir, "origin", packRef)
		if err != nil {
			return err
		}
		err = gitclient.Checkout(gitter, dir, packRef)
		if err != nil {
			return err
		}
		err = gitclient.SetUpstreamTo(gitter, dir, packRef)
		if err != nil {
			return errors.Wrapf(err, "there was a problem setting upstream to remote branch origin/%s", packRef)
		}
	}

	return nil
}

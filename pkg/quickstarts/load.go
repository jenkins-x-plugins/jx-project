package quickstarts

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

type Options struct {
	VersionsDir string
	Namespace   string
	CurrentUser string
	JXClient    versioned.Interface
	ScmClient   *scm.Client
}

func (o *Options) Validate() error {
	var err error
	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create the jx client")
	}
	return nil
}

// LoadQuickStartsModel Load all quickstarts
func (o *Options) LoadQuickStartsModel(gitHubOrganisations []string, ignoreTeam bool) (*QuickstartModel, error) {
	err := o.Validate()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate options")
	}

	locations, err := o.loadQuickStartLocations(gitHubOrganisations, ignoreTeam)
	if err != nil {
		return nil, err
	}

	model, err := o.LoadQuickStartsFromLocations(locations)
	if err != nil {
		return nil, fmt.Errorf("failed to load qs: %s", err)
	}

	devGitDir := filepath.Join(o.VersionsDir, "..")
	quickstartsFile := filepath.Join(devGitDir, "extensions", v1alpha1.QuickstartsFileName)
	exists, err := files.FileExists(quickstartsFile)
	if err != nil {
		return model, errors.Wrapf(err, "failed to check if file exists %s", quickstartsFile)
	}
	if !exists {
		// lets default to using the version stream file
		versionStreamFile := filepath.Join(o.VersionsDir, v1alpha1.QuickstartsFileName)
		exists, err = files.FileExists(versionStreamFile)
		if err != nil {
			return model, errors.Wrapf(err, "failed to check if file exists %s", versionStreamFile)
		}

		if !exists {
			return model, errors.Errorf("development git repository does not contain quickstarts file %s", versionStreamFile)
		}
		quickstartsFile = versionStreamFile
	}

	quickstarts := &v1alpha1.Quickstarts{}
	err = yamls.LoadFile(quickstartsFile, quickstarts)
	if err != nil {
		return model, errors.Wrapf(err, "failed to parse %s", quickstartsFile)
	}

	err = model.LoadQuickStarts(&quickstarts.Spec, devGitDir, quickstartsFile)
	if err != nil {
		return model, errors.Wrapf(err, "loading quickstarts from %s", quickstartsFile)
	}
	return model, nil
}

// LoadQuickStartsFromLocations Load all quickstarts from the given locatiotns
func (o *Options) LoadQuickStartsFromLocations(locations []v1.QuickStartLocation) (*QuickstartModel, error) {
	err := o.Validate()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate options")
	}
	gitMap := map[string]map[string]v1.QuickStartLocation{}
	for _, loc := range locations {
		m := gitMap[loc.GitURL]
		if m == nil {
			m = map[string]v1.QuickStartLocation{}
			gitMap[loc.GitURL] = m
		}
		m[loc.Owner] = loc
	}
	model := NewQuickstartModel()

	/* TODO

	for gitURL, m := range gitMap {
		for _, location := range m {
			kind := location.GitKind
			if kind == "" {
				kind = gits.KindGitHub
			}

			// If this is a default quickstart location but there's no github.com credentials, skip and rely on the version stream alone.
			if kube.IsDefaultQuickstartLocation(location) && o.ScmClient == nil) {
				continue
			}
			gitProvider, err := o.GitProviderForGitServerURL(gitURL, kind, "")
			if err != nil {
				return model, err
			}
			log.Logger().Debugf("Searching for repositories in Git server %s owner %s includes %s excludes %s as user %s ", gitProvider.ServerURL(), location.Owner, strings.Join(location.Includes, ", "), strings.Join(location.Excludes, ", "), o.CurrentUsername)
			err = model.LoadGithubQuickstarts(gitProvider, location.Owner, location.Includes, location.Excludes)
			if err != nil {
				log.Logger().Debugf("Quickstart load error: %s", err.Error())
			}

		}
	}
	*/
	return model, nil
}

// loadQuickStartLocations loads the quickstarts
func (o *Options) loadQuickStartLocations(gitHubOrganisations []string, ignoreTeam bool) ([]v1.QuickStartLocation, error) {
	var locations []v1.QuickStartLocation

	/* TODO
	if !ignoreTeam {
		jxClient := o.JXClient
		ns := o.Namespace

		var err error
		locations, err = kube.GetQuickstartLocations(jxClient, ns)
		if err != nil {
			return nil, err
		}
	}
	*/
	// lets add any extra github organisations if they are not already configured
	for _, org := range gitHubOrganisations {
		found := false
		for _, loc := range locations {
			if loc.GitURL == giturl.GitHubURL && loc.Owner == org {
				found = true
				break
			}
		}
		if !found {
			locations = append(locations, v1.QuickStartLocation{
				GitURL:   giturl.GitHubURL,
				GitKind:  giturl.KindGitHub,
				Owner:    org,
				Includes: []string{"*"},
				Excludes: []string{"WIP-*"},
			})
		}
	}
	return locations, nil
}

// LoadMLProjectSetsModel Load all quickstarts
func (o *Options) LoadMLProjectSetsModel(gitHubOrganisations []string, ignoreTeam bool) (*QuickstartModel, error) {
	err := o.Validate()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate options")
	}
	log.Logger().Debugf("Valid options\n")

	locations, err := o.loadQuickStartLocations(gitHubOrganisations, ignoreTeam)
	if err != nil {
		return nil, err
	}
	log.Logger().Debugf("Locations: %s\n", locations)

	model, err := o.LoadQuickStartsFromLocations(locations)
	if err != nil {
		return nil, fmt.Errorf("failed to load qs: %s", err)
	}
	log.Logger().Debugf("Model: %s\n", model)

	devGitDir := filepath.Join(o.VersionsDir, "..")
	log.Logger().Debugf("devGitDir: %s\n", devGitDir)
	quickstartsFile := filepath.Join(devGitDir, "extensions", v1alpha1.MLProjectSetsFileName)
	log.Logger().Debugf("quickstartsFile: %s\n", quickstartsFile)
	exists, err := files.FileExists(quickstartsFile)
	if err != nil {
		return model, errors.Wrapf(err, "failed to check if file exists %s", quickstartsFile)
	}
	if !exists {
		log.Logger().Debugf("No quickstarts file so trying versionStream\n")
		// lets default to using the version stream file
		versionStreamFile := filepath.Join(o.VersionsDir, v1alpha1.MLProjectSetsFileName)
		log.Logger().Debugf("versionStreamFile: %s\n", versionStreamFile)
		exists, err = files.FileExists(versionStreamFile)
		if err != nil {
			return model, errors.Wrapf(err, "failed to check if file exists %s", versionStreamFile)
		}

		if !exists {
			return model, errors.Errorf("development git repository does not contain quickstarts file %s", versionStreamFile)
		}
		log.Logger().Debugf("Using versionStreamFile\n")
		quickstartsFile = versionStreamFile
	}

	quickstarts := &v1alpha1.Quickstarts{}
	log.Logger().Debugf("Loading quickstartsFile...\n")
	err = yamls.LoadFile(quickstartsFile, quickstarts)
	if err != nil {
		return model, errors.Wrapf(err, "failed to parse %s", quickstartsFile)
	}

	log.Logger().Debugf("Loading model...\n")
	err = model.LoadQuickStarts(&quickstarts.Spec, devGitDir, quickstartsFile)
	if err != nil {
		return model, errors.Wrapf(err, "loading quickstarts from %s", quickstartsFile)
	}
	log.Logger().Debugf("Model: %s\n", model)
	return model, nil
}

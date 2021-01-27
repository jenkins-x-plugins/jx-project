package importcmd

import (
	"fmt"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/lighthouse-client/pkg/config/job"
	"github.com/jenkins-x/lighthouse-client/pkg/triggerconfig"
	"github.com/jenkins-x/lighthouse-client/pkg/triggerconfig/inrepo"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/gitdiscovery"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

var (
	kptFile = `apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
    name: %s
upstream:
    type: git
    git:
        commit: %s
        repo: %s
        directory: %s
        ref: master
`
)

// createMissingLighthouseKptFiles lets create any missing Kptfile for any .lighthouse/somedir directories
// so that after the pipeline folder has been added we can later on upgrade it from its source via kpt
func (o *ImportOptions) createMissingLighthouseKptFiles(lighthouseDir, packName string) error {
	fileSlice, err := ioutil.ReadDir(lighthouseDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %s", lighthouseDir)
	}
	for _, f := range fileSlice {
		if !f.IsDir() {
			continue
		}
		name := f.Name()

		dir := filepath.Join(lighthouseDir, name)
		triggerFile := filepath.Join(dir, "triggers.yaml")
		exists, err := files.FileExists(triggerFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", triggerFile)
		}
		if !exists {
			continue
		}

		// lets check if we have a local Kptfile for this trigger folder
		localKptDir := filepath.Join(o.Dir, ".lighthouse", name)
		localKptFile := filepath.Join(localKptDir, "Kptfile")
		exists, err = files.FileExists(localKptFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", localKptFile)
		}
		if exists {
			continue
		}

		hasUses, err := CheckForUsesImage(dir, triggerFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check for image: uses:sourceURI")
		}
		if hasUses {
			continue
		}

		sha, err := gitclient.GetLatestCommitSha(o.Git(), lighthouseDir)
		if err != nil {
			return errors.Wrapf(err, "failed to discover latest git commit for dir %s", lighthouseDir)
		}

		gitURL, err := gitdiscovery.FindGitURLFromDir(lighthouseDir, true)
		if err != nil {
			return errors.Wrapf(err, "failed to discover git URL in dir %s", lighthouseDir)
		}

		// lets remove any user/passwords just in case
		gitURL = stringhelpers.SanitizeURL(gitURL)

		if gitURL == "" {
			return errors.Errorf("failed to find git URL in dir %s", lighthouseDir)
		}

		err = os.MkdirAll(localKptDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create dir %s", localKptDir)
		}

		fromDir := filepath.Join("/packs", packName, ".lighthouse", name)
		gitURL = strings.TrimSuffix(gitURL, ".git")
		text := fmt.Sprintf(kptFile, name, sha, gitURL, fromDir)
		err = ioutil.WriteFile(localKptFile, []byte(text), files.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save file %s", localKptFile)
		}

		log.Logger().Infof("created %s", localKptFile)
	}
	return nil
}

// CheckForUsesImage checks if the given dir and trigger file has a uses: image
// which if present assumes we are using uses: inheritance rather than kpt
func CheckForUsesImage(dir, triggersFile string) (bool, error) {
	repoConfig := &triggerconfig.Config{}
	err := yamls.LoadFile(triggersFile, repoConfig)
	if err != nil {
		return false, errors.Wrapf(err, "failed to load lighthouse triggers: %s", triggersFile)
	}

	for i := range repoConfig.Spec.Presubmits {
		r := &repoConfig.Spec.Presubmits[i]
		if r.SourcePath != "" {
			path := filepath.Join(dir, r.SourcePath)
			flag, err := loadJobBaseFromSourcePath(path)
			if err != nil {
				log.Logger().Warnf("failed to load file %s", path)
			}
			if flag {
				return true, nil
			}
		}
		if r.Agent == "" && r.PipelineRunSpec != nil {
			r.Agent = job.TektonPipelineAgent
		}
	}
	for i := range repoConfig.Spec.Postsubmits {
		r := &repoConfig.Spec.Postsubmits[i]
		if r.SourcePath != "" {
			path := filepath.Join(dir, r.SourcePath)
			flag, err := loadJobBaseFromSourcePath(path)
			if err != nil {
				log.Logger().Warnf("failed to load file %s", path)
			}
			if flag {
				return true, nil
			}
		}
	}
	return false, nil
}

func loadJobBaseFromSourcePath(path string) (bool, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to load file %s", path)
	}
	if len(data) == 0 {
		return false, errors.Errorf("empty file file %s", path)
	}

	pr, err := LoadTektonResourceAsPipelineRun(data, "for file "+path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to unmarshal YAML file %s", path)
	}
	ps := pr.Spec.PipelineSpec
	if ps == nil {
		return false, nil
	}
	for i := range ps.Tasks {
		task := &ps.Tasks[i]
		if task.TaskSpec == nil {
			continue
		}
		ts := task.TaskSpec.TaskSpec
		if ts.StepTemplate != nil && isUsesImage(ts.StepTemplate.Image) {
			return true, nil
		}
		steps := ts.Steps
		for _, step := range steps {
			if isUsesImage(step.Image) {
				return true, nil
			}
		}
	}
	return false, nil
}

func isUsesImage(image string) bool {
	return strings.HasPrefix(image, "uses:")
}

// LoadTektonResourceAsPipelineRun loads a PipelineRun, Pipeline, Task or TaskRun and convert it to a PipelineRun
func LoadTektonResourceAsPipelineRun(data []byte, message string) (*tektonv1beta1.PipelineRun, error) {
	defaultValues, err := inrepo.NewDefaultValues()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse default values")
	}

	kindPrefix := "kind:"
	kind := "PipelineRun"
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, kindPrefix) {
			continue
		}
		k := strings.TrimSpace(line[len(kindPrefix):])
		if k != "" {
			kind = k
			break
		}
	}
	switch kind {
	case "Pipeline":
		pipeline := &tektonv1beta1.Pipeline{}
		err := yaml.Unmarshal(data, pipeline)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal Pipeline YAML %s", message)
		}
		return inrepo.ConvertPipelineToPipelineRun(pipeline, message, defaultValues)

	case "PipelineRun":
		prs := &tektonv1beta1.PipelineRun{}
		err := yaml.Unmarshal(data, prs)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal PipelineRun YAML %s", message)
		}
		return prs, nil

	case "Task":
		task := &tektonv1beta1.Task{}
		err := yaml.Unmarshal(data, task)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal Task YAML %s", message)
		}
		return inrepo.ConvertTaskToPipelineRun(task, message, defaultValues)

	case "TaskRun":
		tr := &tektonv1beta1.TaskRun{}
		err := yaml.Unmarshal(data, tr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal TaskRun YAML %s", message)
		}
		return inrepo.ConvertTaskRunToPipelineRun(tr, message, defaultValues)

	default:
		return nil, errors.Errorf("kind %s is not supported for %s", kind, message)
	}
}

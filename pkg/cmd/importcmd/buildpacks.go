package importcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-project/pkg/config"
	"github.com/jenkins-x-plugins/jx-project/pkg/gitresolver"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"

	jxdraft "github.com/jenkins-x-plugins/jx-project/pkg/draft"
	"github.com/jenkins-x-plugins/jx-project/pkg/jenkinsfile"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// InvokeDraftPack used to pass arguments into the draft pack invocation
type InvokeDraftPack struct {
	Dir                         string
	DevEnvCloneDir              string
	CustomDraftPack             string
	Jenkinsfile                 string
	InitialisedGit              bool
	DisableAddFiles             bool
	UseNextGenPipeline          bool
	CreateJenkinsxYamlIfMissing bool
	ProjectConfig               *config.ProjectConfig
}

// InitBuildPacks initialise the build packs
func (o *ImportOptions) InitBuildPacks(i *InvokeDraftPack) (string, *v1.TeamSettings, error) {
	bp, settings, err := o.PickPipelineCatalog(i)
	if err != nil {
		return "", settings, err
	}

	if o.PipelineCatalogDir != "" {
		log.Logger().Infof("using the pipeline catalog dir %s", termcolor.ColorInfo(o.PipelineCatalogDir))
		return o.PipelineCatalogDir, settings, err
	}
	dir, err := gitresolver.InitBuildPack(o.Git(), bp.GitURL, bp.GitRef)
	return dir, settings, err
}

// CloneDevEnvironment clones the development environment to a directory
func (o *ImportOptions) CloneDevEnvironment() (string, error) {
	if o.DevEnv == nil {
		return "", errors.Errorf("no Dev Environment")
	}
	devEnvGitURL := o.DevEnv.Spec.Source.URL
	if devEnvGitURL == "" {
		return "", errors.Errorf("no spec.source.url for dev environment so cannot clone the version stream")
	}
	devEnvCloneDir, err := gitclient.CloneToDir(o.Git(), devEnvGitURL, "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to clone dev environment git repository %s", devEnvGitURL)
	}
	return devEnvCloneDir, nil
}

// PickPipelineCatalog lets you pick a build pack
func (o *ImportOptions) PickPipelineCatalog(i *InvokeDraftPack) (*v1alpha1.PipelineCatalogSource, *v1.TeamSettings, error) {
	if o.DevEnv == nil {
		return nil, nil, errors.Errorf("no Dev Environment")
	}
	devEnvCloneDir := i.DevEnvCloneDir
	if devEnvCloneDir == "" {
		return nil, nil, errors.Errorf("no Dev Environment git clone dir")
	}
	settings := &o.DevEnv.Spec.TeamSettings
	pipelineCatalogsFile := filepath.Join(devEnvCloneDir, "extensions", v1alpha1.PipelineCatalogFileName)
	exists, err := files.FileExists(pipelineCatalogsFile)
	if err != nil {
		return nil, settings, errors.Wrapf(err, "failed to check if file exists %s", pipelineCatalogsFile)
	}

	pipelineCatalog := &v1alpha1.PipelineCatalog{}

	if exists {
		err = yamls.LoadFile(pipelineCatalogsFile, pipelineCatalog)
		if err != nil {
			return nil, settings, errors.Wrapf(err, "failed to load PipelineCatalog file %s", pipelineCatalogsFile)
		}
	}

	if len(pipelineCatalog.Spec.Repositories) == 0 {
		// lets add the default pipeline catalog

		defaultCatalog := v1alpha1.PipelineCatalogSource{
			ID:     "default-pipeline-catalog",
			Label:  "Cluster Pipeline Catalog",
			GitURL: "",
			GitRef: "",
		}
		if defaultCatalog.GitURL == "" {
			defaultCatalog.GitURL = "https://github.com/jenkins-x/jx3-pipeline-catalog"
		}
		pipelineCatalog.Spec.Repositories = append(pipelineCatalog.Spec.Repositories, defaultCatalog)
	}

	m := map[string]*v1alpha1.PipelineCatalogSource{}
	names := []string{}
	defaultValue := ""
	for i := range pipelineCatalog.Spec.Repositories {
		pc := &pipelineCatalog.Spec.Repositories[i]
		if pc.Label == "" {
			pc.Label = pc.ID
			if pc.Label == "" {
				pc.Label = pc.GitURL
			}
		}
		label := pc.Label
		if defaultValue == "" {
			defaultValue = label
		}
		names = append(names, label)
		m[label] = pc
	}
	sort.Strings(names)

	if o.BatchMode {
		pc := &pipelineCatalog.Spec.Repositories[0]
		return pc, settings, nil
	}

	name, err := o.Input.PickNameWithDefault(names, "Pick the pipeline catalog you would like to use", defaultValue,
		"the pipeline catalog folder contains the tekton pipelines and associated files")
	if err != nil {
		return nil, settings, errors.Wrap(err, "failed to pick the build pack name")
	}
	return m[name], settings, err
}

// InvokeDraftPack invokes a draft pack copying in a Jenkinsfile if required
func (o *ImportOptions) InvokeDraftPack(i *InvokeDraftPack) (string, error) {
	packsDir, _, err := o.InitBuildPacks(i)
	if err != nil {
		return "", err
	}

	// let's assume Jenkins X import mode
	//
	// was:
	// lets configure the draft pack mode based on the team settings
	// if settings.GetImportMode() != v1.ImportModeTypeJenkinsfile {
	i.UseNextGenPipeline = true
	i.CreateJenkinsxYamlIfMissing = true

	dir := i.Dir
	customDraftPack := i.CustomDraftPack

	pomName := filepath.Join(dir, "pom.xml")
	gradleName := filepath.Join(dir, "build.gradle")
	jenkinsPluginsName := filepath.Join(dir, "plugins.txt")
	packagerConfigName := filepath.Join(dir, "packager-config.yml")
	jenkinsxYaml := filepath.Join(dir, config.ProjectConfigFileName)
	envChart := filepath.Join(dir, "env", "Chart.yaml")
	lpack := ""
	if customDraftPack == "" {
		if i.ProjectConfig == nil {
			i.ProjectConfig, _, err = config.LoadProjectConfig(dir)
			if err != nil {
				return "", err
			}
		}
		customDraftPack = i.ProjectConfig.BuildPack
	}

	if len(customDraftPack) > 0 {
		log.Logger().Infof("trying to use draft pack: %s", customDraftPack)
		lpack = filepath.Join(packsDir, customDraftPack)
		f, err := files.DirExists(lpack)
		if err != nil {
			log.Logger().Error(err.Error())
			return "", err
		}
		if !f {
			log.Logger().Error("Could not find pack: " + customDraftPack + " going to try detect which pack to use")
			lpack = ""
		}
	}

	if lpack == "" {
		if exists, err := files.FileExists(pomName); err == nil && exists {
			pack, err := PomFlavour(packsDir, pomName)
			if err != nil {
				return "", err
			}
			lpack = filepath.Join(packsDir, pack)

			exists, _ = files.DirExists(lpack)
			if !exists {
				log.Logger().Warn("defaulting to maven pack")
				lpack = filepath.Join(packsDir, "maven")
			}
		} else if exists, err := files.FileExists(gradleName); err == nil && exists {
			lpack = filepath.Join(packsDir, "gradle")
		} else if exists, err := files.FileExists(jenkinsPluginsName); err == nil && exists {
			lpack = filepath.Join(packsDir, "jenkins")
		} else if exists, err := files.FileExists(packagerConfigName); err == nil && exists {
			lpack = filepath.Join(packsDir, "cwp")
		} else if exists, err := files.FileExists(envChart); err == nil && exists {
			lpack = filepath.Join(packsDir, "environment")
		} else {
			// pack detection time
			lpack, err = jxdraft.DoPackDetectionForBuildPack(os.Stdout, dir, packsDir)

			if err != nil {
				if lpack == "" {
					// let's detect docker and/or helm

					// TODO one day when our pipelines can include steps conditional on the presence of a file glob
					// we can just use a single docker/helm package that does docker and/or helm
					// but for now we've 3 separate packs for docker, docker-helm and helm
					hasDocker := false
					hasHelm := false

					if exists, err2 := files.FileExists(filepath.Join(dir, "Dockerfile")); err2 == nil && exists {
						hasDocker = true
					}

					// let's check for a helm pack
					glob, err2 := filepath.Glob(filepath.Join(dir, "charts", "*", "Chart.yaml"))
					if err2 != nil {
						return "", errors.Wrapf(err, "failed to detect if there was a chart file in dir %s", dir)
					}
					if len(glob) == 0 {
						glob, err2 = filepath.Glob(filepath.Join(dir, "*", "Chart.yaml"))
						if err2 != nil {
							return "", errors.Wrapf(err, "failed to detect if there was a chart file in dir %s", dir)
						}
					}
					if len(glob) > 0 {
						hasHelm = true
					}

					if hasDocker {
						if hasHelm {
							lpack = filepath.Join(packsDir, "docker-helm")
							err = nil
						} else {
							lpack = filepath.Join(packsDir, "docker")
							err = nil
						}
					} else if hasHelm {
						lpack = filepath.Join(packsDir, "helm")
						err = nil
					}
				}
				if lpack == "" {
					exists, err2 := files.FileExists(filepath.Join(dir, jenkinsfile.Name))
					if exists && err2 == nil {
						lpack = filepath.Join(packsDir, "custom-jenkins")
						err = nil
					}
				}
				if err != nil {
					return "", err
				}
			}
		}
	}

	pack := filepath.Base(lpack)
	pack, err = o.PickCatalogFolderName(packsDir, pack)
	if err != nil {
		return "", err
	}
	lpack = filepath.Join(packsDir, pack)

	log.Logger().Infof("selected catalog folder: %s", termcolor.ColorInfo(pack))
	i.CustomDraftPack = pack

	if i.DisableAddFiles {
		return pack, nil
	}

	chartsDir := filepath.Join(dir, "charts")
	jenkinsxYamlExists, err := files.FileExists(jenkinsxYaml)
	if err != nil {
		return pack, err
	}

	err = o.copyBuildPack(dir, lpack)
	if err != nil {
		log.Logger().Warnf("Failed to apply the build pack in %s due to %s", dir, err)
	}

	// lets delete empty charts dir if a draft pack created one
	exists, err := files.DirExists(chartsDir)
	if err == nil && exists {
		fileList, err := os.ReadDir(chartsDir)
		if err != nil {
			return pack, errors.Wrapf(err, "failed to read charts dir %s", chartsDir)
		}
		if len(fileList) == 0 {
			err = os.Remove(chartsDir)
			if err != nil {
				return pack, errors.Wrapf(err, "failed to remove empty charts dir %s", chartsDir)
			}
		}
	}

	if !jenkinsxYamlExists && i.CreateJenkinsxYamlIfMissing {
		// lets check if we have a lighthouse trigger
		g := filepath.Join(dir, ".lighthouse", "*", "triggers.yaml")
		matches, err := filepath.Glob(g)
		if err != nil {
			return pack, errors.Wrapf(err, "failed to evaluate glob %s", g)
		}
		if len(matches) == 0 {
			pipelineConfig, err := config.LoadProjectConfigFile(jenkinsxYaml)
			if err != nil {
				return pack, err
			}

			// only update the build pack if its not currently set to none so that build packs can
			// use a custom pipeline plugin mechanism
			if pipelineConfig.BuildPack != pack && pipelineConfig.BuildPack != "none" {
				pipelineConfig.BuildPack = pack
				err = pipelineConfig.SaveConfig(jenkinsxYaml)
				if err != nil {
					return pack, err
				}
			}
		}
	}

	lighthouseDir := filepath.Join(packsDir, pack, ".lighthouse")
	exists, err = files.DirExists(lighthouseDir)
	if err != nil {
		return pack, errors.Wrapf(err, "failed to detect lighthouse dir %s", lighthouseDir)
	}
	if exists {
		err = o.createMissingLighthouseKptFiles(lighthouseDir, pack)
		if err != nil {
			return pack, errors.Wrapf(err, "failed to add missing Kptfiles for pipeline catalog")
		}
	}
	return pack, nil
}

// DiscoverBuildPack discovers the build pack given the build pack configuration
func (o *ImportOptions) DiscoverBuildPack(dir string, projectConfig *config.ProjectConfig, packConfig string) (string, error) {
	if packConfig != "" {
		return packConfig, nil
	}
	args := &InvokeDraftPack{
		Dir:             dir,
		CustomDraftPack: packConfig,
		ProjectConfig:   projectConfig,
		DisableAddFiles: true,
	}
	pack, err := o.InvokeDraftPack(args)
	if err != nil {
		return pack, errors.Wrapf(err, "failed to discover task pack in dir %s", dir)
	}
	return pack, nil
}

// Refactor: taken from jx so we can also bring in the draft pack and not fail when copying buildpacks without a charts dir
// CopyBuildPack copies the build pack from the source dir to the destination dir
func (o *ImportOptions) copyBuildPack(dest, src string) error {
	// first do some validation that we are copying from a valid pack directory
	p, err := FromDir(src)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", src, err)
	}

	chartsDir := filepath.Join(dest, "charts")
	newDir := filepath.Join(chartsDir, o.AppName)
	exists, err := files.DirExists(newDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if chart directory exists %s", newDir)
	}

	if exists {
		// If there already is a chart for this application let's skip copying it from the pack
		p.Charts = nil
	} else {
		// if we already have a Charts dir lets move it instead
		chartFile := filepath.Join(chartsDir, "Chart.yaml")
		exists, err := files.FileExists(chartFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if chart file exists %s", chartFile)
		}

		if exists {
			// If there already is a chart for this application let's skip copying it from the pack
			p.Charts = nil

			// let's move the charts folder to charts/$name so its a real chart layout

			fs, err := os.ReadDir(chartsDir)
			if err != nil {
				return errors.Wrapf(err, "failed to read dir %s", chartsDir)
			}
			err = os.MkdirAll(newDir, files.DefaultDirWritePermissions)
			if err != nil {
				return errors.Wrapf(err, "failed to create dir %s", newDir)
			}

			for _, f := range fs {
				name := f.Name()
				oldPath := filepath.Join(chartsDir, name)
				newPath := filepath.Join(newDir, name)
				err = os.Rename(oldPath, newPath)
				if err != nil {
					return errors.Wrapf(err, "failed to move file %s to %s", oldPath, newPath)
				}
			}
		}
	}

	// let's remove any files we think should be zapped
	for _, file := range []string{jenkinsfile.PipelineConfigFileName, jenkinsfile.PipelineTemplateFileName} {
		delete(p.Files, file)
	}

	if o.PackFilter != nil {
		o.PackFilter(p)
	}

	_, packName := filepath.Split(src)
	return p.SaveDir(dest, packName)
}

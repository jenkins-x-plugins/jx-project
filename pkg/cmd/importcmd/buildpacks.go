package importcmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-project/pkg/gitresolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-project/pkg/config"
	jxdraft "github.com/jenkins-x/jx-project/pkg/draft"
	"github.com/jenkins-x/jx-project/pkg/jenkinsfile"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// InvokeDraftPack used to pass arguments into the draft pack invocation
type InvokeDraftPack struct {
	Dir                         string
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
	bp, settings, err := o.PickBuildPackLibrary(i)
	if err != nil {
		return "", settings, err
	}

	dir, err := gitresolver.InitBuildPack(o.Git(), bp.Spec.GitURL, bp.Spec.GitRef)
	return dir, settings, err
}

// PickBuildPackLibrary lets you pick a build pack
func (o *ImportOptions) PickBuildPackLibrary(i *InvokeDraftPack) (*v1.BuildPack, *v1.TeamSettings, error) {
	jxClient := o.JXClient
	ns := o.Namespace
	settings, err := jxenv.GetDevEnvTeamSettings(jxClient, ns)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load the team settings")
	}

	list, err := jxClient.JenkinsV1().BuildPacks(ns).List(metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, settings, err
	}
	if len(list.Items) == 0 {
		list.Items = createDefaultBuildBacks()
	}

	buildPackURL := settings.BuildPackURL
	if i != nil && i.ProjectConfig != nil && i.ProjectConfig.BuildPackGitURL != "" {
		buildPackURL = i.ProjectConfig.BuildPackGitURL
	}
	ref := settings.BuildPackRef

	defaultName := ""
	found := false
	m := map[string]*v1.BuildPack{}
	names := []string{}
	for _, r := range list.Items {
		copy := r
		n := copy.Spec.Label
		if copy.Spec.GitURL == buildPackURL {
			defaultName = n
			found = true
			if ref != "" {
				copy.Spec.GitRef = ref
			}
		}
		m[n] = &copy
		if n != "" {
			names = append(names, n)
		}
	}
	if buildPackURL == "" {
		buildPackURL = "https://github.com/jenkins-x/jxr-packs-kubernetes"
	}
	if !found {
		defaultName = "Team Build Pack"
		bp := &v1.BuildPack{}
		bp.Name = "team-build-pack"
		bp.Spec.GitURL = buildPackURL
		bp.Spec.GitRef = ref
		bp.Spec.Label = defaultName
		names = append(names, bp.Spec.Label)
		m[defaultName] = bp
	}

	sort.Strings(names)

	name := defaultName
	if !o.BatchMode {
		name, err = o.Input.PickNameWithDefault(names, "Pick the build pack library you would like to use", defaultName,
			"the build pack library contains the default pipelines and associated files")
		if err != nil {
			return nil, settings, errors.Wrap(err, "failed to pick the build pack name")
		}
	}
	return m[name], settings, err
}

// createDefaultBuildBacks creates the default build packs if there are no BuildPack CRDs registered in a cluster
func createDefaultBuildBacks() []v1.BuildPack {
	return []v1.BuildPack{
		/* TODO
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kubernetes-workloads",
			},
			Spec: v1.BuildPackSpec{
				Label:  "Kubernetes Workloads: Automated CI+CD with GitOps Promotion",
				GitURL: v1.KubernetesWorkloadBuildPackURL,
				GitRef: v1.KubernetesWorkloadBuildPackRef,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "classic-workloads",
			},
			Spec: v1.BuildPackSpec{
				Label:  "Library Workloads: CI+Release but no CD",
				GitURL: v1.ClassicWorkloadBuildPackURL,
				GitRef: v1.ClassicWorkloadBuildPackRef,
			},
		},
		*/
	}
}

// InvokeDraftPack invokes a draft pack copying in a Jenkinsfile if required
func (o *ImportOptions) InvokeDraftPack(i *InvokeDraftPack) (string, error) {
	packsDir, _, err := o.InitBuildPacks(i)
	if err != nil {
		return "", err
	}

	// lets assume Jenkins X import mode
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
	envChart := filepath.Join(dir, "env/Chart.yaml")
	lpack := ""
	if len(customDraftPack) == 0 {
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

	if len(lpack) == 0 {
		if exists, err := files.FileExists(pomName); err == nil && exists {
			pack, err := PomFlavour(pomName)
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
					// lets detect docker and/or helm

					// TODO one day when our pipelines can include steps conditional on the presence of a file glob
					// we can just use a single docker/helm package that does docker and/or helm
					// but for now we've 3 separate packs for docker, docker-helm and helm
					hasDocker := false
					hasHelm := false

					if exists, err2 := files.FileExists(filepath.Join(dir, "Dockerfile")); err2 == nil && exists {
						hasDocker = true
					}

					// lets check for a helm pack
					files, err2 := filepath.Glob(filepath.Join(dir, "charts/*/Chart.yaml"))
					if err2 != nil {
						return "", errors.Wrapf(err, "failed to detect if there was a chart file in dir %s", dir)
					}
					if len(files) == 0 {
						files, err2 = filepath.Glob(filepath.Join(dir, "*/Chart.yaml"))
						if err2 != nil {
							return "", errors.Wrapf(err, "failed to detect if there was a chart file in dir %s", dir)
						}
					}
					if len(files) > 0 {
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
	pack, err = o.PickBuildPackName(i, packsDir, pack)
	if err != nil {
		return "", err
	}
	lpack = filepath.Join(packsDir, pack)

	log.Logger().Infof("selected build pack: %s", termcolor.ColorInfo(pack))
	i.CustomDraftPack = pack

	if i.DisableAddFiles {
		return pack, nil
	}

	chartsDir := filepath.Join(dir, "charts")
	jenkinsxYamlExists, err := files.FileExists(jenkinsxYaml)
	if err != nil {
		return pack, err
	}

	err = copyBuildPack(dir, lpack)
	if err != nil {
		log.Logger().Warnf("Failed to apply the build pack in %s due to %s", dir, err)
	}

	// lets delete empty charts dir if a draft pack created one
	exists, err := files.DirExists(chartsDir)
	if err == nil && exists {
		files, err := ioutil.ReadDir(chartsDir)
		if err != nil {
			return pack, errors.Wrapf(err, "failed to read charts dir %s", chartsDir)
		}
		if len(files) == 0 {
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
func copyBuildPack(dest, src string) error {
	// first do some validation that we are copying from a valid pack directory
	p, err := FromDir(src)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", src, err)
	}

	// lets remove any files we think should be zapped
	for _, file := range []string{jenkinsfile.PipelineConfigFileName, jenkinsfile.PipelineTemplateFileName} {
		delete(p.Files, file)
	}
	_, packName := filepath.Split(src)
	return p.SaveDir(dest, packName)
}

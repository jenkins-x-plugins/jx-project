package maven

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexflint/go-filemutex"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/downloads"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

// InstallMavenIfRequired installs maven if not available
func InstallMavenIfRequired(runner cmdrunner.CommandRunner) error {
	homeDir, err := homedir.ConfigDir(os.Getenv("JX3_HOME"), ".jx3")
	if err != nil {
		return err
	}
	m, err := filemutex.New(homeDir + "/jx.lock")
	if err != nil {
		panic(err)
	}
	err = m.Lock()
	if err != nil {
		return err
	}

	cmd := &cmdrunner.Command{
		Name: "mvn",
		Args: []string{"-v"},
	}
	_, err = runner(cmd)
	if err == nil {
		err = m.Unlock()
		if err != nil {
			return err
		}
		return nil
	}
	// let's assume maven is not installed so lets download it
	clientURL := fmt.Sprintf("https://repo1.maven.org/maven2/org/apache/maven/apache-maven/%s/apache-maven-%s-bin.zip", MavenVersion, MavenVersion)

	log.Logger().Infof("Apache Maven is not installed so lets download: %s", termcolor.ColorInfo(clientURL))

	mvnDir := filepath.Join(homeDir, "maven")
	mvnTmpDir := filepath.Join(homeDir, "maven-tmp")
	zipFile := filepath.Join(homeDir, "mvn.zip")

	err = os.MkdirAll(mvnDir, files.DefaultDirWritePermissions)
	if err != nil {
		err = m.Unlock()
		if err != nil {
			return err
		}
		return err
	}

	log.Logger().Info("\ndownloadFile")
	err = downloads.DownloadFile(clientURL, zipFile, true)
	if err != nil {
		err = m.Unlock()
		if err != nil {
			return err
		}
		return err
	}

	log.Logger().Info("\nfiles.Unzip")
	err = files.Unzip(zipFile, mvnTmpDir)
	if err != nil {
		err = m.Unlock()
		if err != nil {
			return err
		}
		return err
	}

	// let's find a directory inside the unzipped folder
	log.Logger().Info("\nReadDir")
	files, err := os.ReadDir(mvnTmpDir)
	if err != nil {
		err = m.Unlock()
		if err != nil {
			return err
		}
		return err
	}
	for _, f := range files {
		name := f.Name()
		if f.IsDir() && strings.HasPrefix(name, "apache-maven") {
			err = os.RemoveAll(mvnDir)
			if err != nil {
				return err
			}

			err = os.Rename(filepath.Join(mvnTmpDir, name), mvnDir)
			if err != nil {
				err = m.Unlock()
				if err != nil {
					return err
				}
				return err
			}
			log.Logger().Infof("Apache Maven is installed at: %s", termcolor.ColorInfo(mvnDir))
			err = m.Unlock()
			if err != nil {
				return err
			}
			err = os.Remove(zipFile)
			if err != nil {
				err = m.Unlock()
				if err != nil {
					return err
				}
				return err
			}
			err = os.RemoveAll(mvnTmpDir)
			if err != nil {
				err = m.Unlock()
				if err != nil {
					return err
				}
				return err
			}
			err = m.Unlock()
			if err != nil {
				return err
			}
			return nil
		}
	}
	err = m.Unlock()
	if err != nil {
		return err
	}
	return fmt.Errorf("could not find an apache-maven folder inside the unzipped maven distro at %s", mvnTmpDir)
}

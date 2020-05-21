package importcmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/draft/pkg/osutil"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	kchart "k8s.io/helm/pkg/proto/hapi/chart"
)

/**
CREDIT https://github.com/Azure/draft/blob/9705e36dc23c27c9ef54dc2469dd86ac6093f0f4/pkg/draft/pack/pack.go

This code was originally written in Draft but because Jenkins X build packs doesn't always contain a charts dir we
want to "continue" when looping around files to copy rather than return an error
*/

const (
	ChartsDir = "charts"
)

// Pack defines a Draft Starter Pack.
type Pack struct {
	// Chart is the Helm chart to be installed with the Pack.
	Charts []*kchart.Chart
	// Files are the files inside the Pack that will be installed.
	Files map[string]io.ReadCloser
}

// SaveDir saves a pack as files in a directory.
func (p *Pack) SaveDir(dest string) error {
	// Create the chart directory
	chartPath := filepath.Join(dest, ChartsDir)
	_, err := os.Stat(chartPath)
	if err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(chartPath, 0755); err != nil {
			return fmt.Errorf("Could not create %s: %s", chartPath, err)
		}
	}
	for _, chart := range p.Charts {
		if err := chartutil.SaveDir(chart, chartPath); err != nil {
			return err
		}
	}

	// save the rest of the files
	for relPath, f := range p.Files {
		path := filepath.Join(dest, relPath)
		exists, err := osutil.Exists(path)
		if err != nil {
			return errors.Wrapf(err, "failed to check if path exists %s", path)
		}
		if !exists {
			// lets make sure the parent dir exists
			parent := filepath.Dir(path)
			err = os.MkdirAll(parent, util.DefaultWritePermissions)
			if err != nil {
				return errors.Wrapf(err, "failed to make directory %s", parent)
			}
			newfile, err := os.Create(path)
			if err != nil {
				return errors.Wrapf(err, "failed to create file %s", path)
			}
			defer newfile.Close()
			defer f.Close()
			_, err = io.Copy(newfile, f)
			if err != nil {
				return errors.Wrapf(err, "failed to copy file %s", newfile.Name())
			}
		}
	}

	return nil
}

// CREDIT https://github.com/Azure/draft/blob/9705e36dc23c27c9ef54dc2469dd86ac6093f0f4/pkg/draft/pack/pack.go
// FromDir takes a string name, tries to resolve it to a file or directory, and then loads it.
//
// This is the preferred way to load a pack. It will discover the pack encoding
// and hand off to the appropriate pack reader.
func FromDir(dir string) (*Pack, error) {
	pack := new(Pack)
	pack.Files = make(map[string]io.ReadCloser)

	topdir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	err = loadDirectory(pack, topdir, "")
	return pack, err
}

func loadDirectory(pack *Pack, dir string, relPath string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading %s: %s", dir, err)
	}
	for _, fInfo := range files {
		name := fInfo.Name()
		if fInfo.IsDir() {
			// assume root folders not starting with dot are chart folders
			// could replace this logic with checking for charts / preview strings instead?
			if relPath == "" && !strings.HasPrefix(name, ".") {
				localChart, err := chartutil.LoadDir(filepath.Join(dir, name))
				if err != nil {
					continue
				}
				pack.Charts = append(pack.Charts, localChart)
			} else {
				// allow other directories to copy across
				err = loadDirectory(pack, filepath.Join(dir, name), filepath.Join(relPath, name))
				if err != nil {
					return err
				}
			}
		} else {
			var f, err = os.Open(filepath.Join(dir, name))
			if err != nil {
				return err
			}
			path := name
			if relPath != "" {
				path = filepath.Join(relPath, name)
			}
			pack.Files[path] = f
		}
	}
	return nil
}

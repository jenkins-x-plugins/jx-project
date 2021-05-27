package importcmd

import (
	"fmt"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/draft/pkg/osutil"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/pkg/errors"
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
	Charts []*chart.Chart
	// Files are the files inside the Pack that will be installed.
	Files map[string]io.ReadCloser
}

// SaveDir saves a pack as files in a directory.
func (p *Pack) SaveDir(dest string, packName string) error {
	// Create the chart directory
	chartPath := filepath.Join(dest, ChartsDir)
	_, err := os.Stat(chartPath)
	if err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(chartPath, 0755); err != nil {
			return fmt.Errorf("Could not create %s: %s", chartPath, err)
		}
	}
	for _, chart := range p.Charts {
		// lets make any new directories we need
		chartName := packName
		if chart.Metadata.Name == "preview" {
			chartName = chart.Metadata.Name
		}
		for _, f := range chart.Files {
			path := f.Name
			if path != "" {
				fullPath := filepath.Join(chartPath, chartName, path)
				dir := filepath.Dir(fullPath)

				// lets ensure the dir exists
				err = os.MkdirAll(dir, files.DefaultDirWritePermissions)
				if err != nil {
					return errors.Wrapf(err, "failed to create dir %s", dir)
				}
			}
		}

		if err := SaveDir(chart, chartPath, chartName); err != nil {
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
			err = os.MkdirAll(parent, files.DefaultDirWritePermissions)
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

// SaveDir saves a chart as files in a directory.
func SaveDir(c *chart.Chart, dest string, packName string) error {
	// Create the chart directory
	outdir := filepath.Join(dest, packName)
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return err
	}

	// Save the chart file.
	if err := chartutil.SaveChartfile(filepath.Join(outdir, chartutil.ChartfileName), c.Metadata); err != nil {
		return err
	}

	// Save values.yaml
	if c.Values != nil && len(c.Values) > 0 {
		vf := filepath.Join(outdir, chartutil.ValuesfileName)
		values := chartutil.Values(c.Values)
		data, err := values.YAML()
		if err != nil {
			return errors.Wrapf(err, "failed to marshal values YAML")
		}
		if err = ioutil.WriteFile(vf, []byte(data), 0755); err != nil {
			return errors.Wrapf(err, "failed to save yaml file %s", vf)
		}
	}

	for _, d := range []string{chartutil.TemplatesDir, ChartsDir} {
		if err := os.MkdirAll(filepath.Join(outdir, d), 0755); err != nil {
			return err
		}
	}

	// Save templates
	for _, f := range c.Templates {
		n := filepath.Join(outdir, f.Name)
		if err := ioutil.WriteFile(n, f.Data, 0755); err != nil {
			return err
		}
	}

	// Save files
	for _, f := range c.Files {
		n := filepath.Join(outdir, f.Name)
		if err := ioutil.WriteFile(n, f.Data, 0755); err != nil {
			return err
		}
	}

	// Save dependencies
	base := filepath.Join(outdir, ChartsDir)
	dependencies := c.Dependencies()
	for _, dep := range dependencies {
		// Here, we write each dependency as a tar file.
		if _, err := chartutil.Save(dep, base); err != nil {
			return err
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
	fileSlice, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading %s: %s", dir, err)
	}
	for _, fInfo := range fileSlice {
		name := fInfo.Name()
		chartPath := filepath.Join(dir, name)
		if fInfo.IsDir() {
			// assume root folders not starting with dot are chart folders
			// could replace this logic with checking for charts / preview strings instead?
			if relPath == "" && name != "preview" && !(strings.HasPrefix(name, ".")) {
				chartLoader, err := loader.Loader(chartPath)
				if err != nil {
					return errors.Wrapf(err, "failed to create chart loader for chart %s", chartPath)
				}

				localChart, err := chartLoader.Load()
				if err != nil {
					continue
				}
				pack.Charts = append(pack.Charts, localChart)

				// lets see if there's a nested resources folder
				resourceDir := filepath.Join(dir, name, "resources")
				exists, err := files.DirExists(resourceDir)
				if err != nil {
					return errors.Wrapf(err, "checking if resources dir exists %s", resourceDir)
				}
				if exists {
					_, packName := filepath.Split(dir)
					err = loadDirectory(pack, resourceDir, filepath.Join(relPath, name, packName, "resources"))
					if err != nil {
						return err
					}
				}
			} else {
				// allow other directories to copy across
				err = loadDirectory(pack, chartPath, filepath.Join(relPath, name))
				if err != nil {
					return err
				}
			}
		} else {
			var f, err = os.Open(chartPath)
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

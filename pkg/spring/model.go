package spring

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x-plugins/jx-project/pkg/cache"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/root/version"
)

const (
	OptionGroupID        = "group"
	OptionArtifactID     = "artifact"
	OptionLanguage       = "language"
	OptionJavaVersion    = "java-version"
	OptionBootVersion    = "boot-version"
	OptionPackaging      = "packaging"
	OptionDependency     = "dep"
	OptionDependencyKind = "kind"
	OptionType           = "type"

	startSpringURL = "https://start.spring.io"
)

var DefaultDependencyKinds = []string{"Core", "Web", "Template Engines", "SQL", "I/O", "Ops", "Spring Cloud GCP", "Azure", "Cloud Contract", "Cloud AWS", "Cloud Messaging", "Cloud Tracing"}

type Value struct {
	Type    string
	Default string
}

type Option struct {
	ID           string
	Name         string
	Description  string
	VersionRange string
}

type Options struct {
	Type    string
	Default string
	Values  []Option
}

type TreeGroup struct {
	Name   string
	Values []Option
}

type TreeSelect struct {
	Type   string
	Values []TreeGroup
}

type BootModel struct {
	Packaging    Options
	Language     Options
	JavaVersion  Options
	BootVersion  Options
	Type         Options
	GroupID      Value
	ArtifactID   Value
	Version      Value
	Name         Value
	Description  Value
	PackageName  Value
	Dependencies TreeSelect
}

type BootForm struct {
	Packaging       string
	Language        string
	JavaVersion     string
	BootVersion     string
	GroupID         string
	ArtifactID      string
	Version         string
	Name            string
	PackageName     string
	Dependencies    []string
	DependencyKinds []string
	Type            string
}

type errorResponse struct {
	Timestamp string `json:"timestamp,omitempty"`
	Status    int    `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
	Path      string `json:"path,omitempty"`
}

func LoadSpringBoot(cacheDir string) (*BootModel, error) {
	loader := func() ([]byte, error) {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodGet, startSpringURL, http.NoBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		addClientHeader(req)

		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		return io.ReadAll(res.Body)
	}

	cacheFileName := ""
	if cacheDir != "" {
		cacheFileName = filepath.Join(cacheDir, "start_spring_io.json")
	}
	body, err := cache.LoadCacheData(cacheFileName, loader)
	if err != nil {
		return nil, err
	}

	model := BootModel{}
	err = json.Unmarshal(body, &model)
	if err != nil {
		return nil, err
	}
	// default the build tool
	if model.Type.Default == "" {
		model.Type.Default = "maven"
	}
	if len(model.Type.Values) == 0 {
		model.Type.Values = []Option{
			{
				ID:          "gradle",
				Name:        "Gradle",
				Description: "Build with the gradle build tool",
			},
			{
				ID:          "maven",
				Name:        "Maven",
				Description: "Build with the maven build tool",
			},
		}
	}
	return &model, nil
}

func (model *BootModel) CreateSurvey(data *BootForm, advanced, batchMode bool) error {
	err := model.ValidateInput(OptionLanguage, &model.Language, data.Language)
	if err != nil {
		return err
	}
	err = model.ValidateInput(OptionBootVersion, &model.BootVersion, data.BootVersion)
	if err != nil {
		return err
	}
	err = model.ValidateInput(OptionJavaVersion, &model.JavaVersion, data.JavaVersion)
	if err != nil {
		return err
	}
	err = model.ValidateInput(OptionPackaging, &model.Packaging, data.Packaging)
	if err != nil {
		return err
	}
	err = model.ValidateTreeInput(OptionDependency, &model.Dependencies, data.Dependencies)
	if err != nil {
		return err
	}

	qs := []*survey.Question{}
	if batchMode {
		return nil
	}
	if data.Language == "" {
		qs = append(qs, CreateValueSelect("Language", "language", &model.Language, data))
	}
	if data.BootVersion == "" && advanced {
		qs = append(qs, CreateValueSelect("Spring Boot version", "bootVersion", &model.BootVersion, data))
	}
	if data.JavaVersion == "" && advanced {
		qs = append(qs, CreateValueSelect("Java version", "javaVersion", &model.JavaVersion, data))
	}
	if data.Packaging == "" && advanced {
		qs = append(qs, CreateValueSelect("Packaging", "packaging", &model.Packaging, data))
	}
	if data.Type == "" && advanced {
		qs = append(qs, CreateValueSelect("Build Tool", "type", &model.Type, data))
	}
	if data.GroupID == "" {
		qs = append(qs, CreateValueInput("Group", "groupId", &model.GroupID, data))
	}
	if data.ArtifactID == "" {
		qs = append(qs, CreateValueInput("Artifact", "artifactId", &model.ArtifactID, data))
	}
	if emptyArray(data.Dependencies) {
		qs = append(qs, CreateTreeSelect("Dependencies", "dependencies", &model.Dependencies, data))
	}
	return survey.Ask(qs, data)
}

func (o *Options) StringArray() []string {
	values := []string{}
	for _, o := range o.Values {
		id := o.ID
		if id != "" {
			values = append(values, id)
		}
	}
	sort.Strings(values)
	return values
}

func (options *TreeSelect) StringArray() []string {
	values := []string{}
	for _, g := range options.Values {
		for _, o := range g.Values {
			id := o.ID
			if id != "" {
				values = append(values, id)
			}
		}
	}
	sort.Strings(values)
	return values
}

func (model *BootModel) ValidateInput(name string, o *Options, value string) error {
	if value != "" && o != nil {
		for _, v := range o.Values {
			if v.ID == value {
				return nil
			}
		}
		return options.InvalidOption(name, value, o.StringArray())
	}
	return nil
}

func (model *BootModel) ValidateTreeInput(name string, o *TreeSelect, values []string) error {
	if len(values) > 0 && o != nil {
		for _, value := range values {
			if value != "" {
				valid := false
				for _, g := range o.Values {
					for _, o := range g.Values {
						if o.ID == value {
							valid = true
							break
						}
					}
				}
				if !valid {
					return options.InvalidOption(name, value, o.StringArray())
				}
			}
		}
	}
	return nil
}

func CreateValueSelect(message, name string, options *Options, data *BootForm) *survey.Question {
	values := options.StringArray()
	return &survey.Question{
		Name: name,
		Prompt: &survey.Select{
			Message: message + ":",
			Options: values,
			Default: options.Default,
		},
		Validate: survey.Required,
	}
}

func CreateValueInput(message, name string, value *Value, data *BootForm) *survey.Question {
	return &survey.Question{
		Name: name,
		Prompt: &survey.Input{
			Message: message + ":",
			Default: value.Default,
		},
		Validate: survey.Required,
	}
}

func CreateTreeSelect(message, name string, tree *TreeSelect, data *BootForm) *survey.Question {
	dependencyKinds := []string{}
	if data.DependencyKinds != nil {
		dependencyKinds = data.DependencyKinds
	}
	if len(dependencyKinds) == 0 {
		dependencyKinds = DefaultDependencyKinds
	}
	values := []string{}
	for _, t := range tree.Values {
		tvName := t.Name
		if stringhelpers.StringArrayIndex(dependencyKinds, tvName) >= 0 {
			for _, v := range t.Values {
				id := v.ID
				if id != "" {
					values = append(values, id)
				}
			}
		}
	}
	sort.Strings(values)
	return &survey.Question{
		Name: name,
		Prompt: &survey.MultiSelect{
			Message: message + ":",
			Options: values,
		},
		Validate: survey.Required,
	}
}

func (data *BootForm) CreateProject(workDir string) (string, error) {
	dirName := data.ArtifactID
	if dirName == "" {
		dirName = "project"
	}
	answer := filepath.Join(workDir, dirName)

	client := http.Client{}

	form := url.Values{}
	data.AddFormValues(&form)

	parameters := form.Encode()
	if parameters != "" {
		parameters = "?" + parameters
	}
	u := "http://start.spring.io/starter.zip" + parameters
	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))
	if err != nil {
		return answer, err
	}
	addClientHeader(req)
	res, err := client.Do(req)
	if err != nil {
		return answer, err
	}
	defer res.Body.Close()
	if res.StatusCode == 400 {
		errorBody, err := io.ReadAll(res.Body)
		if err != nil {
			return answer, err
		}

		errorResp := errorResponse{}
		err = json.Unmarshal(errorBody, &errorResp)
		if err != nil {
			return answer, err
		}

		log.Logger().Infof("%s", termcolor.ColorError(errorResp.Message))
		return answer, errors.New("unable to create spring quickstart")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return answer, err
	}

	dir := filepath.Join(workDir, dirName)
	zipFile := dir + ".zip"
	err = os.WriteFile(zipFile, body, files.DefaultFileWritePermissions)
	if err != nil {
		return answer, fmt.Errorf("failed to download file %s due to %s", zipFile, err)
	}
	err = files.Unzip(zipFile, dir)
	if err != nil {
		return answer, fmt.Errorf("failed to unzip new project file %s due to %s", zipFile, err)
	}
	err = os.Remove(zipFile)
	if err != nil {
		return answer, err
	}
	return answer, nil
}

func (data *BootForm) AddFormValues(form *url.Values) {
	AddFormValue(form, "packaging", data.Packaging)
	AddFormValue(form, "language", data.Language)
	AddFormValue(form, "javaVersion", data.JavaVersion)
	AddFormValue(form, "bootVersion", data.BootVersion)
	AddFormValue(form, "groupId", data.GroupID)
	AddFormValue(form, "artifactId", data.ArtifactID)
	AddFormValue(form, "version", data.Version)
	AddFormValue(form, "name", data.Name)
	AddFormValue(form, "type", data.Type)
	AddFormValues(form, "dependencies", data.Dependencies)
}

func AddFormValues(form *url.Values, key string, values []string) {
	for _, v := range values {
		if v != "" {
			form.Add(key, v)
		}
	}
}

func AddFormValue(form *url.Values, key, v string) {
	if v != "" {
		form.Add(key, v)
	}
}

func emptyArray(values []string) bool {
	return len(values) == 0
}

func addClientHeader(req *http.Request) {
	userAgent := "jx/" + version.GetVersion()
	req.Header.Set("User-Agent", userAgent)
}

package maven

import (
	"strings"

	"fmt"
	"sort"

	"github.com/jenkins-x/jx-helpers/pkg/input"
	"github.com/jenkins-x/jx-helpers/pkg/options"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/pkg/errors"
)

const (
	MavenArchetypePluginVersion = "3.0.1"
)

type ArtifactVersions struct {
	GroupId     string
	ArtifactId  string
	Description string
	Versions    []string
}

type GroupArchectypes struct {
	GroupId   string
	Artifacts map[string]*ArtifactVersions
}

type ArchetypeModel struct {
	Groups map[string]*GroupArchectypes
}

type ArtifactData struct {
	GroupId     string
	ArtifactId  string
	Version     string
	Description string
}

type ArchetypeFilter struct {
	GroupIds         []string
	GroupIdFilter    string
	ArtifactIdFilter string
	Version          string
}

type ArchetypeForm struct {
	ArchetypeGroupId    string
	ArchetypeArtifactId string
	ArchetypeVersion    string

	GroupId    string
	ArtifactId string
	Package    string
	Version    string
}

func NewArchetypeModel() ArchetypeModel {
	return ArchetypeModel{
		Groups: map[string]*GroupArchectypes{},
	}
}

func (m *ArchetypeModel) GroupIDs(filter string) []string {
	answer := []string{}
	for group := range m.Groups {
		if filter == "" || strings.Index(group, filter) >= 0 {
			answer = append(answer, group)
		}
	}
	sort.Strings(answer)
	return answer
}

func (m *ArchetypeModel) ArtifactIDs(groupId string, filter string) []string {
	answer := []string{}
	artifact := m.Groups[groupId]
	if artifact != nil {
		for a := range artifact.Artifacts {
			if filter == "" || strings.Index(a, filter) >= 0 {
				answer = append(answer, a)
			}
		}
		sort.Strings(answer)
	}
	return answer
}

func (m *ArchetypeModel) Versions(groupId string, artifactId, filter string) []string {
	answer := []string{}
	artifact := m.Groups[groupId]
	if artifact != nil {
		av := artifact.Artifacts[artifactId]
		if av != nil {
			for _, v := range av.Versions {
				if filter == "" || strings.Index(v, filter) >= 0 {
					answer = append(answer, v)
				}
			}
			// TODO use a version sorter?
			sort.Sort(sort.Reverse(sort.StringSlice(answer)))
		}
	}
	return answer
}

func (m *ArchetypeModel) AddArtifact(a *ArtifactData) *ArtifactVersions {
	groupId := a.GroupId
	artifactId := a.ArtifactId
	version := a.Version
	description := a.Description
	if groupId == "" || artifactId == "" || version == "" {
		return nil
	}

	if m.Groups == nil {
		m.Groups = map[string]*GroupArchectypes{}
	}
	group := m.Groups[groupId]
	if group == nil {
		group = &GroupArchectypes{
			GroupId:   groupId,
			Artifacts: map[string]*ArtifactVersions{},
		}
		m.Groups[groupId] = group
	}
	artifact := group.Artifacts[artifactId]
	if artifact == nil {
		artifact = &ArtifactVersions{
			GroupId:    groupId,
			ArtifactId: artifactId,
			Versions:   []string{},
		}
		group.Artifacts[artifactId] = artifact
	}
	if artifact.Description == "" && description != "" {
		artifact.Description = description
	}
	if stringhelpers.StringArrayIndex(artifact.Versions, version) < 0 {
		artifact.Versions = append(artifact.Versions, version)
	}
	return artifact
}

func (model *ArchetypeModel) CreateSurvey(data *ArchetypeFilter, pickVersion bool, form *ArchetypeForm, i input.Interface) error {
	groupIds := data.GroupIds
	var err error
	if len(data.GroupIds) == 0 {
		filteredGroups := model.GroupIDs(data.GroupIdFilter)
		if len(filteredGroups) == 0 {
			return options.InvalidOption("group-filter", data.GroupIdFilter, model.GroupIDs(""))
		}

		// lets pick from all groups
		form.ArchetypeGroupId, err = i.PickNameWithDefault(filteredGroups, "Group ID:", form.ArchetypeGroupId, "please pick the maven Group ID")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Group ID")
		}
		artifactsWithoutFilter := model.ArtifactIDs(form.ArchetypeGroupId, "")
		if len(artifactsWithoutFilter) == 0 {
			return fmt.Errorf("Could not find any artifacts for group %s", form.ArchetypeGroupId)
		}
	} else {
		// TODO for now lets just support a single group ID being passed in
		form.ArchetypeGroupId = groupIds[0]

		artifactsWithoutFilter := model.ArtifactIDs(form.ArchetypeGroupId, "")
		if len(artifactsWithoutFilter) == 0 {
			return options.InvalidOption("group", form.ArchetypeGroupId, model.GroupIDs(""))
		}
	}
	if form.ArchetypeGroupId == "" {
		return fmt.Errorf("No archetype groupId selected")
	}

	artifactIds := model.ArtifactIDs(form.ArchetypeGroupId, data.ArtifactIdFilter)
	if len(artifactIds) == 0 {
		artifactsWithoutFilter := model.ArtifactIDs(form.ArchetypeGroupId, "")
		return options.InvalidOption("artifact", data.ArtifactIdFilter, artifactsWithoutFilter)
	}

	if len(artifactIds) == 1 {
		form.ArchetypeArtifactId = artifactIds[0]
	} else {
		form.ArchetypeArtifactId, err = i.PickNameWithDefault(artifactIds, "Artifact ID:", form.ArchetypeArtifactId, "please pick the maven Artifact ID")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Artifact ID")
		}
	}
	if form.ArchetypeArtifactId == "" {
		return fmt.Errorf("No archetype artifactId selected")
	}

	version := data.Version
	versions := model.Versions(form.ArchetypeGroupId, form.ArchetypeArtifactId, version)
	if len(versions) == 0 {
		return options.InvalidOption("version", version, model.Versions(form.ArchetypeGroupId, form.ArchetypeArtifactId, ""))
	}

	if len(versions) == 1 || !pickVersion {
		form.ArchetypeVersion = versions[0]
	} else {
		form.ArchetypeVersion, err = i.PickNameWithDefault(versions, "Version:", form.ArchetypeVersion, "please pick the maven version")
		if err != nil {
			return errors.Wrapf(err, "failed to pick version")
		}
	}
	if form.ArchetypeVersion == "" {
		return fmt.Errorf("No archetype version selected")
	}

	if form.GroupId == "" {
		form.GroupId, err = i.PickValue("Project Group ID:", "com.acme", true, "The maven Group ID used to default in the pom")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Project Group ID")
		}
	}
	if form.ArtifactId == "" {
		form.ArtifactId, err = i.PickValue("Project Artifact ID:", "", true, "The maven Artifact ID used to default in the pom")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Project Artifact ID")
		}
	}
	if form.Version == "" {
		form.Version, err = i.PickValue("Project Version:", "1.0.0-SNAPSHOT", true, "The maven Version used to default in the pom")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Project Version")
		}
	}
	return nil
}

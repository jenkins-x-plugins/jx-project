package maven

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/input"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/pkg/errors"
)

const (
	MavenArchetypePluginVersion = "3.0.1"
)

type ArtifactVersions struct {
	GroupID     string
	ArtifactID  string
	Description string
	Versions    []string
}

type GroupArchectypes struct {
	GroupID   string
	Artifacts map[string]*ArtifactVersions
}

type ArchetypeModel struct {
	Groups map[string]*GroupArchectypes
}

type ArtifactData struct {
	GroupID     string
	ArtifactID  string
	Version     string
	Description string
}

type ArchetypeFilter struct {
	GroupIDs         []string
	GroupIDFilter    string
	ArtifactIDFilter string
	Version          string
}

type ArchetypeForm struct {
	ArchetypeGroupID    string
	ArchetypeArtifactID string
	ArchetypeVersion    string

	GroupID    string
	ArtifactID string
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
		if filter == "" || strings.Contains(group, filter) {
			answer = append(answer, group)
		}
	}
	sort.Strings(answer)
	return answer
}

func (m *ArchetypeModel) ArtifactIDs(groupID, filter string) []string {
	answer := []string{}
	artifact := m.Groups[groupID]
	if artifact != nil {
		for a := range artifact.Artifacts {
			if filter == "" || strings.Contains(a, filter) {
				answer = append(answer, a)
			}
		}
		sort.Strings(answer)
	}
	return answer
}

func (m *ArchetypeModel) Versions(groupID, artifactID, filter string) []string {
	answer := []string{}
	artifact := m.Groups[groupID]
	if artifact != nil {
		av := artifact.Artifacts[artifactID]
		if av != nil {
			for _, v := range av.Versions {
				if filter == "" || strings.Contains(v, filter) {
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
	groupID := a.GroupID
	artifactID := a.ArtifactID
	version := a.Version
	description := a.Description
	if groupID == "" || artifactID == "" || version == "" {
		return nil
	}

	if m.Groups == nil {
		m.Groups = map[string]*GroupArchectypes{}
	}
	group := m.Groups[groupID]
	if group == nil {
		group = &GroupArchectypes{
			GroupID:   groupID,
			Artifacts: map[string]*ArtifactVersions{},
		}
		m.Groups[groupID] = group
	}
	artifact := group.Artifacts[artifactID]
	if artifact == nil {
		artifact = &ArtifactVersions{
			GroupID:    groupID,
			ArtifactID: artifactID,
			Versions:   []string{},
		}
		group.Artifacts[artifactID] = artifact
	}
	if artifact.Description == "" && description != "" {
		artifact.Description = description
	}
	if stringhelpers.StringArrayIndex(artifact.Versions, version) < 0 {
		artifact.Versions = append(artifact.Versions, version)
	}
	return artifact
}

func (m *ArchetypeModel) CreateSurvey(data *ArchetypeFilter, pickVersion bool, form *ArchetypeForm, i input.Interface) error {
	groupIDs := data.GroupIDs
	var err error
	if len(data.GroupIDs) == 0 {
		filteredGroups := m.GroupIDs(data.GroupIDFilter)
		if len(filteredGroups) == 0 {
			return options.InvalidOption("group-filter", data.GroupIDFilter, m.GroupIDs(""))
		}

		// let's pick from all groups
		form.ArchetypeGroupID, err = i.PickNameWithDefault(filteredGroups, "Group ID:", form.ArchetypeGroupID, "please pick the maven Group ID")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Group ID")
		}
		artifactsWithoutFilter := m.ArtifactIDs(form.ArchetypeGroupID, "")
		if len(artifactsWithoutFilter) == 0 {
			return fmt.Errorf("could not find any artifacts for group %s", form.ArchetypeGroupID)
		}
	} else {
		// TODO for now lets just support a single group ID being passed in
		form.ArchetypeGroupID = groupIDs[0]

		artifactsWithoutFilter := m.ArtifactIDs(form.ArchetypeGroupID, "")
		if len(artifactsWithoutFilter) == 0 {
			return options.InvalidOption("group", form.ArchetypeGroupID, m.GroupIDs(""))
		}
	}
	if form.ArchetypeGroupID == "" {
		return fmt.Errorf("no archetype groupId selected")
	}

	artifactIDs := m.ArtifactIDs(form.ArchetypeGroupID, data.ArtifactIDFilter)
	if len(artifactIDs) == 0 {
		artifactsWithoutFilter := m.ArtifactIDs(form.ArchetypeGroupID, "")
		return options.InvalidOption("artifact", data.ArtifactIDFilter, artifactsWithoutFilter)
	}

	if len(artifactIDs) == 1 {
		form.ArchetypeArtifactID = artifactIDs[0]
	} else {
		form.ArchetypeArtifactID, err = i.PickNameWithDefault(artifactIDs, "Artifact ID:", form.ArchetypeArtifactID, "please pick the maven Artifact ID")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Artifact ID")
		}
	}
	if form.ArchetypeArtifactID == "" {
		return fmt.Errorf("no archetype artifactId selected")
	}

	version := data.Version
	versions := m.Versions(form.ArchetypeGroupID, form.ArchetypeArtifactID, version)
	if len(versions) == 0 {
		return options.InvalidOption("version", version, m.Versions(form.ArchetypeGroupID, form.ArchetypeArtifactID, ""))
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
		return fmt.Errorf("no archetype version selected")
	}

	if form.GroupID == "" {
		form.GroupID, err = i.PickValue("Project Group ID:", "com.acme", true, "The maven Group ID used to default in the pom")
		if err != nil {
			return errors.Wrapf(err, "failed to pick Project Group ID")
		}
	}
	if form.ArtifactID == "" {
		form.ArtifactID, err = i.PickValue("Project Artifact ID:", "", true, "The maven Artifact ID used to default in the pom")
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

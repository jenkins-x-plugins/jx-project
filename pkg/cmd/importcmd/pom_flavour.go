package importcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
)

const (
	MAVEN      = "maven"
	APPSERVER  = "appserver"
	LIBERTY    = "liberty"
	DROPWIZARD = "dropwizard"
)

func PomFlavour(packsDir, pomPath string) (string, error) {
	b, err := os.ReadFile(pomPath)
	if err != nil {
		return "", nil
	}

	s := string(b)
	if strings.Contains(s, "<packaging>war</packaging>") &&
		strings.Contains(s, "org.eclipse.microprofile") {
		return LIBERTY, nil
	}
	if strings.Contains(s, "<groupId>io.dropwizard") {
		return DROPWIZARD, nil
	}
	if strings.Contains(s, "<groupId>org.apache.tomcat") {
		return APPSERVER, nil
	}
	// java.version is used by Spring Boot
	version, ok := getProp(s, "java.version")
	if !ok {
		// maven-compiler-plugin 3.6 and later versions supports and recommends maven.compiler.release
		version, ok = getProp(s, "maven.compiler.release")
	}
	if !ok {
		// older versions of maven-compiler-plugin uses maven.compiler.target
		version, ok = getProp(s, "maven.compiler.target")
	}
	if ok {
		pack := "maven-java" + version
		if exists, _ := files.DirExists(filepath.Join(packsDir, pack)); exists {
			return pack, nil
		}
	}

	return MAVEN, nil
}

func getProp(pom, prop string) (string, bool) {
	propPattern := regexp.MustCompile(fmt.Sprintf("<%s>(\\d+)</%s>", prop, prop))
	matches := propPattern.FindStringSubmatch(pom)
	if matches != nil {
		return matches[1], true
	}
	return "", false
}

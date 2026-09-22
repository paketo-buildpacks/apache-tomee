// Copyright 2018-2026 the original author or authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/libdependency/buildpack_config"
	"github.com/paketo-buildpacks/libdependency/retrieve"
	"github.com/paketo-buildpacks/libdependency/upstream"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
)

const (
	tomeeRepository = "https://downloads.apache.org/tomee"
	tomeePURLName   = "apache-tomee"
	tomeeLicense    = "https://www.apache.org/licenses/"

	supportGroupID  = "org.cloudfoundry"
	mavenRepository = "https://repo1.maven.org/maven2"
	supportLicense  = "https://github.com/cloudfoundry/java-buildpack-support/blob/main/LICENSE"
)

var tomeeVersionPattern = regexp.MustCompile(`tomee-([0-9]+\.[0-9]+\.[0-9]+[^"/]*)/`)

type tomeeDistribution struct {
	id   string
	dist string
	name string
}

var tomeeDistributions = []tomeeDistribution{
	{id: "tomee-microprofile", dist: "microprofile", name: "Apache Tomee - Microprofile"},
	{id: "tomee-webprofile", dist: "webprofile", name: "Apache Tomee - Webprofile"},
	{id: "tomee-plus", dist: "plus", name: "Apache Tomee - Plus"},
	{id: "tomee-plume", dist: "plume", name: "Apache Tomee - Plume"},
}

type generator struct {
	id               string
	getAllVersions   retrieve.GetAllVersionsFunc
	generateMetadata retrieve.GenerateMetadataFunc
}

type tomeeVersion struct {
	version *semver.Version
	raw     string
}

func (v tomeeVersion) Version() *semver.Version {
	return v.version
}

type mavenVersion struct {
	version *semver.Version
	raw     string
}

func (v mavenVersion) Version() *semver.Version {
	return v.version
}

type mavenMetadata struct {
	Versioning struct {
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
	} `xml:"versioning"`
}

func main() {
	buildpackTomlPath, output := retrieve.FetchArgs()

	config, err := buildpack_config.ParseBuildpackToml(buildpackTomlPath)
	if err != nil {
		panic(err)
	}

	var dependencies []versionology.Dependency
	for _, g := range generators() {
		newVersions, err := retrieve.GetNewVersionsForId(g.id, config, g.getAllVersions)
		if err != nil {
			panic(err)
		}

		dependencies = append(dependencies, retrieve.GenerateAllMetadata(newVersions, g.generateMetadata)...)
	}

	metadataJSON, err := json.Marshal(dependencies)
	if err != nil {
		panic(fmt.Errorf("unable to marshal metadata json\n%w", err))
	}

	if err := os.WriteFile(output, metadataJSON, os.ModePerm); err != nil {
		panic(fmt.Errorf("cannot write to %s: %w", output, err))
	}

	fmt.Printf("Wrote metadata to %s\n", output)
}

func generators() []generator {
	generators := make([]generator, 0, len(tomeeDistributions)+3)
	for _, distribution := range tomeeDistributions {
		generators = append(generators, tomeeGenerator(distribution))
	}

	return append(generators,
		mavenGenerator("tomcat-access-logging-support", "Apache Tomcat Access Logging Support"),
		mavenGenerator("tomcat-lifecycle-support", "Apache Tomcat Lifecycle Support"),
		mavenGenerator("tomcat-logging-support", "Apache Tomcat Logging Support"),
	)
}

func tomeeGenerator(distribution tomeeDistribution) generator {
	return generator{
		id:             distribution.id,
		getAllVersions: getAllTomeeVersions,
		generateMetadata: func(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
			return generateTomeeMetadata(distribution, versionFetcher)
		},
	}
}

func mavenGenerator(artifact, displayName string) generator {
	return generator{
		id: artifact,
		getAllVersions: func() (versionology.VersionFetcherArray, error) {
			return getAllMavenVersions(artifact)
		},
		generateMetadata: func(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
			return generateMavenMetadata(artifact, displayName, versionFetcher)
		},
	}
}

func getAllTomeeVersions() (versionology.VersionFetcherArray, error) {
	body, err := getBody(tomeeRepository + "/")
	if err != nil {
		return nil, err
	}

	var versions versionology.VersionFetcherArray
	seen := map[string]bool{}

	for _, match := range tomeeVersionPattern.FindAllStringSubmatch(body, -1) {
		versionString := match[1]
		if seen[versionString] {
			continue
		}
		seen[versionString] = true

		version, err := semver.NewVersion(versionString)
		if err != nil {
			fmt.Printf("Skipping %s: unable to parse version\n", versionString)
			continue
		}

		versions = append(versions, tomeeVersion{version: version, raw: versionString})
	}

	return versions, nil
}

func generateTomeeMetadata(distribution tomeeDistribution, versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version, ok := versionFetcher.(tomeeVersion)
	if !ok {
		return nil, fmt.Errorf("unexpected version type %T", versionFetcher)
	}

	versionString := version.raw

	uri := fmt.Sprintf("%s/tomee-%s/apache-tomee-%s-%s.tar.gz", tomeeRepository, versionString, versionString, distribution.dist)
	checksum, err := upstream.GetSHA256OfRemoteFile(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to checksum %s\n%w", uri, err)
	}

	source := fmt.Sprintf("%s/tomee-%s/tomee-project-%s-source-release.zip", tomeeRepository, versionString, versionString)
	sourceChecksum, err := upstream.GetSHA256OfRemoteFile(source)
	if err != nil {
		return nil, fmt.Errorf("unable to checksum %s\n%w", source, err)
	}

	dependency := cargo.ConfigMetadataDependency{
		Checksum: fmt.Sprintf("sha256:%s", checksum),
		CPE:      fmt.Sprintf("cpe:2.3:a:apache:%s:%s:*:*:*:*:*:*:*", distribution.id, versionString),
		ID:       distribution.id,
		Licenses: []interface{}{
			map[string]string{
				"type": "Apache-2.0",
				"uri":  tomeeLicense,
			},
		},
		Name:           distribution.name,
		PURL:           retrieve.GeneratePURL(tomeePURLName, versionString, checksum, uri),
		Source:         source,
		SourceChecksum: fmt.Sprintf("sha256:%s", sourceChecksum),
		Stacks:         []string{"*"},
		URI:            uri,
		Version:        versionString,
	}

	return versionology.NewDependencyArray(dependency, "")
}

func getAllMavenVersions(artifact string) (versionology.VersionFetcherArray, error) {
	metadataURL := fmt.Sprintf("%s/%s/%s/maven-metadata.xml", mavenRepository, strings.ReplaceAll(supportGroupID, ".", "/"), artifact)

	body, err := getBody(metadataURL)
	if err != nil {
		return nil, err
	}

	var metadata mavenMetadata
	if err := xml.Unmarshal([]byte(body), &metadata); err != nil {
		return nil, fmt.Errorf("unable to parse %s\n%w", metadataURL, err)
	}

	var versions versionology.VersionFetcherArray
	for _, raw := range metadata.Versioning.Versions.Version {
		version, err := semver.NewVersion(strings.TrimSuffix(raw, ".RELEASE"))
		if err != nil {
			fmt.Printf("Skipping %s: unable to parse version\n", raw)
			continue
		}

		versions = append(versions, mavenVersion{version: version, raw: raw})
	}

	return versions, nil
}

func generateMavenMetadata(artifact, displayName string, versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version, ok := versionFetcher.(mavenVersion)
	if !ok {
		return nil, fmt.Errorf("unexpected version type %T", versionFetcher)
	}

	versionString := version.version.String()

	base := fmt.Sprintf("%s/%s/%s/%s", mavenRepository, strings.ReplaceAll(supportGroupID, ".", "/"), artifact, version.raw)
	uri := fmt.Sprintf("%s/%s-%s.jar", base, artifact, version.raw)
	checksum, err := upstream.GetSHA256OfRemoteFile(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to checksum %s\n%w", uri, err)
	}

	source := fmt.Sprintf("%s/%s-%s-sources.jar", base, artifact, version.raw)
	sourceChecksum, err := upstream.GetSHA256OfRemoteFile(source)
	if err != nil {
		return nil, fmt.Errorf("unable to checksum %s\n%w", source, err)
	}

	dependency := cargo.ConfigMetadataDependency{
		Checksum: fmt.Sprintf("sha256:%s", checksum),
		CPE:      fmt.Sprintf("cpe:2.3:a:cloudfoundry:%s:%s:*:*:*:*:*:*:*", artifact, versionString),
		ID:       artifact,
		Licenses: []interface{}{
			map[string]string{
				"type": "Apache-2.0",
				"uri":  supportLicense,
			},
		},
		Name:           displayName,
		PURL:           retrieve.GeneratePURL(artifact, versionString, checksum, uri),
		Source:         source,
		SourceChecksum: fmt.Sprintf("sha256:%s", sourceChecksum),
		Stacks:         []string{"*"},
		URI:            uri,
		Version:        versionString,
	}

	return versionology.NewDependencyArray(dependency, "")
}

func getBody(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("unable to fetch %s\n%w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to fetch %s: status code %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read %s\n%w", url, err)
	}

	return string(body), nil
}

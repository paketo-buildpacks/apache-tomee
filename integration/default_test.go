package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testDefault(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose()
		docker = occam.NewDocker()
	})

	context("when the buildpack is run with pack build", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		const eventuallyTimeout = time.Second * 30

		it.Before(func() {
			var err error
			name, err = occam.RandomName()

			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			if container.ID != "" {
				Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			}
			if image.ID != "" {
				Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			}
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds with the defaults", func() {
			var (
				logs fmt.Stringer
				err  error
			)

			image, logs, err = pack.
				WithNoColor().
				WithVerbose().
				Build.
				WithPullPolicy("if-not-present").
				WithBuilder("paketobuildpacks/builder:buildpackless-base").
				WithBuildpacks("gcr.io/paketo-buildpacks/ca-certificates",
					"gcr.io/paketo-buildpacks/bellsoft-liberica",
					"gcr.io/paketo-buildpacks/syft",
					"gcr.io/paketo-buildpacks/maven",
					buildpack).
				WithEnv(map[string]string{
					"BP_JAVA_APP_SERVER":       "tomee",
					"BP_MAVEN_BUILT_ARTIFACT":  "test-jaxrs-tomee/target/*.war",
					"BP_MAVEN_BUILT_MODULE":    "test-jaxrs-tomee",
					"BP_MAVEN_BUILD_ARGUMENTS": "-Dmaven.test.skip=true package --no-transfer-progress",
				}).
				WithTrustBuilder().
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				WithReadOnly().
				WithMounts("type=bind,source=/tmp,target=/tmp").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container, eventuallyTimeout).Should(Serve(ContainSubstring("{\"application_status\":\"UP\"}")).OnPort(8080))
		})

		it("builds on tiny", func() {
			var (
				logs fmt.Stringer
				err  error
			)

			image, logs, err = pack.
				WithNoColor().
				WithVerbose().
				Build.
				WithPullPolicy("if-not-present").
				WithBuilder("paketobuildpacks/builder:buildpackless-tiny").
				WithBuildpacks("gcr.io/paketo-buildpacks/ca-certificates",
					"gcr.io/paketo-buildpacks/bellsoft-liberica",
					"gcr.io/paketo-buildpacks/syft",
					"gcr.io/paketo-buildpacks/maven",
					buildpack).
				WithEnv(map[string]string{
					"BP_JAVA_APP_SERVER":       "tomee",
					"BP_MAVEN_BUILT_ARTIFACT":  "test-jaxrs-tomee/target/*.war",
					"BP_MAVEN_BUILT_MODULE":    "test-jaxrs-tomee",
					"BP_MAVEN_BUILD_ARGUMENTS": "-Dmaven.test.skip=true package --no-transfer-progress",
				}).
				WithTrustBuilder().
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{
					"PORT":                             "8080",
					"BPL_TOMEE_ACCESS_LOGGING_ENABLED": "true",
				}).
				WithPublish("8080").
				WithPublishAll().
				WithReadOnly().
				WithMounts("type=bind,source=/tmp,target=/tmp").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container, eventuallyTimeout).Should(Serve(ContainSubstring("{\"application_status\":\"UP\"}")).OnPort(8080))
			logs, err = docker.Container.Logs.Execute(container.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(logs.String()).To(ContainSubstring("Server startup in"))
		})
	})
}

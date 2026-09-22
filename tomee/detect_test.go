/*
 * Copyright 2018-2022 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tomee_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/apache-tomee/tomee"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx    libcnb.DetectContext
		detect tomee.Detect
		path   string
	)

	it.Before(func() {
		var err error
		path, err = os.MkdirTemp("", "tomee")
		Expect(err).NotTo(HaveOccurred())

		ctx.Application.Path = path

		t.Setenv("BP_JAVA_APP_SERVER", "tomee")
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("fails with Main-Class", func() {
		Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(path, "META-INF", "MANIFEST.MF"), []byte(`Main-Class: test-main-class`), 0644)).To(Succeed())

		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{Pass: false}))
	})

	context("WEB-INF not found", func() {
		it("requires jvm-application-artifact", func() {
			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{
					{
						Provides: []libcnb.BuildPlanProvide{
							{Name: "jvm-application"},
							{Name: "java-app-server"},
						},
						Requires: []libcnb.BuildPlanRequire{
							{Name: "syft"},
							{Name: "jre", Metadata: map[string]interface{}{"launch": true}},
							{Name: "jvm-application-package"},
							{Name: "jvm-application"},
							{Name: "java-app-server"},
						},
					},
				},
			}))
		})
	})

	context("WEB-INF found", func() {
		it.Before(func() {
			Expect(os.MkdirAll(filepath.Join(path, "WEB-INF"), 0755)).To(Succeed())
		})

		it("requires and provides jvm-application-artifact", func() {
			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{
					{
						Provides: []libcnb.BuildPlanProvide{
							{Name: "jvm-application"},
							{Name: "java-app-server"},
							{Name: "jvm-application-package"},
						},
						Requires: []libcnb.BuildPlanRequire{
							{Name: "syft"},
							{Name: "jre", Metadata: map[string]interface{}{"launch": true}},
							{Name: "jvm-application-package"},
							{Name: "jvm-application"},
							{Name: "java-app-server"},
						},
					},
				},
			}))
		})
	})

	context("BP_JAVA_APP_SERVER is set to `tomee`", func() {
		it("contributes Tomee", func() {
			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{
					{
						Provides: []libcnb.BuildPlanProvide{
							{Name: "jvm-application"},
							{Name: "java-app-server"},
						},
						Requires: []libcnb.BuildPlanRequire{
							{Name: "syft"},
							{Name: "jre", Metadata: map[string]interface{}{"launch": true}},
							{Name: "jvm-application-package"},
							{Name: "jvm-application"},
							{Name: "java-app-server"},
						},
					},
				},
			}))
		})
	})

	context("BP_JAVA_APP_SERVER is set to `foo`", func() {
		it.Before(func() {
			t.Setenv("BP_JAVA_APP_SERVER", "foo")
		})

		it("fails", func() {
			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{Pass: false}))
		})
	})
}

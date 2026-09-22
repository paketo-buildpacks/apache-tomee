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

package helper_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/apache-tomee/helper"
)

func testAccessLoggingSupport(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		a = helper.AccessLoggingSupport{}
	)

	it("returns if $BPL_TOMEE_ACCESS_LOGGING_ENABLED is not set", func() {
		Expect(a.Execute()).To(BeNil())
	})

	context("$BPL_TOMEE_ACCESS_LOGGING_ENABLED", func() {
		it.Before(func() {
			t.Setenv("BPL_TOMEE_ACCESS_LOGGING_ENABLED", "")
		})

		it("contributes configuration", func() {
			Expect(a.Execute()).To(Equal(map[string]string{"JAVA_TOOL_OPTIONS": "-Daccess.logging.enabled=true"}))
		})

		context("$JAVA_TOOL_OPTIONS", func() {
			it.Before(func() {
				t.Setenv("JAVA_TOOL_OPTIONS", "test-java-tool-options")
			})

			it("contributes configuration appended to existing $JAVA_TOOL_OPTIONS", func() {
				Expect(a.Execute()).To(Equal(map[string]string{
					"JAVA_TOOL_OPTIONS": "test-java-tool-options -Daccess.logging.enabled=true",
				}))
			})
		})
	})

}

package integration_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega/format"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	buildpack string
	root      string

	buildpackInfo struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip integration tests in short mode")
	}

	format.MaxLength = 0

	buildpack = fmt.Sprintf("ttl.sh/%s-%s:1h", os.Getenv("PACKAGE"), os.Getenv("VERSION"))
	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault)
	suite.Run(t)
}

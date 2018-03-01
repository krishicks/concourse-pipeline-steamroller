package steamroller_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConcoursePipelineSteamroller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConcoursePipelineSteamroller Suite")
}

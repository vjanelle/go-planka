package planka_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlanka(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Planka Test Suite")
}

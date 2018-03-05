package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSearchType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Analyzer Tests")
}

var _ = Describe("SearchType", func() {
	It("should return correct string()", func() {
		Expect(SearchModeAll.String()).To(BeEquivalentTo("ALL"))
	})
	It("should parse from string correctly", func() {
		Expect(FromString("LAUNCH_NAME")).To(BeEquivalentTo(SearchModeLaunchName))
	})
})

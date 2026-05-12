package framework

import (
	"github.com/onsi/gomega"
)

// ExpectEqual expects the specified two are the same, otherwise an exception raises
func ExpectEqual(actual any, extra any, explain ...any) {
	gomega.ExpectWithOffset(1, actual).To(gomega.Equal(extra), explain...)
}

// ExpectEqualValues expects the specified two are the same, it not strict about type
func ExpectEqualValues(actual any, extra any, explain ...any) {
	gomega.ExpectWithOffset(1, actual).To(gomega.BeEquivalentTo(extra), explain...)
}

func ExpectEqualValuesWithOffset(offset int, actual any, extra any, explain ...any) {
	gomega.ExpectWithOffset(1+offset, actual).To(gomega.BeEquivalentTo(extra), explain...)
}

// ExpectNotEqual expects the specified two are not the same, otherwise an exception raises
func ExpectNotEqual(actual any, extra any, explain ...any) {
	gomega.ExpectWithOffset(1, actual).NotTo(gomega.Equal(extra), explain...)
}

func ExpectErrorWithOffset(offset int, err error, explain ...any) {
	gomega.ExpectWithOffset(1+offset, err).To(gomega.HaveOccurred(), explain...)
}

// ExpectNoError checks if "err" is set, and if so, fails assertion while logging the error.
func ExpectNoError(err error, explain ...any) {
	ExpectNoErrorWithOffset(1, err, explain...)
}

// ExpectNoErrorWithOffset checks if "err" is set, and if so, fails assertion while logging the error at "offset" levels above its caller
// (for example, for call chain f -> g -> ExpectNoErrorWithOffset(1, ...) error would be logged for "f").
func ExpectNoErrorWithOffset(offset int, err error, explain ...any) {
	gomega.ExpectWithOffset(1+offset, err).NotTo(gomega.HaveOccurred(), explain...)
}

func ExpectContainSubstring(actual, substr string, explain ...any) {
	gomega.ExpectWithOffset(1, actual).To(gomega.ContainSubstring(substr), explain...)
}

func ExpectContainElements(actual any, extra any, explain ...any) {
	gomega.ExpectWithOffset(1, actual).To(gomega.ContainElements(extra), explain...)
}

func ExpectNotContainElements(actual any, extra any, explain ...any) {
	gomega.ExpectWithOffset(1, actual).NotTo(gomega.ContainElements(extra), explain...)
}

func ExpectTrue(actual any, explain ...any) {
	gomega.ExpectWithOffset(1, actual).Should(gomega.BeTrue(), explain...)
}

func ExpectTrueWithOffset(offset int, actual any, explain ...any) {
	gomega.ExpectWithOffset(1+offset, actual).Should(gomega.BeTrue(), explain...)
}

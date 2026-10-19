package main

import (
	"errors"
	"testing"

	"github.com/choria-io/fisk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlankaCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Planka CLI Suite")
}

var _ = Describe("validateInstanceURL", func() {
	It("accepts an https URL", func() {
		got, err := validateInstanceURL("https://example.com")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal("https://example.com"))
	})

	It("accepts an http URL with a port", func() {
		got, err := validateInstanceURL("http://planka.local:8080")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal("http://planka.local:8080"))
	})

	It("trims surrounding whitespace", func() {
		got, err := validateInstanceURL("  https://example.com  ")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal("https://example.com"))
	})

	It("strips a single trailing slash", func() {
		got, err := validateInstanceURL("https://example.com/")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal("https://example.com"))
	})

	It("strips exactly one of two trailing slashes", func() {
		got, err := validateInstanceURL("https://example.com//")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(Equal("https://example.com/"))
	})

	It("rejects an empty string", func() {
		_, err := validateInstanceURL("")
		Expect(err).To(HaveOccurred())
	})

	It("rejects relative paths", func() {
		_, err := validateInstanceURL("/relative/path")
		Expect(err).To(HaveOccurred())
	})

	It("rejects non-URL strings", func() {
		_, err := validateInstanceURL("notaurl")
		Expect(err).To(HaveOccurred())
	})

	It("rejects strings without a scheme", func() {
		_, err := validateInstanceURL("example.com")
		Expect(err).To(HaveOccurred())
	})

	It("rejects non-http(s) schemes", func() {
		_, err := validateInstanceURL("ftp://example.com")
		Expect(err).To(HaveOccurred())
	})

	It("rejects a scheme without a host", func() {
		_, err := validateInstanceURL("https://")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("newLoginForm", func() {
	It("builds url, username, and password fields with prefill values", func() {
		url := "https://example.com"
		username := "admin"
		password := ""
		form := newLoginForm(&url, &username, &password)
		Expect(form).ToNot(BeNil())

		field := form.GetFocusedField()
		Expect(field).ToNot(BeNil())
		Expect(field.GetKey()).To(Equal("url"))
		Expect(field.GetValue()).To(Equal("https://example.com"))

		form.NextField()
		field = form.GetFocusedField()
		Expect(field).ToNot(BeNil())
		Expect(field.GetKey()).To(Equal("username"))
		Expect(field.GetValue()).To(Equal("admin"))

		form.NextField()
		field = form.GetFocusedField()
		Expect(field).ToNot(BeNil())
		Expect(field.GetKey()).To(Equal("password"))
		Expect(field.GetValue()).To(BeEmpty())
	})
})

var _ = Describe("newAPIKeyForm", func() {
	It("builds url and api_key fields with prefill values", func() {
		url := "https://example.com"
		apiKey := ""
		form := newAPIKeyForm(&url, &apiKey)
		Expect(form).ToNot(BeNil())

		field := form.GetFocusedField()
		Expect(field).ToNot(BeNil())
		Expect(field.GetKey()).To(Equal("url"))
		Expect(field.GetValue()).To(Equal("https://example.com"))

		form.NextField()
		field = form.GetFocusedField()
		Expect(field).ToNot(BeNil())
		Expect(field.GetKey()).To(Equal("api_key"))
		Expect(field.GetValue()).To(BeEmpty())
	})
})

var _ = Describe("helpRequested", func() {
	It("detects --help", func() {
		Expect(helpRequested([]string{"auth", "login", "--help"})).To(BeTrue())
	})

	It("detects -h", func() {
		Expect(helpRequested([]string{"-h"})).To(BeTrue())
	})

	It("detects --help=true", func() {
		Expect(helpRequested([]string{"--help=true"})).To(BeTrue())
	})

	It("returns false for a plain invocation", func() {
		Expect(helpRequested([]string{"auth", "login"})).To(BeFalse())
	})

	It("returns false when all arguments are provided", func() {
		Expect(helpRequested([]string{"auth", "login", "https://x", "u", "p"})).To(BeFalse())
	})
})

var _ = Describe("firstMissingLoginArg", func() {
	It("returns nil when all arguments are set", func() {
		Expect(firstMissingLoginArg("https://x", "u", "p")).To(BeNil())
	})

	It("reports a missing url", func() {
		err := firstMissingLoginArg("", "u", "p")
		Expect(err).To(MatchError("required argument 'url' not provided"))
		Expect(errors.Is(err, fisk.ErrRequiredArgument)).To(BeTrue())
	})

	It("reports a missing username", func() {
		err := firstMissingLoginArg("https://x", "", "p")
		Expect(err).To(MatchError("required argument 'username' not provided"))
		Expect(errors.Is(err, fisk.ErrRequiredArgument)).To(BeTrue())
	})

	It("reports a missing password", func() {
		err := firstMissingLoginArg("https://x", "u", "")
		Expect(err).To(MatchError("required argument 'password' not provided"))
		Expect(errors.Is(err, fisk.ErrRequiredArgument)).To(BeTrue())
	})
})

var _ = Describe("firstMissingAPIKeyArg", func() {
	It("reports a missing url", func() {
		err := firstMissingAPIKeyArg("", "k")
		Expect(err).To(MatchError("required argument 'url' not provided"))
		Expect(errors.Is(err, fisk.ErrRequiredArgument)).To(BeTrue())
	})

	It("reports a missing key", func() {
		err := firstMissingAPIKeyArg("https://x", "")
		Expect(err).To(MatchError("required argument 'key' not provided"))
		Expect(errors.Is(err, fisk.ErrRequiredArgument)).To(BeTrue())
	})
})

var _ = Describe("configured", func() {
	It("is true when the config has a URL", func() {
		Expect(configured(&Config{URL: "https://x"})).To(BeTrue())
	})

	It("is false for an empty config", func() {
		Expect(configured(&Config{})).To(BeFalse())
	})
})

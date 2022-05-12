package main

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
)

var reItemHeader = regexp.MustCompile(`^- ([[:alpha:]]+:)`)

func emphasizeItemHeaders(body string) string {
	lines := strings.Split(body, "\n")
	inFence := false
	for i, l := range lines {
		if strings.HasPrefix(l, "```") {
			inFence = !inFence
		}
		if !inFence && strings.HasPrefix(l, "- ") {
			lines[i] = reItemHeader.ReplaceAllString(l, "- **$1**")
		}
	}
	return strings.Join(lines, "\n")
}

type link struct {
	name string
	url  string
}

// ChangeLog is a struct to generate changelog output from given repository URL
type ChangeLog struct {
	repoURL string
	out     io.Writer
}

// Generate generates changelog text from given releases and outputs it to its writer
func (cl *ChangeLog) Generate(rels []*github.RepositoryRelease) error {
	out := bufio.NewWriter(cl.out)

	numRels := len(rels)
	relLinks := make([]link, 0, numRels)
	for i, rel := range rels {
		prevTag := ""
		if i+1 < numRels {
			prevTag = rels[i+1].GetTagName()
		}

		tag := rel.GetTagName()

		var compareURL string
		if prevTag == "" {
			compareURL = fmt.Sprintf("%s/tree/%s", cl.repoURL, tag)
		} else {
			compareURL = fmt.Sprintf("%s/compare/%s...%s", cl.repoURL, prevTag, tag)
		}

		fmt.Fprintf(out, "<a name=\"%s\"></a>\n", tag)

		title := rel.GetName()
		if title == "" {
			title = tag
		} else if title != tag {
			title = fmt.Sprintf("%s (%s)", title, tag)
		}

		pageURL := fmt.Sprintf("%s/releases/tag/%s", cl.repoURL, tag)

		fmt.Fprintf(out, "# [%s](%s) - %s\n\n", title, pageURL, rel.GetPublishedAt().Format("02 Jan 2006"))
		fmt.Fprint(out, emphasizeItemHeaders(strings.Replace(rel.GetBody(), "\r", "", -1)))
		fmt.Fprintf(out, "\n\n[Changes][%s]\n\n\n", tag)

		relLinks = append(relLinks, link{tag, compareURL})
	}

	for _, link := range relLinks {
		fmt.Fprintf(out, "[%s]: %s\n", link.name, link.url)
	}

	fmt.Fprint(out, "\n <!-- Generated by https://github.com/rhysd/changelog-from-release -->\n")

	return out.Flush()
}

// NewChangeLog creates a new ChangeLog instance
func NewChangeLog(w io.Writer, u *url.URL) *ChangeLog {
	// Strip credentials in the repository URL (#9)
	u.User = nil
	return &ChangeLog{strings.TrimSuffix(u.String(), ".git"), w}
}

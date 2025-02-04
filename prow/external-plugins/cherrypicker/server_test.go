/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"k8s.io/test-infra/prow/git/v2"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/git/localgit"
	"k8s.io/test-infra/prow/github"
)

type fghc struct {
	sync.Mutex
	pr       *github.PullRequest
	isMember bool

	patch      []byte
	comments   []string
	prs        []github.PullRequest
	prComments []github.IssueComment
	prLabels   []github.Label
	createdNum int
	orgMembers []github.TeamMember
}

func (f *fghc) AssignIssue(org, repo string, number int, logins []string) error {
	f.Lock()
	defer f.Unlock()
	return nil
}

func (f *fghc) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	f.Lock()
	defer f.Unlock()
	return f.pr, nil
}

func (f *fghc) GetPullRequestPatch(org, repo string, number int) ([]byte, error) {
	f.Lock()
	defer f.Unlock()
	return f.patch, nil
}

func (f *fghc) GetPullRequests(org, repo string) ([]github.PullRequest, error) {
	f.Lock()
	defer f.Unlock()
	return f.prs, nil
}

func (f *fghc) CreateComment(org, repo string, number int, comment string) error {
	f.Lock()
	defer f.Unlock()
	f.comments = append(f.comments, fmt.Sprintf("%s/%s#%d %s", org, repo, number, comment))
	return nil
}

func (f *fghc) IsMember(org, user string) (bool, error) {
	f.Lock()
	defer f.Unlock()
	return f.isMember, nil
}

func (f *fghc) GetRepo(owner, name string) (github.FullRepo, error) {
	f.Lock()
	defer f.Unlock()
	return github.FullRepo{}, nil
}

var expectedFmt = `title=%q body=%q head=%s base=%s`

func prToString(pr github.PullRequest) string {
	return fmt.Sprintf(expectedFmt, pr.Title, pr.Body, pr.Head.Ref, pr.Base.Ref)
}

func (f *fghc) CreatePullRequest(org, repo, title, body, head, base string, canModify bool) (int, error) {
	f.Lock()
	defer f.Unlock()
	f.prs = append(f.prs, github.PullRequest{
		Title: title,
		Body:  body,
		Head:  github.PullRequestBranch{Ref: head},
		Base:  github.PullRequestBranch{Ref: base},
	})
	return f.createdNum, nil
}

func (f *fghc) ListIssueComments(org, repo string, number int) ([]github.IssueComment, error) {
	f.Lock()
	defer f.Unlock()
	return f.prComments, nil
}

func (f *fghc) GetIssueLabels(org, repo string, number int) ([]github.Label, error) {
	f.Lock()
	defer f.Unlock()
	return f.prLabels, nil
}

func (f *fghc) ListOrgMembers(org, role string) ([]github.TeamMember, error) {
	f.Lock()
	defer f.Unlock()
	if role != "all" {
		return nil, fmt.Errorf("all is only supported role, not: %s", role)
	}
	return f.orgMembers, nil
}

func (f *fghc) CreateFork(org, repo string) error {
	return nil
}

var initialFiles = map[string][]byte{
	"bar.go": []byte(`// Package bar does an interesting thing.
package bar

// Foo does a thing.
func Foo(wow int) int {
	return 42 + wow
}
`),
}

var patch = []byte(`From af468c9e69dfdf39db591f1e3e8de5b64b0e62a2 Mon Sep 17 00:00:00 2001
From: Wise Guy <wise@guy.com>
Date: Thu, 19 Oct 2017 15:14:36 +0200
Subject: [PATCH] Update magic number

---
 bar.go | 3 ++-
 1 file changed, 2 insertions(+), 1 deletion(-)

diff --git a/bar.go b/bar.go
index 1ea52dc..5bd70a9 100644
--- a/bar.go
+++ b/bar.go
@@ -3,5 +3,6 @@ package bar

 // Foo does a thing.
 func Foo(wow int) int {
-	return 42 + wow
+	// Needs to be 49 because of a reason.
+	return 49 + wow
 }
`)

var body = "This PR updates the magic number.\n\n```release-note\nUpdate the magic number from 42 to 49\n```"

func TestCherryPickIC(t *testing.T) {
	lg, c, err := localgit.New()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "stage"); err != nil {
		t.Fatalf("Checking out pull branch: %v", err)
	}

	ghc := &fghc{
		pr: &github.PullRequest{
			Base: github.PullRequestBranch{
				Ref: "master",
			},
			Merged: true,
			Title:  "This is a fix for X",
			Body:   body,
		},
		isMember:   true,
		createdNum: 3,
		patch:      patch,
	}
	ic := github.IssueCommentEvent{
		Action: github.IssueCommentActionCreated,
		Repo: github.Repo{
			Owner: github.User{
				Login: "foo",
			},
			Name:     "bar",
			FullName: "foo/bar",
		},
		Issue: github.Issue{
			Number:      2,
			State:       "closed",
			PullRequest: &struct{}{},
		},
		Comment: github.IssueComment{
			User: github.User{
				Login: "wiseguy",
			},
			Body: "/cherrypick stage",
		},
	}

	botName := "ci-robot"
	expectedTitle := "[stage] This is a fix for X"
	expectedBody := "This is an automated cherry-pick of #2\n\n/assign wiseguy\n\n```release-note\nUpdate the magic number from 42 to 49\n```"
	expectedBase := "stage"
	expectedHead := fmt.Sprintf(botName+":"+cherryPickBranchFmt, 2, expectedBase)
	expected := fmt.Sprintf(expectedFmt, expectedTitle, expectedBody, expectedHead, expectedBase)

	getSecret := func() []byte {
		return []byte("sha=abcdefg")
	}

	s := &Server{
		botName:        botName,
		gc:             git.ClientFactoryFrom(c),
		push:           func(newBranch string) error { return nil },
		ghc:            ghc,
		tokenGenerator: getSecret,
		log:            logrus.StandardLogger().WithField("client", "cherrypicker"),
		repos:          []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},

		prowAssignments: true,
	}

	if err := s.handleIssueComment(logrus.NewEntry(logrus.StandardLogger()), ic); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	got := prToString(ghc.prs[0])
	if got != expected {
		t.Errorf("Expected (%d):\n%s\nGot (%d):\n%+v\n", len(expected), expected, len(got), got)
	}
}

func TestCherryPickPR(t *testing.T) {
	lg, c, err := localgit.New()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "release-1.5"); err != nil {
		t.Fatalf("Checking out pull branch: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "release-1.6"); err != nil {
		t.Fatalf("Checking out pull branch: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "cherry-pick-2-to-release-1.5"); err != nil {
		t.Fatalf("Checking out existing PR branch: %v", err)
	}

	ghc := &fghc{
		orgMembers: []github.TeamMember{
			{
				Login: "approver",
			},
			{
				Login: "merge-bot",
			},
		},
		prComments: []github.IssueComment{
			{
				User: github.User{
					Login: "developer",
				},
				Body: "a review comment",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/cherrypick release-1.5\r",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/cherrypick release-1.6",
			},
			{
				User: github.User{
					Login: "fan",
				},
				Body: "/cherrypick release-1.7",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/approve",
			},
			{
				User: github.User{
					Login: "merge-bot",
				},
				Body: "Automatic merge from submit-queue.",
			},
		},
		prs: []github.PullRequest{
			{
				Title: "[release-1.5] This is a fix for Y",
				Body:  "This is an automated cherry-pick of #2",
				Base: github.PullRequestBranch{
					Ref: "release-1.5",
				},
				Head: github.PullRequestBranch{
					Ref: "ci-robot:cherry-pick-2-to-release-1.5",
				},
			},
		},
		isMember:   true,
		createdNum: 3,
		patch:      patch,
	}
	pr := github.PullRequestEvent{
		Action: github.PullRequestActionClosed,
		PullRequest: github.PullRequest{
			Base: github.PullRequestBranch{
				Ref: "master",
				Repo: github.Repo{
					Owner: github.User{
						Login: "foo",
					},
					Name: "bar",
				},
			},
			Number:   2,
			Merged:   true,
			MergeSHA: new(string),
			Title:    "This is a fix for Y",
		},
	}

	botName := "ci-robot"

	getSecret := func() []byte {
		return []byte("sha=abcdefg")
	}

	s := &Server{
		botName:        botName,
		gc:             git.ClientFactoryFrom(c),
		push:           func(newBranch string) error { return nil },
		ghc:            ghc,
		tokenGenerator: getSecret,
		log:            logrus.StandardLogger().WithField("client", "cherrypicker"),
		repos:          []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},

		prowAssignments: false,
	}

	if err := s.handlePullRequest(logrus.NewEntry(logrus.StandardLogger()), pr); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	var expectedFn = func(branch string) string {
		expectedTitle := fmt.Sprintf("[%s] This is a fix for Y", branch)
		expectedBody := "This is an automated cherry-pick of #2"
		expectedHead := fmt.Sprintf(botName+":"+cherryPickBranchFmt, 2, branch)
		return fmt.Sprintf(expectedFmt, expectedTitle, expectedBody, expectedHead, branch)
	}

	if len(ghc.prs) != 2 {
		t.Fatalf("Expected %d PRs, got %d", 2, len(ghc.prs))
	}

	expectedBranches := []string{"release-1.5", "release-1.6"}
	seenBranches := make(map[string]struct{})
	for _, p := range ghc.prs {
		pr := prToString(p)
		if pr != expectedFn("release-1.5") && pr != expectedFn("release-1.6") {
			t.Errorf("Unexpected PR:\n%s\nExpected to target one of the following branches: %v\n%s", pr, expectedBranches, expectedFn("release-1.5"))
		}
		if pr == expectedFn("release-1.5") {
			seenBranches["release-1.5"] = struct{}{}
		}
		if pr == expectedFn("release-1.6") {
			seenBranches["release-1.6"] = struct{}{}
		}
	}
	if len(seenBranches) != 2 {
		t.Fatalf("Expected to see PRs for %d branches, got %d (%v)", 2, len(seenBranches), seenBranches)
	}
}

func TestCherryPickPRWithLabels(t *testing.T) {
	lg, c, err := localgit.New()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "release-1.5"); err != nil {
		t.Fatalf("Checking out pull branch: %v", err)
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "release-1.6"); err != nil {
		t.Fatalf("Checking out pull branch: %v", err)
	}

	pr := func(evt github.PullRequestEventAction) github.PullRequestEvent {
		return github.PullRequestEvent{
			Action: evt,
			PullRequest: github.PullRequest{
				User: github.User{
					Login: "developer",
				},
				Base: github.PullRequestBranch{
					Ref: "master",
					Repo: github.Repo{
						Owner: github.User{
							Login: "foo",
						},
						Name: "bar",
					},
				},
				Number:   2,
				Merged:   true,
				MergeSHA: new(string),
				Title:    "This is a fix for Y",
			},
		}
	}

	events := []github.PullRequestEventAction{github.PullRequestActionClosed, github.PullRequestActionLabeled}

	botName := "ci-robot"

	getSecret := func() []byte {
		return []byte("sha=abcdefg")
	}

	for _, evt := range events {
		ghc := &fghc{
			orgMembers: []github.TeamMember{
				{
					Login: "approver",
				},
				{
					Login: "merge-bot",
				},
				{
					Login: "developer",
				},
			},
			prComments: []github.IssueComment{
				{
					User: github.User{
						Login: "developer",
					},
					Body: "a review comment",
				},
			},
			prLabels: []github.Label{
				{
					Name: "cherrypick/release-1.5",
				},
				{
					Name: "cherrypick/release-1.6",
				},
				{
					Name: "cherrypick/release-1.7",
				},
			},
			isMember:   true,
			createdNum: 3,
			patch:      patch,
		}

		s := &Server{
			botName:        botName,
			gc:             git.ClientFactoryFrom(c),
			push:           func(newBranch string) error { return nil },
			ghc:            ghc,
			tokenGenerator: getSecret,
			log:            logrus.StandardLogger().WithField("client", "cherrypicker"),
			repos:          []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},

			prowAssignments: false,
		}

		if err := s.handlePullRequest(logrus.NewEntry(logrus.StandardLogger()), pr(evt)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var expectedFn = func(branch string) string {
			expectedTitle := fmt.Sprintf("[%s] This is a fix for Y", branch)
			expectedBody := "This is an automated cherry-pick of #2"
			expectedHead := fmt.Sprintf(botName+":"+cherryPickBranchFmt, 2, branch)
			return fmt.Sprintf(expectedFmt, expectedTitle, expectedBody, expectedHead, branch)
		}

		if len(ghc.prs) != 2 {
			t.Fatalf("Expected %d PRs, got %d", 2, len(ghc.prs))
		}

		expectedBranches := []string{"release-1.5", "release-1.6"}
		seenBranches := make(map[string]struct{})
		for _, p := range ghc.prs {
			pr := prToString(p)
			if pr != expectedFn("release-1.5") && pr != expectedFn("release-1.6") {
				t.Errorf("Unexpected PR:\n%s\nExpected to target one of the following branches: %v", pr, expectedBranches)
			}
			if pr == expectedFn("release-1.5") {
				seenBranches["release-1.5"] = struct{}{}
			}
			if pr == expectedFn("release-1.6") {
				seenBranches["release-1.6"] = struct{}{}
			}
		}
		if len(seenBranches) != 2 {
			t.Fatalf("Expected to see PRs for %d branches, got %d (%v)", 2, len(seenBranches), seenBranches)
		}
	}
}

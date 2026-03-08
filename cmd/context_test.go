package cmd

import (
	"testing"
)

func TestParseTopicRef(t *testing.T) {
	cases := []struct {
		name        string
		ref         string
		projectFlag string
		wantProject string
		wantTopic   string
		wantVersion string
	}{
		{
			name:        "plain topic uses project flag",
			ref:         "plain_topic",
			projectFlag: "default",
			wantProject: "default",
			wantTopic:   "plain_topic",
			wantVersion: "",
		},
		{
			name:        "plain topic with version suffix",
			ref:         "plain_topic@v2",
			projectFlag: "default",
			wantProject: "default",
			wantTopic:   "plain_topic",
			wantVersion: "v2",
		},
		{
			name:        "project/topic deeplink",
			ref:         "project/topic",
			projectFlag: "default",
			wantProject: "project",
			wantTopic:   "topic",
			wantVersion: "",
		},
		{
			name:        "_global/ deeplink",
			ref:         "_global/devops",
			projectFlag: "default",
			wantProject: "_global",
			wantTopic:   "devops",
			wantVersion: "",
		},
		{
			name:        "project/topic@version deeplink",
			ref:         "stompy/spec@v1.0",
			projectFlag: "default",
			wantProject: "stompy",
			wantTopic:   "spec",
			wantVersion: "v1.0",
		},
		{
			name:        "empty project flag with plain topic",
			ref:         "my_topic",
			projectFlag: "",
			wantProject: "",
			wantTopic:   "my_topic",
			wantVersion: "",
		},
		{
			name:        "deeplink overrides project flag",
			ref:         "other_project/topic",
			projectFlag: "my_project",
			wantProject: "other_project",
			wantTopic:   "topic",
			wantVersion: "",
		},
		{
			name:        "deeplink with hyphenated topic",
			ref:         "myapp/auth-rules",
			projectFlag: "default",
			wantProject: "myapp",
			wantTopic:   "auth-rules",
			wantVersion: "",
		},
		{
			name:        "_global deeplink with version",
			ref:         "_global/devops_ci@v3",
			projectFlag: "default",
			wantProject: "_global",
			wantTopic:   "devops_ci",
			wantVersion: "v3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotProject, gotTopic, gotVersion := parseTopicRef(tc.ref, tc.projectFlag)
			if gotProject != tc.wantProject {
				t.Errorf("project: got %q, want %q", gotProject, tc.wantProject)
			}
			if gotTopic != tc.wantTopic {
				t.Errorf("topic: got %q, want %q", gotTopic, tc.wantTopic)
			}
			if gotVersion != tc.wantVersion {
				t.Errorf("version: got %q, want %q", gotVersion, tc.wantVersion)
			}
		})
	}
}

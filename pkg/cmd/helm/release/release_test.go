package release_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helm/release"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/fakerunners"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"

	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func TestStepHelmRelease(t *testing.T) {
	runner := fakerunners.NewFakeRunnerWithGitClone()
	helmBin := "helm"

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-kubernetes.git"

	requirements := jxcore.NewRequirementsConfig()
	requirements.Spec.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
	data, err := yaml.Marshal(requirements)
	require.NoError(t, err, "failed to marshal requirements")
	devEnv.Spec.TeamSettings.BootRequirements = string(data)

	jxClient := jxfake.NewSimpleClientset(devEnv)

	_, o := release.NewCmdHelmRelease()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join("testdata", "charts")
	o.JXClient = jxClient
	o.Namespace = ns
	o.Version = "1.2.3"

	o.KubeClient = fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kube.SecretJenkinsChartMuseum,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"BASIC_AUTH_USER": []byte("myuser"),
				"BASIC_AUTH_PASS": []byte("mypwd"),
			},
		},
	)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	for _, c := range runner.OrderedCommands {
		t.Logf("ran: %s\n", c.CLI())
	}

	assert.Equal(t, o.ReleasedCharts, 1, "should have released 1 chart")
}

func TestStepHelmReleaseWithArtifactory(t *testing.T) {
	runner := fakerunners.NewFakeRunnerWithGitClone()
	helmBin := "helm"

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-kubernetes.git"

	requirements := jxcore.NewRequirementsConfig()
	requirements.Spec.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
	data, err := yaml.Marshal(requirements)
	require.NoError(t, err, "failed to marshal requirements")
	devEnv.Spec.TeamSettings.BootRequirements = string(data)

	jxClient := jxfake.NewSimpleClientset(devEnv)

	_, o := release.NewCmdHelmRelease()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join("testdata", "charts")
	o.JXClient = jxClient
	o.Namespace = ns
	o.Version = "2.3.4"
	o.Artifactory = true

	o.KubeClient = fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kube.SecretJenkinsChartMuseum,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"BASIC_AUTH_USER": []byte("myuser"),
				"BASIC_AUTH_PASS": []byte("mypwd"),
			},
		},
	)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	for _, c := range runner.OrderedCommands {
		t.Logf("ran: %s\n", c.CLI())
	}
}

func TestStepHelmReleaseWithChartPages(t *testing.T) {
	runner := fakerunners.NewFakeRunnerWithGitClone()
	helmBin := "helm"

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-kubernetes.git"

	requirements := jxcore.NewRequirementsConfig()
	requirements.Spec.Repository = "none"
	requirements.Spec.Cluster.ChartRepository = "http://pages/pages/chart/"
	// doesn't do anything
	requirements.Spec.Cluster.ChartKind = "pages"
	data, err := yaml.Marshal(requirements)
	require.NoError(t, err, "failed to marshal requirements")
	devEnv.Spec.TeamSettings.BootRequirements = string(data)

	jxClient := jxfake.NewSimpleClientset(devEnv)

	_, o := release.NewCmdHelmRelease()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join("testdata", "charts")
	o.JXClient = jxClient
	o.Namespace = ns
	o.Version = "1.2.3"
	// force ChartPages to true
	o.ChartPages = true
	// fake GiHubPages vars
	o.GitHubPagesDir = "testdata/pages/"
	o.GithubPagesURL = "http://bucketrepo.jx.svc.cluster.local/bucketrepo/pages"

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	for _, c := range runner.OrderedCommands {
		t.Logf("ran: %s\n", c.CLI())
	}

	assert.Equal(t, o.ReleasedCharts, 1, "should have released 1 chart")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			// workaround for dynamically generated git clone destination folder
			CLI: runner.OrderedCommands[0].Name + " " + strings.Join(runner.OrderedCommands[0].Args, " "),
		},
		fakerunner.FakeResult{
			CLI: "helm dependency build .",
		},
		fakerunner.FakeResult{
			CLI: "helm lint",
		},
		fakerunner.FakeResult{
			CLI: "helm package .",
		},
		fakerunner.FakeResult{
			CLI: "helm repo index .",
		},
		fakerunner.FakeResult{
			CLI: "git add *",
		},
		fakerunner.FakeResult{
			CLI: "git status -s",
		},
		fakerunner.FakeResult{
			CLI: "git commit -m chore: add helm chart for myapp v1.2.3",
		},
		fakerunner.FakeResult{
			CLI: "git push --set-upstream origin gh-pages",
		},
	)
}

package e2e

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func prepareNoTargetTest(t *testing.T, withDeploymentYaml bool) *TestProject {
	p := NewTestProject(t, defaultCluster1)
	p.mergeKubeconfig(defaultCluster2)

	createNamespace(t, defaultCluster1, p.testSlug())
	createNamespace(t, defaultCluster2, p.testSlug())

	cm := createConfigMapObject(map[string]string{
		"targetName":    `{{ target.name }}`,
		"targetContext": `{{ target.context }}`,
	}, resourceOpts{
		name:      "cm",
		namespace: p.testSlug(),
	})

	if withDeploymentYaml {
		p.addKustomizeDeployment("cm", []kustomizeResource{{name: "cm.yaml", content: cm}}, nil)
	} else {
		p.addKustomizeResources("", []kustomizeResource{{name: "cm.yaml", content: cm}})
		err := os.Remove(filepath.Join(p.gitServer.LocalRepoDir(p.getKluctlProjectRepo()), "deployment.yml"))
		assert.NoError(t, err)
	}

	return p
}

func testNoTarget(t *testing.T, withDeploymentYaml bool) {
	t.Parallel()

	p := prepareNoTargetTest(t, withDeploymentYaml)

	p.KluctlMust("deploy", "--yes")
	cm := assertConfigMapExists(t, defaultCluster1, p.testSlug(), "cm")
	assertConfigMapNotExists(t, defaultCluster2, p.testSlug(), "cm")
	assert.Equal(t, map[string]any{
		"targetName":    "",
		"targetContext": defaultCluster1.Context,
	}, cm.Object["data"])

	p.KluctlMust("deploy", "--yes", "-T", "override-name")
	cm = assertConfigMapExists(t, defaultCluster1, p.testSlug(), "cm")
	assert.Equal(t, map[string]any{
		"targetName":    "override-name",
		"targetContext": defaultCluster1.Context,
	}, cm.Object["data"])

	p.KluctlMust("deploy", "--yes", "-T", "override-name", "--context", defaultCluster2.Context)
	cm = assertConfigMapExists(t, defaultCluster2, p.testSlug(), "cm")
	assert.Equal(t, map[string]any{
		"targetName":    "override-name",
		"targetContext": defaultCluster2.Context,
	}, cm.Object["data"])
}

func TestNoTarget(t *testing.T) {
	testNoTarget(t, true)
}

func TestNoTargetNoDeployment(t *testing.T) {
	testNoTarget(t, false)
}

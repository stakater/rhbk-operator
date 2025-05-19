package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gkampitakis/go-snaps/match"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/dsl/core"
)

var excludeFields = []string{
	"$.metadata.uid",
	"$.metadata.resourceVersion",
	"$.metadata.creationTimestamp",
	"$.metadata.annotations",
	"$.metadata.managedFields",
	"$.metadata.ownerReferences",
	"$.spec.selector",
	"$.spec.template.metadata",
	"$.status.startTime",
	"$.status.completionTime",
	"$.status.conditions[*].lastTransitionTime",
	"$.status.conditions[*].lastProbeTime",
	"$.status.lastSuccessfulRunTime",
	"$.status.uncountedTerminatedPods",
	"$.status.lastScheduleTime",
	"$.status.lastSuccessfulTime",
	"$.status.conditions[*].lastUpdateTime",
	"$.status.ingress",
}

var excludeJobFields = append(excludeFields, []string{
	"$.metadata.name",
}...)

var excludeFieldMap = map[string][]string{
	"job": excludeJobFields,
}

func MatchYAMLResource(resource interface{}, snapshotName ...string) {
	kind := strings.ToLower(GetKind(resource))
	currentSpec := ginkgo.CurrentSpecReport()
	name := strings.Join(snapshotName, "_")
	if name == "" {
		name = currentSpec.LeafNodeText
	}

	name = fmt.Sprintf("[%s] %s", kind, name)
	exclude, ok := excludeFieldMap[kind]
	if !ok {
		exclude = excludeFields
	}

	snaps.WithConfig(
		snaps.Dir(fmt.Sprintf("__snapshots__/%s/%s", filepath.Base(currentSpec.FileName()), currentSpec.LeafNodeText)),
		snaps.Filename(name),
		snaps.Ext(".yaml"),
	).MatchYAML(
		core.GinkgoT(),
		resource,
		match.Any(exclude...).ErrOnMissingPath(false),
	)
}

func MatchResource(resource interface{}, kind string, snapshotName ...string) {
	currentSpec := ginkgo.CurrentSpecReport()
	name := strings.Join(snapshotName, "_")
	if name == "" {
		name = currentSpec.LeafNodeText
	}

	name = fmt.Sprintf("[%s] %s", kind, name)
	exclude, ok := excludeFieldMap[kind]
	if !ok {
		exclude = excludeFields
	}

	snaps.WithConfig(
		snaps.Dir(fmt.Sprintf("__snapshots__/%s/%s", filepath.Base(currentSpec.FileName()), currentSpec.LeafNodeText)),
		snaps.Filename(name),
		snaps.Ext(".yaml"),
	).MatchYAML(
		core.GinkgoT(),
		resource,
		match.Any(exclude...).ErrOnMissingPath(false),
	)
}

func MatchJsonResource(resource interface{}, kind string, snapshotName ...string) {
	currentSpec := ginkgo.CurrentSpecReport()
	name := strings.Join(snapshotName, "_")
	if name == "" {
		name = currentSpec.LeafNodeText
	}

	name = fmt.Sprintf("[%s] %s", kind, name)
	exclude, ok := excludeFieldMap[kind]
	if !ok {
		exclude = excludeFields
	}

	snaps.WithConfig(
		snaps.Dir(fmt.Sprintf("__snapshots__/%s/%s", filepath.Base(currentSpec.FileName()), currentSpec.LeafNodeText)),
		snaps.Filename(name),
		snaps.Ext(".yaml"),
	).MatchJSON(
		core.GinkgoT(),
		resource,
		match.Any(exclude...).ErrOnMissingPath(false),
	)
}

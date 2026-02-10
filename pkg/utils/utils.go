package utils

// cSpell: words kioutil

import (
	"fmt"
	"strconv"

	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var buildAnnotations = []string{
	BuildAnnotationPreviousKinds,
	BuildAnnotationPreviousNames,
	BuildAnnotationPrefixes,
	BuildAnnotationSuffixes,
	BuildAnnotationPreviousNamespaces,
	BuildAnnotationsRefBy,
	BuildAnnotationsGenBehavior,
	BuildAnnotationsGenAddHashSuffix,
}

// RemoveBuildAnnotations removes kustomize build annotations from r.
//
// Contrary to the method available in resource.Resource, this method doesn't
// remove the file name related annotations, as this would prevent modification
// of the source file.
func RemoveBuildAnnotations(r *resource.Resource) {
	annotations := r.GetAnnotations()
	if len(annotations) == 0 {
		return
	}
	for _, a := range buildAnnotations {
		delete(annotations, a)
	}
	if err := r.SetAnnotations(annotations); err != nil {
		panic(err)
	}
}

type AnnotationProperties struct {
	Path        string
	Kind        string
	ApiVersion  string
	Index       int
	Local       bool
	PathSet     bool
	IndexSet    bool
	InjectLocal bool
}

func GeAnnotationProperties(annotations map[string]string, config bool) *AnnotationProperties {
	var properties AnnotationProperties
	var err error

	_, properties.Local = annotations[FunctionAnnotationLocalConfig]

	if properties.Path, properties.PathSet = annotations[FunctionAnnotationPath]; !properties.PathSet {
		if config {
			properties.Path = ".karmafun.yaml"
		}
	}
	if indexStr, ok := annotations[FunctionAnnotationIndex]; ok {
		properties.IndexSet = true
		properties.Index, err = strconv.Atoi(indexStr)
		// Anything that is not a valid integer means we don't want to set an index at all.
		if err != nil {
			properties.Index = -1
		}
	}
	properties.InjectLocal = annotations[FunctionAnnotationInjectLocal] == "true"
	properties.Kind = annotations[FunctionAnnotationKind]
	properties.ApiVersion = annotations[FunctionAnnotationApiVersion]
	return &properties
}

func TransferAnnotationsToNode(r *yaml.RNode, configProperties *AnnotationProperties, index int) error {
	annotations := r.GetAnnotations()

	properties := GeAnnotationProperties(annotations, false)

	if configProperties.Local {
		annotations[FunctionAnnotationLocalConfig] = "true"
	}

	actualPath := configProperties.Path
	if properties.PathSet {
		actualPath = properties.Path
	}
	var curIndex string
	if properties.IndexSet {
		if properties.Index >= 0 {
			curIndex = strconv.Itoa(properties.Index)
		}
	} else if configProperties.Index >= 0 && !properties.PathSet {
		// If path is set on the resource, index should be set on the resource as well.
		curIndex = strconv.Itoa(configProperties.Index + index)
	}

	if actualPath != "" {
		//lint:ignore SA1019 used by kustomize
		annotations[kioutil.LegacyPathAnnotation] = actualPath //nolint:staticcheck // still in use.
		annotations[kioutil.PathAnnotation] = actualPath

		if curIndex != "" {
			//lint:ignore SA1019 used by kustomize
			annotations[kioutil.LegacyIndexAnnotation] = curIndex //nolint:staticcheck // still in use.
			annotations[kioutil.IndexAnnotation] = curIndex
		}
	}

	if properties.InjectLocal || configProperties.InjectLocal {
		// It's an heredoc document
		if configProperties.Kind != "" {
			r.SetKind(configProperties.Kind)
		}
		if configProperties.ApiVersion != "" {
			r.SetApiVersion(configProperties.ApiVersion)
		}
	}

	delete(annotations, FunctionAnnotationInjectLocal)
	delete(annotations, FunctionAnnotationFunction)
	delete(annotations, FunctionAnnotationPath)
	delete(annotations, FunctionAnnotationIndex)
	delete(annotations, FunctionAnnotationKind)
	delete(annotations, FunctionAnnotationApiVersion)
	delete(annotations, filters.LocalConfigAnnotation)
	if err := r.SetAnnotations(annotations); err != nil {
		return fmt.Errorf("while setting annotations on resource at index %d: %w", index, err)
	}
	return nil
}

func TransferAnnotations(list []*yaml.RNode, configAnnotations map[string]string) error {
	configProperties := GeAnnotationProperties(configAnnotations, true)

	for index, r := range list {
		if err := TransferAnnotationsToNode(r, configProperties, index); err != nil {
			return fmt.Errorf("while transferring annotations to resource at index %d: %w", index, err)
		}
	}
	return nil
}

func unLocal(list []*yaml.RNode) ([]*yaml.RNode, error) {
	output := []*yaml.RNode{}
	for _, r := range list {
		annotations := r.GetAnnotations()
		// We don't append resources with config.karmafun.dev/local-config resources
		if _, ok := annotations[FunctionAnnotationLocalConfig]; !ok {
			// For the remaining resources, if a path and/or index was specified
			// we copy it.
			if path, ok := annotations[FunctionAnnotationPath]; ok {
				//lint:ignore SA1019 used by kustomize
				annotations[kioutil.LegacyPathAnnotation] = path //nolint:staticcheck // still in use.
				annotations[kioutil.PathAnnotation] = path
				delete(annotations, FunctionAnnotationPath)
			}
			if index, ok := annotations[FunctionAnnotationIndex]; ok {
				//lint:ignore SA1019 used by kustomize
				annotations[kioutil.LegacyIndexAnnotation] = index //nolint:staticcheck // still in use.
				annotations[kioutil.IndexAnnotation] = index
				delete(annotations, FunctionAnnotationIndex)
			}

			if err := r.SetAnnotations(annotations); err != nil {
				return nil, fmt.Errorf("while setting annotations on resource: %w", err)
			}
			output = append(output, r)
		}
	}
	return output, nil
}

var UnLocal kio.FilterFunc = unLocal

func ResourceMapFromNodes(nodes []*yaml.RNode) resmap.ResMap {
	result := resmap.New()
	for _, n := range nodes {
		if err := result.Append(&resource.Resource{RNode: *n}); err != nil {
			panic(err)
		}
	}
	return result
}

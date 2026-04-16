// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package pml_domain

import (
	"maps"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_dto"
)

// StyleManager provides style lookup for components during transformation.
// It merges styles from all sources by precedence and gives access to the
// final computed values.
type StyleManager struct {
	// styles holds the computed attribute values keyed by name.
	styles map[string]string

	// sources tracks which source last set each attribute, allowing
	// components to distinguish user-provided values from defaults.
	sources map[string]AttributeSource
}

// NewStyleManager creates a StyleManager by merging styles from various sources
// based on the component's precedence rules.
//
// If comp is nil, returns an empty StyleManager (used when creating a context
// for future children).
//
// Precedence System:
//   - The component defines the order of sources via GetAttributePrecedence()
//   - Later sources in the array override earlier ones
//
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// extract styles from.
// Takes comp (Component) which defines the style precedence rules.
// Takes config (*pml_dto.Config) which provides configuration settings.
//
// Returns *StyleManager which contains the merged styles ready for use.
func NewStyleManager(node *ast_domain.TemplateNode, comp Component, config *pml_dto.Config) *StyleManager {
	mergedStyles := make(map[string]string)
	mergedSources := make(map[string]AttributeSource)

	if comp == nil {
		return &StyleManager{styles: mergedStyles, sources: mergedSources}
	}

	precedence := comp.GetAttributePrecedence()

	for _, source := range precedence {
		sourceStyles := extractSourceStyles(source, node, comp, config)
		mergeStylesWithSource(mergedStyles, mergedSources, sourceStyles, source)
	}

	return &StyleManager{styles: mergedStyles, sources: mergedSources}
}

// Get returns the computed value for a given attribute or property name.
//
// Takes name (string) which specifies the attribute or property to look up.
//
// Returns string which is the computed value for the requested name.
// Returns bool which indicates whether the name was found.
func (sm *StyleManager) Get(name string) (string, bool) {
	value, ok := sm.styles[name]
	return value, ok
}

// IsExplicit returns true when the attribute was set by the user (via inline
// attributes or the style attribute) rather than coming from the component's
// built-in defaults. This allows components to omit default values and let
// CSS cascade handle inheritance from parent elements.
//
// Takes name (string) which specifies the attribute to check.
//
// Returns bool which is true when the value was provided inline.
func (sm *StyleManager) IsExplicit(name string) bool {
	source, ok := sm.sources[name]
	return ok && source == SourceInline
}

// all returns the complete map of final computed styles.
//
// Returns map[string]string which contains all style key-value pairs.
func (sm *StyleManager) all() map[string]string {
	return sm.styles
}

// TransformationContext holds the state and settings available to a component
// during the PML transformation process.
type TransformationContext struct {
	// Registry provides access to registered components for tag lookup.
	Registry ComponentRegistry

	// MediaQueryCollector gathers responsive CSS rules for mobile breakpoints.
	MediaQueryCollector MediaQueryCollector

	// MSOConditionalCollector gathers conditional styles for Microsoft Outlook.
	MSOConditionalCollector MSOConditionalCollector

	// ParentComponent is the outer component in the
	// transformation hierarchy.
	ParentComponent Component

	// EmailAssetRegistry collects asset requests for CID embedding in emails.
	EmailAssetRegistry *EmailAssetRegistry

	// ParentNode is the parent node in the AST; nil means this is a root node.
	ParentNode *ast_domain.TemplateNode

	// StyleManager provides access to element styles during transformation.
	StyleManager *StyleManager

	// Config holds the PML settings used when creating style managers.
	Config *pml_dto.Config

	// InheritedTextAlign holds the explicit text-align value from a parent
	// row for propagation to child columns.
	InheritedTextAlign string

	// SourceFilePath is the path of the source .pk file being transformed.
	// Used to resolve module-relative (@/) asset paths in pml-img.
	SourceFilePath string

	// AssetServePath is the URL path prefix for serving assets in preview mode.
	AssetServePath string

	// ComponentPath tracks the hierarchy of parent component tag names.
	ComponentPath []string

	// diagnostics holds errors and warnings found during transformation.
	diagnostics []*Error

	// ContainerWidth is the width of the parent container in pixels.
	ContainerWidth float64

	// SiblingCount is the number of sibling elements at this level in the tree.
	SiblingCount int

	// IsInsideGroup indicates whether this context is within a non-stacking group.
	IsInsideGroup bool

	// IsEmailContext indicates whether the output is for an email template.
	IsEmailContext bool

	// IsPreviewMode indicates browser preview mode. When true, local image
	// paths are resolved to served asset URLs instead of CID references.
	IsPreviewMode bool
}

// CloneForChild creates a new context for a child node, inheriting parent
// information.
//
// Takes childNode (*ast_domain.TemplateNode) which is the child node to create
// a context for.
// Takes childComponent (Component) which is the component associated with the
// child node.
// Takes parentNode (*ast_domain.TemplateNode) which is the parent node in the
// template tree.
// Takes parentComponent (Component) which is the component associated with the
// parent node.
//
// Returns *TransformationContext which is a new context with inherited settings
// and a fresh diagnostics collection.
func (c *TransformationContext) CloneForChild(childNode *ast_domain.TemplateNode, childComponent Component, parentNode *ast_domain.TemplateNode, parentComponent Component) *TransformationContext {
	newPath := make([]string, len(c.ComponentPath)+1)
	copy(newPath, c.ComponentPath)
	if parentComponent != nil {
		newPath[len(c.ComponentPath)] = parentComponent.TagName()
	}

	return &TransformationContext{
		Config:                  c.Config,
		StyleManager:            NewStyleManager(childNode, childComponent, c.Config),
		ParentNode:              parentNode,
		ParentComponent:         parentComponent,
		ContainerWidth:          c.ContainerWidth,
		ComponentPath:           newPath,
		IsInsideGroup:           c.IsInsideGroup,
		SiblingCount:            c.SiblingCount,
		MediaQueryCollector:     c.MediaQueryCollector,
		MSOConditionalCollector: c.MSOConditionalCollector,
		Registry:                c.Registry,
		IsEmailContext:          c.IsEmailContext,
		EmailAssetRegistry:      c.EmailAssetRegistry,
		InheritedTextAlign:      c.InheritedTextAlign,
		SourceFilePath:          c.SourceFilePath,
		IsPreviewMode:           c.IsPreviewMode,
		AssetServePath:          c.AssetServePath,
		diagnostics:             make([]*Error, 0),
	}
}

// AddDiagnostic adds a new error or warning to the context.
//
// Takes message (string) which describes the issue found.
// Takes tagName (string) which identifies the documentation tag involved.
// Takes severity (Severity) which indicates if this is an error or warning.
// Takes location (ast_domain.Location) which specifies where the issue occurs.
func (c *TransformationContext) AddDiagnostic(message, tagName string, severity Severity, location ast_domain.Location) {
	c.diagnostics = append(c.diagnostics, &Error{
		Message:  message,
		TagName:  tagName,
		Severity: severity,
		Location: location,
	})
}

// Diagnostics returns all diagnostics collected within this context.
//
// Returns []*Error which contains the collected diagnostic errors.
func (c *TransformationContext) Diagnostics() []*Error {
	return c.diagnostics
}

// extractSourceStyles gets styles from a given source type.
//
// Takes source (AttributeSource) which specifies where to get styles from.
// Takes node (*ast_domain.TemplateNode) which provides the template node for
// inline style extraction.
// Takes comp (Component) which provides the component for default styles.
// Takes config (*pml_dto.Config) which specifies the configuration settings.
//
// Returns map[string]string which contains the extracted style key-value pairs,
// or nil when the source type is not recognised.
func extractSourceStyles(source AttributeSource, node *ast_domain.TemplateNode, comp Component, config *pml_dto.Config) map[string]string {
	switch source {
	case SourceDefault:
		return extractDefaultStyles(comp, config)
	case SourceInline:
		return extractInlineStyles(node)
	default:
		return nil
	}
}

// extractDefaultStyles builds a map of default styles for a component.
//
// It starts with the component's built-in defaults, then applies any
// overrides from the config. If the config clears defaults, only overrides
// are used.
//
// Takes comp (Component) which provides the default attributes for the element.
// Takes config (*pml_dto.Config) which specifies attribute overrides and
// whether to clear defaults.
//
// Returns map[string]string which contains the merged default styles, or nil
// if no defaults apply.
func extractDefaultStyles(comp Component, config *pml_dto.Config) map[string]string {
	var sourceStyles map[string]string

	if !config.ClearDefaultAttributes {
		sourceStyles = comp.DefaultAttributes()
	}

	if overrides, ok := config.OverrideAttributes[comp.TagName()]; ok {
		if sourceStyles == nil {
			sourceStyles = make(map[string]string)
		}
		maps.Copy(sourceStyles, overrides)
	}

	return sourceStyles
}

// extractInlineStyles gathers inline attributes and style values from a node.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// read styles from.
//
// Returns map[string]string which contains the style key-value pairs found.
func extractInlineStyles(node *ast_domain.TemplateNode) map[string]string {
	if node == nil {
		return nil
	}

	sourceStyles := make(map[string]string)

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Name != "style" && !strings.HasPrefix(attr.Name, "data-pml-") {
			sourceStyles[attr.Name] = attr.Value
		}
	}

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Name == "style" {
			inlineStyles := parseInlineStyle(attr.Value)
			maps.Copy(sourceStyles, inlineStyles)
			break
		}
	}

	return sourceStyles
}

// mergeStyles copies all styles from the source map into the target map.
//
// Takes target (map[string]string) which receives the copied styles.
// Takes source (map[string]string) which provides the styles to copy.
func mergeStyles(target map[string]string, source map[string]string) {
	maps.Copy(target, source)
}

// mergeStylesWithSource copies styles into the target map and records which
// source provided each value. Later sources overwrite earlier ones, matching
// the precedence behaviour of mergeStyles.
//
// Takes targetStyles (map[string]string) which receives the copied styles.
// Takes targetSources (map[string]AttributeSource) which records the source
// for each key.
// Takes source (map[string]string) which provides the styles to copy.
// Takes sourceType (AttributeSource) which identifies where these styles
// came from.
func mergeStylesWithSource(targetStyles map[string]string, targetSources map[string]AttributeSource, source map[string]string, sourceType AttributeSource) {
	for k, v := range source {
		targetStyles[k] = v
		targetSources[k] = sourceType
	}
}

// newRootTransformationContext creates a root context for a transformation pass.
//
// Takes config (*pml_dto.Config) which provides the configuration settings.
// Takes initialWidth (float64) which sets the starting container width.
// Takes registry (ComponentRegistry) which gives access to components.
//
// Returns *TransformationContext which is the root context ready for use.
func newRootTransformationContext(config *pml_dto.Config, initialWidth float64, registry ComponentRegistry) *TransformationContext {
	return &TransformationContext{
		Config:                  config,
		StyleManager:            &StyleManager{styles: make(map[string]string), sources: make(map[string]AttributeSource)},
		ParentNode:              nil,
		ParentComponent:         nil,
		MediaQueryCollector:     nil,
		MSOConditionalCollector: nil,
		Registry:                registry,
		EmailAssetRegistry:      nil,
		diagnostics:             make([]*Error, 0),
		ComponentPath:           make([]string, 0),
		ContainerWidth:          initialWidth,
		SiblingCount:            0,
		IsInsideGroup:           false,
		IsEmailContext:          false,
	}
}

// newRootTransformationContextForEmail creates a root context specifically for
// email rendering. It initialises the EmailAssetRegistry to collect asset
// embedding requests.
//
// Takes config (*pml_dto.Config) which provides the PML configuration settings.
// Takes initialWidth (float64) which sets the initial container width.
// Takes registry (ComponentRegistry) which provides component lookup.
//
// Returns *TransformationContext which is configured for email rendering with
// asset collection enabled.
func newRootTransformationContextForEmail(config *pml_dto.Config, initialWidth float64, registry ComponentRegistry) *TransformationContext {
	return &TransformationContext{
		Config:                  config,
		StyleManager:            &StyleManager{styles: make(map[string]string), sources: make(map[string]AttributeSource)},
		ParentNode:              nil,
		ParentComponent:         nil,
		MediaQueryCollector:     nil,
		MSOConditionalCollector: nil,
		Registry:                registry,
		EmailAssetRegistry:      NewEmailAssetRegistry(),
		diagnostics:             make([]*Error, 0),
		ComponentPath:           make([]string, 0),
		ContainerWidth:          initialWidth,
		SiblingCount:            0,
		IsInsideGroup:           false,
		IsEmailContext:          true,
		IsPreviewMode:           config.PreviewMode,
		AssetServePath:          config.AssetServePath,
	}
}

// parseInlineStyle parses a CSS style attribute string into a map.
//
// Takes styleAttr (string) which contains CSS property pairs split by
// semicolons, such as "color: red; font-size: 12px".
//
// Returns map[string]string which maps each CSS property name to its value.
func parseInlineStyle(styleAttr string) map[string]string {
	styles := make(map[string]string)
	if styleAttr == "" {
		return styles
	}
	for declaration := range strings.SplitSeq(styleAttr, ";") {
		parts := strings.SplitN(declaration, ":", 2)
		if len(parts) == 2 {
			property := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if property != "" && value != "" {
				styles[property] = value
			}
		}
	}
	return styles
}

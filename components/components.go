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

package components

import (
	"piko.sh/piko/internal/component/component_dto"
)

// modulePath is the Go module path for the built-in components package.
const modulePath = "piko.sh/piko/components"

var (
	// m3eAssets lists module-root-relative directories whose files should be
	// seeded into the registry alongside M3E PKC source files. These provide
	// SVG icons referenced by piko:svg elements within the components.
	m3eAssets = []string{"lib/icons"}

	// m2Assets lists module-root-relative directories whose files should be
	// seeded into the registry alongside M2 PKC source files. These provide
	// SVG icons referenced by piko:svg elements within the components.
	m2Assets = []string{"lib/icons"}
)

// Piko returns component definitions for the built-in piko-* components
// (e.g. piko-counter, piko-card).
//
// Returns []ComponentDefinition which contains the built-in piko-*
// component definitions.
func Piko() []component_dto.ComponentDefinition {
	return []component_dto.ComponentDefinition{
		{TagName: "piko-counter", SourcePath: "piko/piko-counter.pkc", ModulePath: modulePath, IsExternal: true},
		{TagName: "piko-card", SourcePath: "piko/piko-card.pkc", ModulePath: modulePath, IsExternal: true},
	}
}

// Example returns component definitions for the example-* components.
// These are provided as reference implementations and for testing.
//
// Returns []ComponentDefinition which contains the example-*
// component definitions.
func Example() []component_dto.ComponentDefinition {
	return []component_dto.ComponentDefinition{
		{TagName: "example-greeting", SourcePath: "example/example-greeting.pkc", ModulePath: modulePath, IsExternal: true},
	}
}

// M2 returns component definitions for components based on the Material
// Design 2 specification. These components use the m2-* tag prefix and
// integrate into the M3E design system for theming.
//
// Returns []ComponentDefinition which contains the m2-* component
// definitions.
func M2() []component_dto.ComponentDefinition { //nolint:revive // intentional acronym
	return []component_dto.ComponentDefinition{
		{TagName: "m2-data-table", SourcePath: "m2/m2-data-table.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m2Assets},
		{TagName: "m2-data-table-cell", SourcePath: "m2/m2-data-table-cell.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m2Assets},
		{TagName: "m2-data-table-header", SourcePath: "m2/m2-data-table-header.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m2Assets},
		{TagName: "m2-data-table-pagination", SourcePath: "m2/m2-data-table-pagination.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m2Assets},
		{TagName: "m2-data-table-row", SourcePath: "m2/m2-data-table-row.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m2Assets},
	}
}

// M3E returns component definitions for the Material Design 3 Expressive
// component library. All components use the m3e-* tag prefix.
//
// Returns []ComponentDefinition which contains the m3e-* component
// definitions.
func M3E() []component_dto.ComponentDefinition { //nolint:revive // intentional acronym
	return []component_dto.ComponentDefinition{
		{TagName: "m3e-badge", SourcePath: "m3e/m3e-badge.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-bottom-app-bar", SourcePath: "m3e/m3e-bottom-app-bar.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-bottom-sheet", SourcePath: "m3e/m3e-bottom-sheet.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-button", SourcePath: "m3e/m3e-button.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-button-group", SourcePath: "m3e/m3e-button-group.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-card", SourcePath: "m3e/m3e-card.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-carousel", SourcePath: "m3e/m3e-carousel.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-checkbox", SourcePath: "m3e/m3e-checkbox.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-chip", SourcePath: "m3e/m3e-chip.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-chip-set", SourcePath: "m3e/m3e-chip-set.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-date-picker", SourcePath: "m3e/m3e-date-picker.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-dialog", SourcePath: "m3e/m3e-dialog.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-divider", SourcePath: "m3e/m3e-divider.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-elevation", SourcePath: "m3e/m3e-elevation.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-extended-fab", SourcePath: "m3e/m3e-extended-fab.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-fab", SourcePath: "m3e/m3e-fab.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-fab-menu", SourcePath: "m3e/m3e-fab-menu.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-icon", SourcePath: "m3e/m3e-icon.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-icon-button", SourcePath: "m3e/m3e-icon-button.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-list", SourcePath: "m3e/m3e-list.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-list-item", SourcePath: "m3e/m3e-list-item.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-loading-indicator", SourcePath: "m3e/m3e-loading-indicator.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-menu", SourcePath: "m3e/m3e-menu.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-menu-item", SourcePath: "m3e/m3e-menu-item.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-navigation-bar", SourcePath: "m3e/m3e-navigation-bar.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-navigation-drawer", SourcePath: "m3e/m3e-navigation-drawer.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-navigation-rail", SourcePath: "m3e/m3e-navigation-rail.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-progress", SourcePath: "m3e/m3e-progress.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-radio", SourcePath: "m3e/m3e-radio.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-radio-group", SourcePath: "m3e/m3e-radio-group.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-ripple", SourcePath: "m3e/m3e-ripple.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-search", SourcePath: "m3e/m3e-search.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-segmented-button", SourcePath: "m3e/m3e-segmented-button.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-select", SourcePath: "m3e/m3e-select.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-side-sheet", SourcePath: "m3e/m3e-side-sheet.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-slider", SourcePath: "m3e/m3e-slider.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-snackbar", SourcePath: "m3e/m3e-snackbar.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-split-button", SourcePath: "m3e/m3e-split-button.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-switch", SourcePath: "m3e/m3e-switch.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-tab", SourcePath: "m3e/m3e-tab.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-tabs", SourcePath: "m3e/m3e-tabs.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-text-field", SourcePath: "m3e/m3e-text-field.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-time-picker", SourcePath: "m3e/m3e-time-picker.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-toolbar", SourcePath: "m3e/m3e-toolbar.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-tooltip", SourcePath: "m3e/m3e-tooltip.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
		{TagName: "m3e-top-app-bar", SourcePath: "m3e/m3e-top-app-bar.pkc", ModulePath: modulePath, IsExternal: true, AssetPaths: m3eAssets},
	}
}

// Dev returns component definitions for the developer tools overlay widget.
// These components are intended for dev mode only and are NOT included in All().
//
// Returns []ComponentDefinition which contains the dev-only component
// definitions.
func Dev() []component_dto.ComponentDefinition {
	return []component_dto.ComponentDefinition{
		{TagName: "piko-dev-widget", SourcePath: "dev/piko-dev-widget.pkc", ModulePath: modulePath, IsExternal: true},
	}
}

// All returns component definitions for every built-in component across all
// categories. Dev components are excluded; use Dev() to register them
// separately in dev mode.
//
// Returns []ComponentDefinition which contains definitions from all
// component categories combined.
func All() []component_dto.ComponentDefinition {
	piko := Piko()
	example := Example()
	m2 := M2()
	m3e := M3E()
	all := make([]component_dto.ComponentDefinition, 0, len(piko)+len(example)+len(m2)+len(m3e))
	all = append(all, piko...)
	all = append(all, example...)
	all = append(all, m2...)
	all = append(all, m3e...)
	return all
}

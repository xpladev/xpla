package dynamicdeflation_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBeginBlockOrderPlacesDynamicDeflationBetweenMintAndDistribution(t *testing.T) {
	order := beginBlockModuleSelectors(t)

	mintIndex := moduleIndex(t, order, "minttypes")
	dynamicDeflationIndex := moduleIndex(t, order, "dynamicdeflationtypes")
	distributionIndex := moduleIndex(t, order, "distrtypes")

	require.Equal(t, mintIndex+1, dynamicDeflationIndex)
	require.Equal(t, dynamicDeflationIndex+1, distributionIndex)
}

func TestBeginBlockOrderPreservesExistingModuleRelativeOrder(t *testing.T) {
	order := beginBlockModuleSelectors(t)
	order = withoutModule(order, "dynamicdeflationtypes")

	require.Equal(t, []string{
		"minttypes",
		"distrtypes",
		"slashingtypes",
		"evidencetypes",
		"stakingtypes",
		"authtypes",
		"banktypes",
		"govtypes",
		"crisistypes",
		"ibcexported",
		"ibctransfertypes",
		"icatypes",
		"pfmroutertypes",
		"ratelimittypes",
		"genutiltypes",
		"authz",
		"feegrant",
		"paramstypes",
		"consensusparamtypes",
		"wasmtypes",
		"feemarkettypes",
		"evmtypes",
		"rewardtypes",
		"volunteertypes",
		"burntypes",
	}, order)
}

func beginBlockModuleSelectors(t *testing.T) []string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	modulesPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "app", "modules.go")

	parsed, err := parser.ParseFile(token.NewFileSet(), modulesPath, nil, 0)
	require.NoError(t, err)

	var selectors []string
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "orderBeginBlockers" {
			continue
		}

		ast.Inspect(function.Body, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "ModuleName" {
				return true
			}

			identifier, ok := selector.X.(*ast.Ident)
			require.True(t, ok)
			selectors = append(selectors, identifier.Name)
			return true
		})
	}

	require.NotEmpty(t, selectors)
	return selectors
}

func moduleIndex(t *testing.T, modules []string, target string) int {
	t.Helper()
	for index, module := range modules {
		if module == target {
			return index
		}
	}
	require.FailNow(t, "module is missing from BeginBlock order", target)
	return -1
}

func withoutModule(modules []string, target string) []string {
	result := make([]string, 0, len(modules))
	for _, module := range modules {
		if module != target {
			result = append(result, module)
		}
	}
	return result
}

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/open-policy-agent/golang-opa-wasm/opa"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/test/cases"
)

func TestE2E(t *testing.T) {
	ctx := context.Background()
	for _, tc := range cases.MustLoad("../../opa/test/cases/testdata").Sorted().Cases {
		t.Run(tc.Filename, func(t *testing.T) {
			opts := []func(*rego.Rego){
				rego.Query(tc.Query),
			}
			for i := range tc.Modules {
				opts = append(opts, rego.Module(fmt.Sprintf("module-%d.rego", i), tc.Modules[i]))
			}
			cr, err := rego.New(opts...).Compile(ctx)
			if err != nil {
				t.Fatal(err)
			}
			o := opa.New().WithPolicyBytes(cr.Bytes)
			if tc.Data != nil {
				o = o.WithDataJSON(tc.Data)
			}
			o, err = o.Init()
			if err != nil {
				t.Fatal(err)
			}

			if tc.InputTerm != nil {
				t.Log("not implemented: non-json input values")
				return
			}

			result, err := o.Eval(ctx, tc.Input)
			assert(t, tc, result, err)
		})
	}
}

func assert(t *testing.T, tc cases.TestCase, result *opa.Result, err error) {
	t.Helper()
	if tc.WantDefined != nil {
		if err != nil {
			t.Fatal("unexpected error:", err)
		} else {
			assertDefined(t, defined(*tc.WantDefined), result)
		}
	} else if tc.WantResult != nil {
		if err != nil {
			t.Fatal("unexpected error:", err)
		} else {
			assertResultSet(t, *tc.WantResult, tc.SortBindings, result)
		}
	} else if tc.WantErrorCode != nil || tc.WantError != nil {
		if err == nil {
			t.Fatal("expected error")
		}
		t.Log("err:", err)
		// TODO: implement more specific error checking
	}
}

type defined bool

func (x defined) String() string {
	if x {
		return "defined"
	}
	return "undefined"
}

func assertDefined(t *testing.T, want defined, result *opa.Result) {
	t.Helper()
	got := defined(len(result.Result.([]interface{})) > 0)
	if got != want {
		t.Fatalf("expected %v but got %v", want, got)
	}
}

func assertResultSet(t *testing.T, want []map[string]interface{}, sortBindings bool, result *opa.Result) {
	t.Helper()
	a := toAST(want)
	b := toAST(result.Result)

	if sortBindings {
		if arr, ok := b.Value.(*ast.Array); !ok {
			t.Fatal("expected array value for sortable result")
		} else {
			b.Value = arr.Sorted()
		}
	}

	if !a.Equal(b) {
		t.Fatalf("expected %v but got %v", a, b)
	}
}

func toAST(a interface{}) *ast.Term {

	buf := bytes.NewBuffer(nil)

	if err := json.NewEncoder(buf).Encode(a); err != nil {
		panic(err)
	}

	return ast.MustParseTerm(buf.String())
}
